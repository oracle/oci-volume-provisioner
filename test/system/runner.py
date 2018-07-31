#!/usr/bin/env python

# Copyright (c) 2018 Oracle and/or its affiliates. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import atexit
import argparse
import datetime
import json
import os
import re
import select
from shutil import copyfile
import subprocess
import sys
import time
import uuid
import oci
import yaml

DEBUG_FILE = "runner.log"
TERRAFORM_CLUSTER = "terraform/cluster"
TERRAFORM_DIR = "terraform"
# Variable name correspond to the ones found in the terraform config file
TERRAFORM_AVAILABILITY_DOMAIN = "availability_domain"
TERRAFORM_VOLUME_OCID = "volume_ocid"
TEST_NAME = "volumeprovisionersystemtest"
TMP_OCICONFIG = "/tmp/ociconfig"
TMP_KUBECONFIG = "/tmp/kubeconfig.conf"
TMP_OCI_API_KEY_FILE = "/tmp/oci_api_key.pem"
REGION = "us-ashburn-1"
TIMEOUT = 600
WRITE_REPORT=True
REPORT_DIR_PATH="/tmp/results"
REPORT_FILE="done"
POD_CONTROLLER = "controller"
POD_VOLUME = "volume"

# On exit return 0 for success or any other integer for a failure.
# If write_report is true then write a completion file to the Sonabuoy plugin result file.
# The default location is: /tmp/results/done
def _finish_with_exit_code(exit_code, write_report=True, report_dir_path=REPORT_DIR_PATH, report_file=REPORT_FILE):
    print "finishing with exit code: " + str(exit_code)
    if write_report:
        if not os.path.exists(report_dir_path):
            os.makedirs(report_dir_path)
        if exit_code == 0:
            _debug_file("\nTest Suite Success\n")
        else:
            _debug_file("\nTest Suite Failed\n")
        time.sleep(3)
        copyfile(DEBUG_FILE, report_dir_path + "/" + DEBUG_FILE)
        with open(report_dir_path + "/" + report_file, "w+") as file: 
            file.write(str(report_dir_path + "/" + DEBUG_FILE))
    finish_canary_metrics()
    sys.exit(exit_code)          


def _check_env(check_oci):
    if check_oci:
        if "OCICONFIG" not in os.environ and "OCICONFIG_VAR" not in os.environ:
            _log("Error. Can't find either OCICONFIG or OCICONFIG_VAR in the environment.")
            _finish_with_exit_code(1)


def _create_key_files(check_oci):
    _log("Setting environment variables")
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
            except yaml.YAMLError as err:
                _log("Error. Failed to parse oci config file %s. Error: %s " % (oci_config_file, err))
                _finish_with_exit_code(1)


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

def _get_terraform_env():
    timestamp = datetime.datetime.now().strftime('%Y%m%d%H%M%S%f')
    return "TF_VAR_test_id=" + timestamp

def _terraform(action, cwd, terraform_env):
    (stdout, _, returncode) = _run_command(terraform_env + " terraform " + action, cwd)
    if returncode != 0:
        _log("Error running terraform")
        sys.exit(1)
    return stdout

def _kubectl(action, exit_on_error=True, display_errors=True, log_stdout=True):
    if "KUBECONFIG" not in os.environ and "KUBECONFIG_VAR" not in os.environ:
        (stdout, _, returncode) = _run_command("kubectl " + action, ".", display_errors)
    else:
        (stdout, _, returncode) = _run_command("KUBECONFIG=" + _get_kubeconfig() + " kubectl " + action, ".", display_errors)
    if exit_on_error and returncode != 0:
        _log("Error running kubectl")
        _finish_with_exit_code(1)
    if log_stdout:
        _log(stdout)
    return stdout

def _get_pod_infos(test_id, pod_type):
    '''Retrieve pod information from kube-system
    @param test_id: Test id to use to search for the pod to get infor for
    @type test_id: C{Str}
    @param pod_type: Pod type to search for
    @type pod_type: C{Str}
    @return: Tuple containing the name of the resource, its status and the 
    node it's running on
    @rtype: C{Tuple}'''
    _namespace = "-n kube-system" if pod_type == POD_VOLUME else ""
    stdout = _kubectl(_namespace + " get pods -o wide")
    infos = []
    for line in stdout.split("\n"):
        line_array = line.split()
        if len(line_array) > 0:
            name = line_array[0]
            if name.startswith('oci-volume-provisioner')and pod_type == POD_VOLUME:
                status = line_array[2]
                node = line_array[6]
                infos.append((name, status, node))
            if re.match(r"nginx-controller-" + test_id + ".*", line) and pod_type == POD_CONTROLLER:
                name = line_array[0]
                status = line_array[2]
                node = line_array[6]
                infos.append((name, status, node))
    return infos

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
        _finish_with_exit_code(1)  
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
            _finish_with_exit_code(1)


