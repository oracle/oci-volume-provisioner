#!/usr/bin/env python

import argparse
import datetime
import json
import os
import select
import subprocess
import sys
import time
import oci
import yaml

DEBUG_FILE = "runner.log"
TERRAFORM_CLUSTER = "terraform/cluster"
TEST_NAME = "volumeprovisionersystemtest"
TMP_OCICONFIG = "/tmp/ociconfig"
TMP_KUBECONFIG = "/tmp/kubeconfig.conf"
TMP_OCI_API_KEY_FILE = "/tmp/oci_api_key.pem"
REGION = "us-phoenix-1"
TIMEOUT = 600


def _check_env():
    if "OCICONFIG" not in os.environ and "OCICONFIG_VAR" not in os.environ:
        _log("Error. Can't find either OCICONFIG or OCICONFIG_VAR in the environment.")
        sys.exit(1)
    if "KUBECONFIG" not in os.environ and "KUBECONFIG_VAR" not in os.environ:
        _log("Error. Can't find either KUBECONFIG or KUBECONFIG_VAR in the environment.")
        sys.exit(1)


def _create_key_files():
    if "OCICONFIG_VAR" in os.environ:
        _run_command("echo \"$OCICONFIG_VAR\" | openssl enc -base64 -d -A > " + TMP_OCICONFIG, ".")
        _run_command("chmod 600 " + TMP_OCICONFIG, ".")
    if "KUBECONFIG_VAR" in os.environ:
        _run_command("echo \"$KUBECONFIG_VAR\" | openssl enc -base64 -d -A > " + TMP_KUBECONFIG, ".")

    oci_config_file = _get_oci_config_file()
    with open(_get_oci_config_file(), 'r') as stream:
        try:
            cnf = yaml.load(stream)
            with open(TMP_OCI_API_KEY_FILE, 'w') as stream:
                stream.write(cnf['auth']['key'])
        except yaml.YAMLError:
            _log("Error. Failed to parse oci config file " + oci_config_file)
            sys.exit(1)


def _destroy_key_files():
    if "OCICONFIG_VAR" in os.environ:
        os.remove(TMP_OCICONFIG)
    if "KUBECONFIG_VAR" in os.environ:
        os.remove(TMP_KUBECONFIG)
    os.remove(TMP_OCI_API_KEY_FILE)


def _get_kubeconfig():
    return os.environ['KUBECONFIG'] if "KUBECONFIG" in os.environ else TMP_KUBECONFIG


def _get_oci_config_file():
    return os.environ['OCICONFIG'] if "OCICONFIG" in os.environ else TMP_OCICONFIG


def _get_oci_api_key_file():
    return TMP_OCI_API_KEY_FILE


def _banner(as_banner, bold):
    if as_banner:
        if bold:
            print "********************************************************"
        else:
            print "--------------------------------------------------------"


def _reset_debug_file():
    if os.path.exists(DEBUG_FILE):
        os.remove(DEBUG_FILE)


def _debug_file(string):
    with open(DEBUG_FILE, "a") as debug_file:
        debug_file.write(string)


def _log(string, as_banner=False, bold=False):
    _banner(as_banner, bold)
    print string
    _banner(as_banner, bold)


def _process_stream(stream, read_fds, global_buf, line_buf):
    char = stream.read(1)
    if char == '':
        read_fds.remove(stream)
    global_buf.append(char)
    line_buf.append(char)
    if char == '\n':
        _debug_file(''.join(line_buf))
        line_buf = []
    return line_buf


def _poll(stdout, stderr):
    stdoutbuf = []
    stdoutbuf_line = []
    stderrbuf = []
    stderrbuf_line = []
    read_fds = [stdout, stderr]
    x_fds = [stdout, stderr]
    while read_fds:
        rlist, _, _ = select.select(read_fds, [], x_fds)
        if rlist:
            for stream in rlist:
                if stream == stdout:
                    stdoutbuf_line = _process_stream(stream, read_fds, stdoutbuf, stdoutbuf_line)
                if stream == stderr:
                    stderrbuf_line = _process_stream(stream, read_fds, stderrbuf, stderrbuf_line)
    return (''.join(stdoutbuf), ''.join(stderrbuf))


