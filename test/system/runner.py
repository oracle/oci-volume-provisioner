#!/usr/bin/env python

import atexit
import argparse
import datetime
import json
import os
import re
import select
import subprocess
import sys
import time
import uuid
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


def _check_env(check_oci):
    if check_oci:
        if "OCICONFIG" not in os.environ and "OCICONFIG_VAR" not in os.environ:
            _log("Error. Can't find either OCICONFIG or OCICONFIG_VAR in the environment.")
            sys.exit(1)


def _create_key_files(check_oci):
    if "OCICONFIG_VAR" in os.environ:
        _run_command("echo \"$OCICONFIG_VAR\" | openssl enc -base64 -d -A > " + TMP_OCICONFIG, ".")
        _run_command("chmod 600 " + TMP_OCICONFIG, ".")
    if "KUBECONFIG_VAR" in os.environ:
        _run_command("echo \"$KUBECONFIG_VAR\" | openssl enc -base64 -d -A > " + TMP_KUBECONFIG, ".")

    if check_oci:
        oci_config_file = _get_oci_config_file()
        with open(oci_config_file, 'r') as stream:
            try:
                cnf = yaml.load(stream)
                with open(TMP_OCI_API_KEY_FILE, 'w') as stream:
                    stream.write(cnf['auth']['key'])
            except yaml.YAMLError:
                _log("Error. Failed to parse oci config file " + oci_config_file)
                sys.exit(1)


def _destroy_key_files(check_oci):
    if "OCICONFIG_VAR" in os.environ:
        os.remove(TMP_OCICONFIG)
    if "KUBECONFIG_VAR" in os.environ:
        os.remove(TMP_KUBECONFIG)
    if check_oci:
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
    if "KUBECONFIG" not in os.environ and "KUBECONFIG_VAR" not in os.environ:
        (stdout, _, returncode) = _run_command("kubectl " + action, ".", display_errors)
    else:
        (stdout, _, returncode) = _run_command("KUBECONFIG=" + _get_kubeconfig() + " kubectl " + action, ".", display_errors)
    if exit_on_error and returncode != 0:
        _log("Error running kubectl")
        sys.exit(1)
    if log_stdout:
        _log(stdout)
    return stdout


def _get_pod_infos():
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


def _wait_for_pod_status(desired_status):
    infos = _get_pod_infos()
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
        infos = _get_pod_infos()
    return (infos[0][0], infos[0][1], infos[0][2])


def _get_volume(volume_name):
    stdout = _kubectl("get PersistentVolumeClaim -o wide")
    for line in stdout.split("\n"):
        line_array = line.split()
        if len(line_array) >= 3:
            name = line_array[0]
            status = line_array[1]
            if name == volume_name and status == "Bound":
                return line_array[2]
    return None


def _get_volume_and_wait(volume_name):
    num_polls = 0
    volume = _get_volume(volume_name)
    while not volume:
        _log("    waiting...")
        time.sleep(1)
        num_polls += 1
        if num_polls == TIMEOUT:
            return False
        volume = _get_volume(volume_name)
    return volume


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


def _volume_exists(compartment_id, volume, state):
    client = oci.core.blockstorage_client.BlockstorageClient(_oci_config())
    volumes = client.list_volumes(compartment_id)
    for vol in _get_json_doc(str(volumes.data)):
        if vol['id'].endswith(volume) and vol['lifecycle_state'] == state:
            return True
    return False


def _wait_for_volume(compartment_id, volume, state):
    num_polls = 0
    while not _volume_exists(compartment_id, volume, state):
        _log("    waiting...")
        time.sleep(1)
        num_polls += 1
        if num_polls == TIMEOUT:
            return False
    return True

def _wait_for_volume_to_create(compartment_id, volume):
    return _wait_for_volume(compartment_id, volume, 'AVAILABLE')


def _wait_for_volume_to_delete(compartment_id, volume):
    return _wait_for_volume(compartment_id, volume, 'TERMINATED')


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


def _handle_args():
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('--setup',
                        help='Setup the provisioner on the cluster',
                        action='store_true',
                        default=False)
    parser.add_argument('--no-test',
                        help='Dont run the tests on the test cluster',
                        action='store_true',
                        default=False)
    parser.add_argument('--check-oci',
                        help='Check with OCI that the volumes have been created/destroyed (requires --setup)',
                        action='store_true',
                        default=False)
    parser.add_argument('--teardown',
                        help='Teardown the provisioner on the cluster',
                        action='store_true',
                        default=False)
    args = vars(parser.parse_args())

    if args['check_oci'] and not args['setup']:
        _log("If --check-oci is specified, then --setup also needs to be set.")
        sys.exit(1)

    return args