def _volume_exists(compartment_id, volume, state, backup=False):
    '''Verify whether the volume is available or not'''
    client = oci.core.blockstorage_client.BlockstorageClient(_oci_config())
    if backup:
        volumes= oci.pagination.list_call_get_all_results(client.list_volume_backups, compartment_id)
    else:
        volumes = oci.pagination.list_call_get_all_results(client.list_volumes, compartment_id)
    _log("Getting status for volume %s" % volume)
    for vol in _get_json_doc(str(volumes.data)):
        if vol['id'].endswith(volume) and vol['lifecycle_state'] == state:
            return True
    return False

def _create_backup(volume_ocid, test_id):
    '''Create volume backup on OCI from existing volume
    @param volume_ocid: Ocid of course volume
    @type volume_ocid: C{Str}
    @param test_id: Test id used to append to component name
    @type test_id: C{Str}
    @return: Tuple containing the backup id, compartment id and display name
    @rtype: C{Tuple}'''
    client = oci.core.blockstorage_client.BlockstorageClient(_oci_config())
    _backup_details = oci.core.models.CreateVolumeBackupDetails(volume_id=volume_ocid, 
                                                                display_name="backup_volume_system_test" + test_id)
    _response = client.create_volume_backup(_backup_details)
    _log("Response for creating backup for volume %s: %s" % (volume_ocid, _response.data))
    _res = _get_json_doc(str(_response.data))
    return _res['id'], _res['compartment_id'], _res['display_name']

def _delete_backup(backup_ocid):
    '''Delete volume backup from OCI
    @param backup_ocid: Ocid of backup volume to delete
    @type backup_ocid: C{Str}'''
    client = oci.core.blockstorage_client.BlockstorageClient(_oci_config())
    _response = client.delete_volume_backup(backup_ocid)
    _log("Response for deleting volume backup %s: %s" % (backup_ocid, _response.data))


def _create_volume_from_backup(backup_ocid, test_id, availability_domain, compartment_id):
    client = oci.core.blockstorage_client.BlockstorageClient(_oci_config())
    _volume_details = oci.core.models.CreateVolumeDetails(volume_backup_id=backup_ocid, 
                                                          display_name="restored_volume_system_test" + test_id,
                                                          availability_domain=availability_domain,
                                                          compartment_id=compartment_id)
    try:
        _response = client.create_volume(_volume_details)
        _log("Response for creating volume from backup %s: %s %s" % (_response.data, _get_json_doc(str(_response.data))['id'], compartment_id))
        return _get_json_doc(str(_response.data))['id']
    except Exception as exc:
        _log("Failed to create volume from backup %s" % exc)

def _wait_for_volume(compartment_id, volume, state, backup=False):
    num_polls = 0
    while not _volume_exists(compartment_id, volume, state, backup):
        _log("    waiting...")
        time.sleep(1)
        num_polls += 1
        if num_polls == TIMEOUT:
            return False
    return True

def _wait_for_volume_to_create(compartment_id, volume, backup=False):
    return _wait_for_volume(compartment_id, volume, 'AVAILABLE', backup)


def _wait_for_volume_to_delete(compartment_id, volume, backup=False):
    return _wait_for_volume(compartment_id, volume, 'TERMINATED', backup)


def _get_compartment_id(pod_name):
    '''Gets the oci compartment_id from the oci-volume-provisioner pod host.
    This is where oci volume resources will be created.'''
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
        _finish_with_exit_code(1)

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
    _finish_with_exit_code(1)


def _create_yaml(template, test_id, region=None, backup_id=None):
    '''Generate yaml based on the given template and fill in additional details
    @param template: Name of file to use as template
    @type template: C{Str}
    @param test_id: Used for tagging resources with test id
    @type test_id: C{Str}
    @param region: Used for selecting resources from specified region
    @type region: C{Str}
    @param backup_id: Backup id to create PVC from
    @type backup_id: C{Str}
    @return: Name of generated config file
    @rtype: C{Str}'''
    yaml_file = template + ".yaml"
    with open(template, "r") as sources:
        lines = sources.readlines()
    with open(yaml_file, "w") as sources:
        for line in lines:
            patched_line = line
            patched_line = re.sub('{{TEST_ID}}', test_id, patched_line)
            if region is not None:
                patched_line = re.sub('{{REGION}}', region, patched_line)
            if backup_id is not None:
                patched_line = re.sub('{{BACKUP_ID}}', backup_id, patched_line)
            sources.write(patched_line)
    return yaml_file