def _run_command(cmd, cwd, display_errors=True):
    _log(cwd + ": " + cmd)
    process = subprocess.Popen(cmd,
                               stdout=subprocess.PIPE,
                               stderr=subprocess.PIPE,
                               shell=True, cwd=cwd)
    (stdout, stderr) = _poll(process.stdout, process.stderr)
    returncode = process.wait()
    if returncode != 0 and display_errors:
        _log("    stdout: " + stdout)
        _log("    stderr: " + stderr)
        _log("    result: " + str(returncode))
    return (stdout, stderr, returncode)


def _get_timestamp(test_id):
    return test_id if test_id is not None else datetime.datetime.now().strftime('%Y%m%d%H%M%S%f')


def _kubectl(action, exit_on_error=True, display_errors=True, log_stdout=True):
    (stdout, _, returncode) = _run_command("KUBECONFIG=" + _get_kubeconfig() + " kubectl " + action, ".", display_errors)
    if exit_on_error and returncode != 0:
        _log("Error running kubectl")
        sys.exit(1)
    if log_stdout:
        _log(stdout)
    return stdout


def _get_json_doc(response):
    decoder = json.JSONDecoder()
    try:
        doc = decoder.decode(response)
    except (ValueError, UnicodeError) as _:
        _log('Invalid JSON in response: %s' % str(response))
        sys.exit(1)
    return doc


def _oci_config():
    config = dict(oci.config.DEFAULT_CONFIG)
    oci_config_file = _get_oci_config_file()
    with open(oci_config_file, 'r') as stream:
        try:
            cnf = yaml.load(stream)
            config["user"] = cnf['auth']['user']
            config["tenancy"] = cnf['auth']['tenancy']
            config["fingerprint"] = cnf['auth']['fingerprint']
            config["key_file"] = TMP_OCI_API_KEY_FILE
            config["region"] = cnf['auth']['region']
            return config
        except yaml.YAMLError:
            _log("Error. Failed to parse oci config file " + oci_config_file)
            sys.exit(1)


def _get_compartment_id(pod_name):
    """
    Gets the oci compartment_id from the oci-volume-provisioner pod host.
    This is where oci volume resources will be created.
    """
    result = _kubectl("-n kube-system exec %s -- curl -s http://169.254.169.254/opc/v1/instance/" % pod_name,
                exit_on_error=False, log_stdout=False)
    result_json = _get_json_doc(str(result))
    compartment_id = result_json["compartmentId"]
    return compartment_id


def get_availability_domain(compartment_id, ad_region_key):
    """
    Take the specified availability domain region key e.g: 'PHX-AD-1' and 
    return the tenancy availability domain region key e.g: 'Uocm:PHX-AD-1'

    See: https://docs.us-phoenix-1.oraclecloud.com/Content/General/Concepts/regions.htm?Highlight=list%20availability%20domains
    """
    client = oci.identity.identity_client.IdentityClient(_oci_config())
    ads = client.list_availability_domains(compartment_id)
    for ad in _get_json_doc(str(ads.data)):
        if ad['name'].endswith(ad_region_key):
            return ad['name']
    _log("Failed to determine the required tenancy for '" + ad_region_key + "' in compartment: '" + compartment_id + "'.")
    sys.exit(1)


def deref(d, *keys):
    for k in keys:
        try:
            d = d[k]
        except KeyError:
            return None
    return d


def _extract_pvc_manifest_opts(pvc_manifest_file):
    with open(pvc_manifest_file, 'r') as stream:
        try:
            pvc_manifest = yaml.load(stream)
            pvc_name = deref(pvc_manifest, 'metadata', 'name')
            ad_region_key = deref(pvc_manifest, 'spec', 'selector', 'matchLabels', 'oci-availability-domain')
            return pvc_name, ad_region_key
        except yaml.YAMLError:
            _log("Error. Failed to parse pvc manifest file " + pvc_manifest_file)
            sys.exit(1)


def _get_oci_volume_provisioner_infos():
    stdout = _kubectl("-n kube-system get pods -o wide")
    infos = []
    for line in stdout.split("\n"):
        line_array = line.split()
        if len(line_array) > 0:
            name = line_array[0]
            if name.startswith('oci-volume-provisioner'):
                status = line_array[2]
                node = line_array[6]
                infos.append((name, status, node))
    return infos


def _wait_for_oci_volume_provisioner_status(desired_status):
    infos = _get_oci_volume_provisioner_infos()
    num_polls = 0
    while not any(i[1] == desired_status for i in infos):
        for i in infos:
            _log("    - pod: " + i[0] + ", status: " + i[1] + ", node: " + i[2])
        time.sleep(1)
        num_polls += 1
        if num_polls == TIMEOUT:
            for i in infos:
                _log("Error: Pod: " + i[0] + " " + \
                     "failed to achieve status: " + desired_status + "." + \
                     "Final status was: " + i[1])
            sys.exit(1)
        infos = _get_oci_volume_provisioner_infos()
    return (infos[0][0], infos[0][1], infos[0][2])


