#!/usr/bin/env python

import argparse
import datetime
import json
import os
import select
import subprocess
import sys
import tempfile
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
    if "DOCKER_REGISTRY_USERNAME" not in os.environ:
        _log("Error. Can't find DOCKER_REGISTRY_USERNAME in the environment.")
        sys.exit(1)
    if "DOCKER_REGISTRY_PASSWORD" not in os.environ:
        _log("Error. Can't find DOCKER_REGISTRY_PASSWORD in the environment.")
        sys.exit(1)
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


def _get_pod_infos():
    stdout = _kubectl("-n kube-system get pods -o wide")
    infos = []
    for line in stdout.split("\n"):
        line_array = line.split()
        if len(line_array) > 0:
            name = line_array[0]
            if name == "oci-volume-provisioner":
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


def _get_volume():
    stdout = _kubectl("get PersistentVolumeClaim -o wide")
    for line in stdout.split("\n"):
        line_array = line.split()
        if len(line_array) >= 3:
            name = line_array[0]
            status = line_array[1]
            if name == "demooci" and status == "Bound":
                return line_array[2]
    return None


def _get_volume_and_wait():
    num_polls = 0
    volume = _get_volume()
    while not volume:
        _log("    waiting...")
        time.sleep(1)
        num_polls += 1
        if num_polls == TIMEOUT:
            return False
        volume = _get_volume()
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


def _wait_for_volume(compartment_id, volume):
    num_polls = 0
    while not _volume_exists(compartment_id, volume, 'AVAILABLE'):
        _log("    waiting...")
        time.sleep(1)
        num_polls += 1
        if num_polls == TIMEOUT:
            return False
    return True

def _get_compartment_id():
    """
    Gets the oci compartment_id from the oci-volume-provisioner pod host.
    This is where oci volume resources will be created.
    """
    result = _kubectl("-n kube-system exec oci-volume-provisioner -- curl -s http://169.254.169.254/opc/v1/instance/",
                exit_on_error=False, log_stdout=False)
    result_json = _get_json_doc(str(result))
    compartment_id = result_json["compartmentId"]
    return compartment_id


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

def cleanup(exit_on_error=False, display_errors=True):
    _kubectl("delete -f ../../dist/oci-volume-provisioner.yaml", exit_on_error, display_errors)
    _kubectl("delete -f ../../manifests/oci-volume-provisioner-rbac.yaml", exit_on_error, display_errors)
    _kubectl("delete -f ../../manifests/storage-class.yaml", exit_on_error, display_errors)
    _kubectl("-n kube-system delete secret oci-volume-provisioner", exit_on_error, display_errors)
    _kubectl("-n kube-system delete secret wcr-docker-pull-secret", exit_on_error, display_errors)

def _main():
    _reset_debug_file()
    args = _handle_args()

    _check_env()
    _create_key_files()

    success = True

    # Cleanup in case any existing state exists in the cluster
    cleanup(display_errors=False)

    if not args['no_setup']:
        _log("Setting up the volume provisioner", as_banner=True)
        _kubectl("-n kube-system create secret docker-registry wcr-docker-pull-secret " + \
                 "--docker-server=\"wcr.io\" " + \
                 "--docker-username=\"" + os.environ['DOCKER_REGISTRY_USERNAME'] +"\" " + \
                 "--docker-password=\"" + os.environ['DOCKER_REGISTRY_PASSWORD'] +"\" " + \
                 "--docker-email=\"k8s@oracle.com\"",
                 exit_on_error=False)
        _kubectl("-n kube-system create secret generic oci-volume-provisioner " + \
                 "--from-file=config.yaml=" + _get_oci_config_file(),
                 exit_on_error=False)

        _kubectl("create -f ../../manifests/storage-class.yaml", exit_on_error=False)
        _kubectl("create -f ../../manifests/oci-volume-provisioner-rbac.yaml", exit_on_error=False)
        _kubectl("create -f ../../dist/oci-volume-provisioner.yaml", exit_on_error=False)

        _wait_for_pod_status("Running")

    # get the compartment_id of the oci-volume-provisioner
    compartment_id = _get_compartment_id()

    if not args['no_test']:
        _log("Running system test: ", as_banner=True)

        _log("Creating the volume claim")
        _kubectl("create -f ../../manifests/example-claim.yaml",
                 exit_on_error=False)

        volume = _get_volume_and_wait()
        _log("Created volume with name: " + volume)

        _log("Querying the OCI api to make sure a volume with this name exists...")
        if not _wait_for_volume(compartment_id, volume):
            _log("Failed to find volume with name: " + volume)
            sys.exit(1)
        _log("Volume: " + volume + " is present and available")

        _log("Delete the volume claim")
        _kubectl("delete -f ../../manifests/example-claim.yaml", exit_on_error=False)

        _log("Querying the OCI api to make sure a volume with this name now doesnt exist...")
        if not _volume_exists(compartment_id, volume, 'TERMINATED'):
            _log("Volume with name: " + volume + " still exists")
            sys.exit(1)
        _log("Volume: " + volume + " has now been terminated")

    if not args['no_teardown']:
        _log("Tearing down the volume provisioner", as_banner=True)
        cleanup()

    _destroy_key_files()

    if not success:
        sys.exit(1)


if __name__ == "__main__":
    _main()