def _test_create_volume(compartment_id, claim_target, claim_volume_name, check_oci, test_id=None, 
                        availability_domain=None, verify_func=None):
    '''Test making a volume claim from a configuration file
    @param backup_ocid: Verify whether the volume created from a backup contains backup info
    @type backup_ocid: C{Str}'''
    _kubectl("create -f " + claim_target, exit_on_error=False)

    volume = _get_volume_and_wait(claim_volume_name)
    _log("Created volume with name: " + volume)

    if check_oci:
        _log("Querying the OCI api to make sure a volume with this name exists...")
        if not _wait_for_volume_to_create(compartment_id, volume):
            _log("Failed to find volume with name: " + volume)
            return False
        _log("Volume: " + volume + " is present and available")

    if verify_func:
        verify_func(test_id, availability_domain, volume)
   
    _log("Delete the volume claim")
    _kubectl("delete -f " + claim_target, exit_on_error=False)

    if check_oci:
        _log("Querying the OCI api to make sure a volume with this name now doesnt exist...")
        _wait_for_volume_to_delete(compartment_id, volume)
        if not _volume_exists(compartment_id, volume, 'TERMINATED'):
            _log("Volume with name: " + volume + " still exists")
            return False
        _log("Volume: " + volume + " has now been terminated")
    
    return True

def _patch_template_file(infile, outfile, volume_name, test_id, availability_domain):
    '''Generate yaml based on the given template and fill in additional details
    @param template: Name of file to use as template
    @type template: C{Str}
    @param test_id: Used for tagging resources with test id
    @type test_id: C{Str}
    @param availability_domain: Used for selecting resources from specified AD
    @type availability_domain: C{Str}
    @return: Name of generated config file
    @rtype: C{Str}'''
    with open(infile, "r") as sources:
        lines = sources.readlines()
    with open(outfile + "." + test_id, "w") as sources:
        for line in lines:
            patched_line = line
            if volume_name is not None:
                patched_line = re.sub('{{VOLUME_NAME}}', volume_name, patched_line)
            patched_line = re.sub('{{TEST_ID}}', test_id, patched_line)
            if availability_domain:
                availability_domain = availability_domain.replace(':', '-') # yaml config does not allow ':'
                patched_line = re.sub('{{AVAILABILITY_DOMAIN}}', availability_domain, patched_line)
            sources.write(patched_line)
    return outfile + "." + test_id

def _create_rc_yaml(using_oci, volume_name, test_id, availability_domain):
    '''Generate replication controller yaml file from provided templates'''
    if using_oci:
        return _patch_template_file( "replication-controller.yaml.template",
                                     "replication-controller.yaml",
                                     volume_name, test_id, availability_domain)
    else:
        return _patch_template_file( "replication-controller-with-volume-claim.yaml.template",
                                     "replication-controller-with-volume-claim.yaml",
                                     volume_name, test_id, availability_domain)

def _get_terraform_output_var(terraform_env, var_name):
    '''Retrieve variable value from terraform output from state file
    @param terraform_env: Terraform test id
    @type terraform_env: C{Str}
    @param var_name: Name of variable to retrieve from output
    @type var_name: C{Str}
    @return: Value of variable
    @rtype: C{Str}'''
    output = _terraform("output -json", TERRAFORM_DIR, terraform_env)
    jsn = json.loads(output)
    return jsn[var_name]["value"]

def _get_volume_name(terraform_env):
    '''Retrieve volume name from terraform status output
    @param terraform_env: Terraform test id
    @type terraform_env: C{Str}
    @return: Volume OCID
    @rtype: C{Str}'''
    _ocid = _get_terraform_output_var(terraform_env, TERRAFORM_VOLUME_OCID).split('.')
    return _ocid[len(_ocid)-1]

def _wait_for_pod_status(desired_status, test_id, pod_type):
    '''Wait until the pod gets to the desired status
    @param desired_status: Status to wait for
    @type desired_status: C{Str}
    @param test_id: Test_id used to retrieve components generated by this test
    @type test_id: C{Str}
    @param pod_type: Pod type to query
    @type pod_type: C{Str}
    @return: Tuple containing the name of the resource, its status and the 
    node it's running on
    @rtype: C{Tuple}'''
    infos = _get_pod_infos(test_id, pod_type)
    num_polls = 0
    while not any(i[1] == desired_status for i in infos):
        for i in infos:
            _log("    - pod: " + i[0] + ", status: " + i[1] + ", node: " + i[2])
        time.sleep(1)
        num_polls += 1
        if num_polls == TIMEOUT:
            for i in infos:
                _log("Error: Pod: " + i[0] + " " +
                     "failed to achieve status: " + desired_status + "." +
                     "Final status was: " + i[1])
            sys.exit(1)
        infos = _get_pod_infos(test_id, pod_type)
    for i in infos:
        if i[1] == desired_status:
            return (i[0], i[1], i[2])
    # Should never get here.
    return (None, None, None)