def get_bound_pvc(pvc_name):
    stdout = _kubectl("get PersistentVolumeClaim -o wide")
    for line in stdout.split("\n"):
        line_array = line.split()
        if len(line_array) >= 3:
            name = line_array[0]
            status = line_array[1]
            if name == pvc_name and status == "Bound":
                return line_array[2]
    return None


def _wait_for_bound_pvc(pvc_name):
    num_polls = 0
    pvc = get_bound_pvc(pvc_name)
    while not pvc:
        _log("    waiting...")
        time.sleep(1)
        num_polls += 1
        if num_polls == TIMEOUT:
            return False
        pvc = get_bound_pvc(pvc_name)
    return pvc


def _volume_exists(compartment_id, volume_id, state):
    client = oci.core.blockstorage_client.BlockstorageClient(_oci_config())
    volumes = client.list_volumes(compartment_id)
    for vol in _get_json_doc(str(volumes.data)):
        if vol['id'].endswith(volume_id) and vol['lifecycle_state'] == state:
            return True
    return False


def _wait_for_volume(compartment_id, volume_id, state):
    num_polls = 0
    while not _volume_exists(compartment_id, volume_id, state):
        _log("    waiting...")
        time.sleep(1)
        num_polls += 1
        if num_polls == TIMEOUT:
            return False
    return True


def _test_create_volume(compartment_id, pvc_manifest_file):
    pvc_name, _ = _extract_pvc_manifest_opts(pvc_manifest_file)

    _log("Creating the volume claim")
    _kubectl("create -f " + pvc_manifest_file, exit_on_error=False)

    volume_id = _wait_for_bound_pvc(pvc_name)
    _log("Created volume with ocid: " + volume_id)

    _log("Querying the OCI api to make sure a volume with this name exists...")
    if not _wait_for_volume(compartment_id, volume_id, 'AVAILABLE'):
        _log("Failed to find volume with ocid: " + volume_id)
        sys.exit(1)
    _log("Volume: " + volume_id + " is present and available")

    _log("Delete the volume claim")
    _kubectl("delete -f " + pvc_manifest_file, exit_on_error=False)

    _log("Querying the OCI api to make sure a volume with this name now doesnt exist...")
    if not _wait_for_volume(compartment_id, volume_id, 'TERMINATED'):
        _log("Failed to terminate volume with ocid: " + volume_id)
        sys.exit(1)
    _log("Volume: " + volume_id + " has now been terminated")


def _filesystem_exists(compartment_id, filesystem_id, availability_domain, state):
    client = oci.file_storage.file_storage_client.FileStorageClient(_oci_config())
    filesystems = client.list_file_systems(compartment_id, availability_domain)
    for fs in _get_json_doc(str(filesystems.data)):
        if fs['id'].endswith(filesystem_id) and fs['lifecycle_state'] == state:
            return True
    return False


def _wait_for_filesystem(compartment_id, filesystem_id, availability_domain, state):
    num_polls = 0
    while not _filesystem_exists(compartment_id, filesystem_id, availability_domain, state):
        _log("    waiting...")
        time.sleep(1)
        num_polls += 1
        if num_polls == TIMEOUT:
            return False
    return True


def _test_create_filesystem(compartment_id, pvc_manifest_file):
    pvc_name, ad_region_key = _extract_pvc_manifest_opts(pvc_manifest_file)
    availability_domain = get_availability_domain(compartment_id, ad_region_key)

    _log("Creating the filesystem claim")
    _kubectl("create -f " + pvc_manifest_file, exit_on_error=False)

    fs_ocid = _wait_for_bound_pvc(pvc_name)
    _log("Created filesystem with ocid: " + fs_ocid)

    _log("Querying the OCI api to make sure a filesystem with this name exists...")
    if not _wait_for_filesystem(compartment_id, fs_ocid, availability_domain, 'ACTIVE'):
        _log("Failed to find filesystem with ocid: " + fs_ocid)
        sys.exit(1)
    _log("Filesystem: " + fs_ocid + " is present and available")

    _log("Delete the volume claim")
    _kubectl("delete -f " + pvc_manifest_file, exit_on_error=False)

    _log("Querying the OCI api to make sure a filesystem with this name now doesnt exist...")
    if not _wait_for_filesystem(compartment_id, fs_ocid, availability_domain, 'DELETED'):
        _log("Failed to delete filesystem with ocid: " + fs_ocid)
        sys.exit(1)
    _log("Filesystem: " + fs_ocid + " has now been deleted")


