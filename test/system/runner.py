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

DEBUG_FILE = "runner.log"
TERRAFORM_CLUSTER = "terraform/cluster"
TEST_NAME = "volumeprovisionersystemtest"
TMP_OCI_API_KEY = "/tmp/oci_api_key.pem"
TMP_KUBECONFIG = "/tmp/kubeconfig.conf"
USER_OCID = "ocid1.user.oc1..aaaaaaaao235lbcxvdrrqlrpwv4qvil2xzs4544h3lof4go3wz2ett6arpeq"
TENNANCY_OCID = "ocid1.tenancy.oc1..aaaaaaaatyn7scrtwtqedvgrxgr2xunzeo6uanvyhzxqblctwkrpisvke4kq"
FINGERPRINT = "4d:f5:ff:0e:a9:10:e8:5a:d3:52:6a:f8:1e:99:a3:47"
COMPARTMENT_ID = "ocid1.compartment.oc1..aaaaaaaa6yrzvtwcumheirxtmbrbrya5lqkr7k7lxi34q3egeseqwlq2l5aq"
REGION = "us-phoenix-1"
TIMEOUT = 600


def _check_env():
    if "DOCKER_REGISTRY_USERNAME" not in os.environ:
        _log("Error. Can't find DOCKER_REGISTRY_USERNAME in the environment.")
        sys.exit(1)
    if "DOCKER_REGISTRY_PASSWORD" not in os.environ:
        _log("Error. Can't find DOCKER_REGISTRY_PASSWORD in the environment.")
        sys.exit(1)
    if "OCI_API_KEY" not in os.environ and "OCI_API_KEY_VAR" not in os.environ:
        _log("Error. Can't find either OCI_API_KEY or OCI_API_KEY_VAR in the environment.")
        sys.exit(1)
    if "KUBECONFIG" not in os.environ and "KUBECONFIG_VAR" not in os.environ:
        _log("Error. Can't find either KUBECONFIG or KUBECONFIG_VAR in the environment.")
        sys.exit(1)


def _create_key_files():
    if "OCI_API_KEY_VAR" in os.environ:
        _run_command("echo \"$OCI_API_KEY_VAR\" > " + TMP_OCI_API_KEY, ".")
        _run_command("chmod 600 " + TMP_OCI_API_KEY, ".", verbose=False)
    if "KUBECONFIG_VAR" in os.environ:
        _run_command("echo \"$KUBECONFIG_VAR\" > " + TMP_KUBECONFIG, ".")


def _destroy_key_files():
    if "OCI_API_KEY_VAR" in os.environ:
        os.remove(TMP_OCI_API_KEY)
    if "KUBECONFIG_VAR" in os.environ:
        os.remove(TMP_KUBECONFIG)


def _get_kubeconfig():
    return os.environ['KUBECONFIG'] if "KUBECONFIG" in os.environ else TMP_KUBECONFIG


def _get_oci_api_key():
    return os.environ['OCI_API_KEY'] if "OCI_API_KEY" in os.environ else TMP_OCI_API_KEY


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


def _run_command(cmd, cwd, verbose=True):
    if verbose:
        _log(cwd + ": " + cmd)
    process = subprocess.Popen(cmd,
                               stdout=subprocess.PIPE,
                               stderr=subprocess.PIPE,
                               shell=True, cwd=cwd)
    (stdout, stderr) = _poll(process.stdout, process.stderr)
    returncode = process.wait()
    if returncode != 0:
        _log("    stdout: " + stdout)
        _log("    stderr: " + stderr)
        _log("    result: " + str(returncode))
    return (stdout, stderr, returncode)


def _get_timestamp(test_id):
    return test_id if test_id is not None else datetime.datetime.now().strftime('%Y%m%d%H%M%S%f')


def _get_kubectl_env():
    return "KUBECONFIG=" + _get_kubeconfig()


def _kubectl(action, kubectl_env, exit_on_error=True):
    (stdout, _, returncode) = _run_command(kubectl_env + " kubectl " + action, ".")
    if exit_on_error and returncode != 0:
        _log("Error running kubectl")
        sys.exit(1)
    _log(stdout)
    return stdout


def _get_pod_infos(kubectl_env):
    stdout = _kubectl("-n kube-system get pods -o wide", kubectl_env)
    infos = []
    for line in stdout.split("\n"):
        line_array = line.split()
        if len(line_array) > 0:
            name = line_array[0]
            if name == "oci-provisioner":
                status = line_array[2]
                node = line_array[6]
                infos.append((name, status, node))
    return infos


def _wait_for_pod_status(kubectl_env, desired_status):
    infos = _get_pod_infos(kubectl_env)
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
        infos = _get_pod_infos(kubectl_env)
    return (infos[0][0], infos[0][1], infos[0][2])


def _get_volume(kubectl_env):
    stdout = _kubectl("get PersistentVolumeClaim -o wide", kubectl_env)
    for line in stdout.split("\n"):
        line_array = line.split()
        if len(line_array) >= 3:
            name = line_array[0]
            status = line_array[1]
            if name == "demooci" and status == "Bound":
                return line_array[2]
    return None