def _create_replication_controller(test_id, availability_domain, volume_name="default_volume"):
    '''Create replication controller and wait for it to start
    @param test_id: Test id used to append to component names
    @type test_id : C{Str}
    @param availability_domain: Availability domain to start rc in
    @type availability_domain: C{Str}
    @param volume_name: Volume name used by the replication controller
    @type volume_name: C{Str}
    @return: Tuple containing the name of the created rc and its config file
    @rtype: C{Tuple}'''
    _rc_config = _create_rc_yaml(True, volume_name, test_id, availability_domain)
    _log("Starting the replication controller (creates a single nginx pod).")
    _kubectl("delete -f " + _rc_config, exit_on_error=False, display_errors=False)
    _kubectl("create -f " + _rc_config)
    _log("Waiting for the pod to start.")
    _rc_name, _, _ = _wait_for_pod_status("Running", test_id, POD_CONTROLLER)
    return _rc_name, _rc_config

def _create_file_via_replication_controller(rc_name, file_name="hello.txt"):
    '''Create file via the replication controller
    @param rcName: Name of the replication controller to write data to
    @type rcName: C{Str}
    @param fileName: Name of file to create
    @type fileName: C{Str}'''
    _kubectl("exec " + rc_name + " -- touch /usr/share/nginx/html/" + file_name)

def _verify_file_existance_via_replication_controller(rc_name, file_name="hello.txt"):
    '''Verify whether file exists via the replication controller
    @param rcName: Name of the replication controller to verify
    @type rcName: C{Str}
    @param fileName: Name of file to create
    @type fileName: C{Str}'''
    _log("Does the new file exist?")
    stdout = _kubectl("exec " + rc_name + " -- ls /usr/share/nginx/html")
    if file_name not in stdout.split("\n"):
        _log("Error: Failed to find file %s in mounted volume" % file_name)
        sys.exit(1)
    _log("Yes it does!")

def  _setup_create_volume_from_backup(terraform_env, test_id):
    '''Setup environment for creating a volume from a backup device
    @param test_id: Test id used to append to component names
    @type test_id : C{Str}
    @return: OCID of generated backup
    @rtype: C{Str}'''
    _log("Creating test volume (using terraform)", as_banner=True)
    _terraform("init", TERRAFORM_DIR, terraform_env)
    _terraform("apply", TERRAFORM_DIR, terraform_env)
    _availability_domain = _get_terraform_output_var(terraform_env, TERRAFORM_AVAILABILITY_DOMAIN)
    _log(_terraform("output -json", TERRAFORM_DIR, terraform_env))
    # Create replication controller and write data to the generated volume
    _rc_name, _rc_config = _create_replication_controller(test_id, _availability_domain, volume_name=_get_volume_name(terraform_env))
    _create_file_via_replication_controller(_rc_name)
    _verify_file_existance_via_replication_controller(_rc_name)
    # Create backup from generated volume
    _backup_ocid, compartment_id, _volume_name = _create_backup(_get_terraform_output_var(terraform_env, TERRAFORM_VOLUME_OCID), test_id)
    if not _wait_for_volume_to_create(compartment_id, _backup_ocid, backup=True):
        _log("Failed to find backup with name: " + _volume_name)
    return _backup_ocid, _availability_domain

def _tear_down_create_volume_from_backup(terraform_env, backup_ocid):
    '''Tear down create volume from backup
    @param test_id: Test id used to append to component names
    @type test_id: C{Str}
    @param backup_ocid: OCID of backup from which the test volume was created
    @type backup_ocid: C{Str}'''
    def _destroy_test_volume_atexit():
        _log("Destroying test volume (using terraform)", as_banner=True)
        _terraform("destroy -force", TERRAFORM_DIR, terraform_env)
    atexit.register(_destroy_test_volume_atexit)
    _delete_backup(backup_ocid)