def _handle_args():
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('--no-setup',
                        help='Dont setup the provisioner on the cluster',
                        action='store_true',
                        default=False)
    parser.add_argument('--no-test',
                        help='Dont run the tests on the test cluster',
                        action='store_true',
                        default=False)
    parser.add_argument('--no-teardown',
                        help='Dont teardown the provisioner on the cluster',
                        action='store_true',
                        default=False)
    return vars(parser.parse_args())


def _cleanup(exit_on_error=False, display_errors=True):
    _kubectl("delete -f ../../dist/oci-volume-provisioner.yaml",
             exit_on_error, display_errors)
    _kubectl("delete -f ../../dist/oci-volume-provisioner-rbac.yaml",
             exit_on_error, display_errors)
    _kubectl("delete -f ../../dist/storage-class.yaml",
             exit_on_error, display_errors)
    _kubectl("delete -f ../../dist/storage-class-ffsw.yaml",
             exit_on_error, display_errors)    
    _kubectl("delete -f ../../dist/storage-class-ext3.yaml",
             exit_on_error, display_errors)
    _kubectl("-n kube-system delete secret oci-volume-provisioner",
             exit_on_error, display_errors)
    _kubectl("-n kube-system delete secret wcr-docker-pull-secret",
             exit_on_error, display_errors)


def _main():
    _reset_debug_file()
    args = _handle_args()

    _check_env()
    _create_key_files()

    success = True

    # Cleanup in case any existing state exists in the cluster
    _cleanup(display_errors=False)

    if not args['no_setup']:
        _log("Setting up the volume provisioner", as_banner=True)
        if "DOCKER_REGISTRY_USERNAME" in os.environ and "DOCKER_REGISTRY_PASSWORD" in os.environ:
            _kubectl("-n kube-system create secret docker-registry wcr-docker-pull-secret " + \
                     "--docker-server=\"wcr.io\" " + \
                     "--docker-username=\"" + os.environ['DOCKER_REGISTRY_USERNAME'] +"\" " + \
                     "--docker-password=\"" + os.environ['DOCKER_REGISTRY_PASSWORD'] +"\" " + \
                     "--docker-email=\"k8s@oracle.com\"",
                     exit_on_error=False)
        _kubectl("-n kube-system create secret generic oci-volume-provisioner " + \
                 "--from-file=config.yaml=" + _get_oci_config_file(),
                 exit_on_error=False)
        _kubectl("create -f ../../dist/storage-class.yaml", exit_on_error=False)
        _kubectl("create -f ../../dist/storage-class-ffsw.yaml", exit_on_error=False)        
        _kubectl("create -f ../../dist/storage-class-ext3.yaml", exit_on_error=False)
        _kubectl("create -f ../../dist/oci-volume-provisioner-rbac.yaml", exit_on_error=False)
        _kubectl("create -f ../../dist/oci-volume-provisioner.yaml", exit_on_error=False)

    pod_name, _, _ = _wait_for_oci_volume_provisioner_status("Running")
    compartment_id = _get_compartment_id(pod_name)

    if not args['no_test']:
        _log("Running system test: Block volume - Default", as_banner=True)
        _test_create_volume(compartment_id, "../../manifests/example-claim.yaml")

        _log("Running system test: Block volume - Ext3 file system", as_banner=True)
        _test_create_volume(compartment_id, "../../manifests/example-claim-ext3.yaml")

        _log("Running system test: Block volume - No AD specified", as_banner=True)
        _test_create_volume(compartment_id, "../../manifests/example-claim-no-AD.yaml")
        
        # trjl - disabled until the new flex_volume_driver (with the fs stuff) is ready!
        # 
        # _log("Running system test: NFSv3 filesystem - Default", as_banner=True)
        # _test_create_filesystem(compartment_id, "../../manifests/example-claim-ffsw.yaml")
        
    if not args['no_teardown']:
        _log("Tearing down the volume provisioner", as_banner=True)
        _cleanup()

    _destroy_key_files()

    if not success:
        sys.exit(1)


if __name__ == "__main__":
    _main()