def _cleanup(exit_on_error=False, display_errors=True):
    _kubectl("delete -f ../../dist/oci-volume-provisioner.yaml",
             exit_on_error, display_errors)
    _kubectl("delete -f ../../dist/oci-volume-provisioner-rbac.yaml",
             exit_on_error, display_errors)
    _kubectl("delete -f ../../dist/storage-class.yaml",
             exit_on_error, display_errors)
    _kubectl("delete -f ../../dist/storage-class-ext3.yaml",
             exit_on_error, display_errors)
    _kubectl("-n kube-system delete secret oci-volume-provisioner",
             exit_on_error, display_errors)


def _get_region():
    nodes_json = _kubectl("get nodes -o json", log_stdout=False)
    nodes = json.loads(nodes_json)
    for node in nodes['items']:
        return node['metadata']['labels']['failure-domain.beta.kubernetes.io/zone']
    _log("Region lookup failed")
    sys.exit(1)


def _create_yaml(template, test_id, region):
    yaml_file = template + ".yaml"
    with open(template, "r") as sources:
        lines = sources.readlines()
    with open(yaml_file, "w") as sources:
        for line in lines:
            patched_line = line
            patched_line = re.sub('{{TEST_ID}}', test_id, patched_line)
            if region is not None:
                patched_line = re.sub('{{REGION}}', region, patched_line)
            sources.write(patched_line)
    return yaml_file


def _test_create_volume(compartment_id, claim_target, claim_volume_name, check_oci):
    _log("Creating the volume claim")
    _kubectl("create -f " + claim_target, exit_on_error=False)

    volume = _get_volume_and_wait(claim_volume_name)
    _log("Created volume with name: " + volume)

    if check_oci:
        _log("Querying the OCI api to make sure a volume with this name exists...")
        if not _wait_for_volume_to_create(compartment_id, volume):
            _log("Failed to find volume with name: " + volume)
            sys.exit(1)
        _log("Volume: " + volume + " is present and available")

    _log("Delete the volume claim")
    _kubectl("delete -f " + claim_target, exit_on_error=False)

    if check_oci:
        _log("Querying the OCI api to make sure a volume with this name now doesnt exist...")
        _wait_for_volume_to_delete(compartment_id, volume)
        if not _volume_exists(compartment_id, volume, 'TERMINATED'):
            _log("Volume with name: " + volume + " still exists")
            sys.exit(1)
        _log("Volume: " + volume + " has now been terminated")


def _main():
    _reset_debug_file()
    args = _handle_args()

    _check_env(args['check_oci'])
    _create_key_files(args['check_oci'])
    def _destroy_key_files_atexit():
        _destroy_key_files(args['check_oci'])
    atexit.register(_destroy_key_files_atexit)

    test_id = str(uuid.uuid4())[:8]

    success = True

    if args['setup']:
        # Cleanup in case any existing state exists in the cluster
        _cleanup(display_errors=False)
        _log("Setting up the volume provisioner", as_banner=True)
        _kubectl("-n kube-system create secret generic oci-volume-provisioner " + \
                 "--from-file=config.yaml=" + _get_oci_config_file(),
                 exit_on_error=False)
        _kubectl("create -f ../../dist/storage-class.yaml", exit_on_error=False)
        _kubectl("create -f ../../dist/storage-class-ext3.yaml", exit_on_error=False)
        _kubectl("create -f ../../dist/oci-volume-provisioner-rbac.yaml", exit_on_error=False)
        _kubectl("create -f ../../dist/oci-volume-provisioner.yaml", exit_on_error=False)
        pod_name, _, _ = _wait_for_pod_status("Running")
        compartment_id = _get_compartment_id(pod_name)
    else:
        compartment_id = None

    if args['teardown']:
        def _teardown_atexit():
            _log("Tearing down the volume provisioner", as_banner=True)
            _cleanup()
        atexit.register(_teardown_atexit)

    if not args['no_test']:
        _log("Running system test: Simple", as_banner=True)
        _test_create_volume(compartment_id,
                            _create_yaml("../../manifests/example-claim.template", test_id, _get_region()),
                            "demooci-" + test_id, args['check_oci'])

        _log("Running system test: Ext3 file system", as_banner=True)
        _test_create_volume(compartment_id,
                            _create_yaml("../../manifests/example-claim-ext3.template", test_id, None),
                            "demooci-ext3-" + test_id, args['check_oci'])

        _log("Running system test: No AD specified", as_banner=True)
        _test_create_volume(compartment_id,
                            _create_yaml("../../manifests/example-claim-no-AD.template", test_id, None),
                            "demooci-no-ad-" + test_id, args['check_oci'])

    if not success:
        sys.exit(1)


if __name__ == "__main__":
    _main()