def _volume_from_backup_check(test_id, availability_domain, volume, file_name='hello.txt'):
    '''Verify whether the volume created from the backup is in a healthy state
    @param test_id: Test id to use for creating components
    @type test_id: C{Str}
    @param availability_domain: Availability domain to create resource in
    @type availability_domain: C{Str}
    @param volume: Name of volume to verify
    @type volume: C{Str}
    @param file_name: Name of file to do checks for
    @type file_name: C{Str}'''
    _ocid = volume.split('.')
    _ocid = _ocid[-1]
    _rc_name, _rc_config = _create_replication_controller(test_id, availability_domain, _ocid)
    _log("Does the file from the previous backup exist?")
    stdout = _kubectl("exec " + _rc_name + " -- ls /usr/share/nginx/html")
    if file_name not in stdout.split("\n"):
        _log("Error: Failed to find file %s in mounted volume" % file_name)
    _log("Deleting the replication controller (deletes the single nginx pod).")
    _kubectl("delete -f " + _rc_config)


# Canary Metrics **************************************************************
# 

CM_SIMPLE = "volume_provisioner_simple"
CM_EXT3 = "volume_provisioner_ext3"
CM_NO_AD = "volume_provisioner_no_ad"
CM_VOLUME_FROM_BACKUP = "volume_provisioner_volume_from_backup" 

def canary_metric_date():
   return datetime.datetime.today().strftime('%Y-%m-%d-%H%m%S')

def init_canary_metrics(check_oci):
    if "METRICS_FILE" in os.environ:
        _log("generating metrics file...")
        canary_metrics = {}
        canary_metrics["start_time"] = canary_metric_date()
        canary_metrics[CM_SIMPLE] = 0
        canary_metrics[CM_EXT3] = 0
        canary_metrics[CM_NO_AD] = 0
        if check_oci:
            canary_metrics[CM_VOLUME_FROM_BACKUP] = 0 
        with open(os.environ.get("METRICS_FILE"), 'w') as metrics_file:
            json.dump(canary_metrics, metrics_file, sort_keys=True, indent=4)

def update_canary_metric(name, result):
    if "METRICS_FILE" in os.environ:
        _log("updating metrics fle...")
        with open(os.environ.get("METRICS_FILE"), 'r') as metrics_file:
            canary_metrics = json.load(metrics_file)
            canary_metrics[name] = result
        with open(os.environ.get("METRICS_FILE"), 'w') as metrics_file:
            json.dump(canary_metrics, metrics_file, sort_keys=True, indent=4)

def finish_canary_metrics():
   update_canary_metric("end_time", canary_metric_date())


# Main ************************************************************************
# 

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
        pod_name, _, _ = _wait_for_pod_status("Running", test_id, POD_VOLUME)
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
        init_canary_metrics(args['check_oci']) 
        res = _test_create_volume(compartment_id,
                            _create_yaml("../../examples/example-claim.template", test_id, _get_region()),
                            "demooci-" + test_id, args['check_oci'])
        update_canary_metric(CM_SIMPLE, int(res))
        success = False if res == False else success

        _log("Running system test: Ext3 file system", as_banner=True)
        res = _test_create_volume(compartment_id,
                            _create_yaml("../../examples/example-claim-ext3.template", test_id, None),
                            "demooci-ext3-" + test_id, args['check_oci'])
        update_canary_metric(CM_EXT3, int(res))
        success = False if res == False else success 

        _log("Running system test: No AD specified", as_banner=True)
        res = _test_create_volume(compartment_id,
                            _create_yaml("../../examples/example-claim-no-AD.template", test_id, None),
                            "demooci-no-ad-" + test_id, args['check_oci'])
        update_canary_metric(CM_NO_AD, int(res))
        success = False if res == False else success

        if args['check_oci']: 
            _log("Running system test: Create volume from backup", as_banner=True)
            terraform_env = _get_terraform_env()
            _backup_ocid, _availability_domain = _setup_create_volume_from_backup(terraform_env, test_id)
            _claim_target = _create_yaml("../../examples/example-claim-from-backup.template", test_id, 
                                        region=_availability_domain.split(':')[1], backup_id=_backup_ocid)
            res = _test_create_volume(compartment_id, _claim_target,
                                "demooci-from-backup-" + test_id, args['check_oci'],
                                test_id=test_id, availability_domain=_availability_domain,
                                verify_func=_volume_from_backup_check)
            update_canary_metric(CM_VOLUME_FROM_BACKUP, int(res))
            success = False if res == False else success
            _tear_down_create_volume_from_backup(terraform_env, _backup_ocid)

    if not success:
        _finish_with_exit_code(1)
    else: 
        _finish_with_exit_code(0)

if __name__ == "__main__":
    _main()