def _get_volume_and_wait(kubectl_env):
    num_polls = 0
    volume = _get_volume(kubectl_env)
    while not volume:
        _log("    waiting...")
        time.sleep(1)
        num_polls += 1
        if num_polls == TIMEOUT:
            return False
        volume = _get_volume(kubectl_env)
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
    config["user"] = USER_OCID
    config["tenancy"] = TENNANCY_OCID
    config["fingerprint"] = FINGERPRINT
    config["key_file"] = _get_oci_api_key()
    config["region"] = REGION
    return config


def _volume_exists(volume, state):
    client = oci.core.blockstorage_client.BlockstorageClient(_oci_config())
    volumes = client.list_volumes(COMPARTMENT_ID)
    for vol in _get_json_doc(str(volumes.data)):
        if vol['id'].endswith(volume) and vol['lifecycle_state'] == state:
            return True
    return False


def _wait_for_volume(volume):
    num_polls = 0
    while not _volume_exists(volume, 'AVAILABLE'):
        _log("    waiting...")
        time.sleep(1)
        num_polls += 1
        if num_polls == TIMEOUT:
            return False
    return True


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


def _main():
    _reset_debug_file()
    args = _handle_args()

    _check_env()
    _create_key_files()

    kubectl_env = _get_kubectl_env()
    success = True

    if not args['no_setup']:
        _log("Setting up the volume provisioner", as_banner=True)

        _kubectl("create -f ../../manifests/auth/serviceaccount.yaml",
                 kubectl_env, exit_on_error=False)
        _kubectl("create -f../../manifests/auth/clusterrole.yaml",
                 kubectl_env, exit_on_error=False)
        _kubectl("create -f ../../manifests/auth/clusterrolebinding.yaml",
                 kubectl_env, exit_on_error=False)
        _run_command(kubectl_env + " ../../scripts/generate-oci-configmap.sh " + \
                                   USER_OCID + " " + \
                                   FINGERPRINT + " " + \
                                   TENNANCY_OCID, ".", verbose=False)
        _run_command(kubectl_env + " ../../scripts/generate-oci-secret.sh " + \
                                   _get_oci_api_key(), ".", verbose=False)
        _run_command(kubectl_env + " ../../scripts/generate-docker-registry-secret.sh \'" + \
                                   os.environ['DOCKER_REGISTRY_USERNAME'] + "\' \'" + \
                                   os.environ['DOCKER_REGISTRY_PASSWORD'] + "\' " + \
                                   "test-user@oracle.com", ".", verbose=False)
        _kubectl("create -f ../../dist/oci-volume-provisioner.yaml",
                 kubectl_env, exit_on_error=False)
        _kubectl("create -f ../../manifests/storage-class.yaml",
                 kubectl_env, exit_on_error=False)

        _wait_for_pod_status(kubectl_env, "Running")

    if not args['no_test']:
        _log("Running system test: ", as_banner=True)

        _log("Creating the volume claim")
        _kubectl("create -f ../../manifests/example-claim.yaml",
                 kubectl_env, exit_on_error=False)

        volume = _get_volume_and_wait(kubectl_env)
        _log("Created volume with name: " + volume)

        _log("Querying the OCI api to make sure a volume with this name exists...")
        if not _wait_for_volume(volume):
            _log("Failed to find volume with name: " + volume)
            sys.exit(1)
        _log("Volume: " + volume + " is present and available")

        _log("Delete the volume claim")
        _kubectl("delete -f ../../manifests/example-claim.yaml",
                 kubectl_env, exit_on_error=False)

        _log("Querying the OCI api to make sure a volume with this name now doesnt exist...")
        if not _volume_exists(volume, 'TERMINATED'):
            _log("Volume with name: " + volume + " still exists")
            sys.exit(1)
        _log("Volume: " + volume + " has now been terminated")

    if not args['no_teardown']:
        _log("Tearing down the volume provisioner", as_banner=True)

        _kubectl("delete -f ../../manifests/storage-class.yaml",
                 kubectl_env, exit_on_error=False)
        _kubectl("delete -f ../../dist/oci-volume-provisioner.yaml",
                 kubectl_env, exit_on_error=False)
        _kubectl("-n kube-system delete secret odx-docker-pull-secret",
                 kubectl_env, exit_on_error=False)
        _kubectl("-n kube-system delete secret ocisapikey",
                 kubectl_env, exit_on_error=False)
        _kubectl("-n kube-system delete configmap oci-volume-provisioner",
                 kubectl_env, exit_on_error=False)
        _kubectl("delete -f ../../manifests/auth/clusterrolebinding.yaml",
                 kubectl_env, exit_on_error=False)
        _kubectl("delete -f../../manifests/auth/clusterrole.yaml",
                 kubectl_env, exit_on_error=False)
        _kubectl("delete -f ../../manifests/auth/serviceaccount.yaml",
                 kubectl_env, exit_on_error=False)

    _destroy_key_files()

    if not success:
        sys.exit(1)


if __name__ == "__main__":
    _main()
