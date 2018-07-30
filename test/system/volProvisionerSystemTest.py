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

import utils
import time
import oci
import yaml
import os
import json
import datetime
import sys
import re
from yamlUtils import PopulateYaml


class VolumeProvisionerSystemTestInterface(object):

    TERRAFORM_CLUSTER = "terraform/cluster"
    TERRAFORM_DIR = "terraform"
    # Variable name correspond to the ones found in the terraform config file
    TERRAFORM_AVAILABILITY_DOMAIN = "availability_domain"
    TERRAFORM_VOLUME_OCID = "volume_ocid"
    POD_CONTROLLER = "controller"
    POD_VOLUME = "volume"    
    BLOCK_STORAGE = "block"
    FS_STORAGE = "fileSystem"
    TIMEOUT = 600
    BOUND = "Bound"
    TMP_OCI_API_KEY_FILE = "/tmp/oci_api_key.pem"
    TMP_OCICONFIG = "/tmp/ociconfig"
    LIFECYCLE_STATE_ON = {BLOCK_STORAGE: 'AVAILABLE',
                          FS_STORAGE: 'ACTIVE'}
    LIFECYCLE_STATE_OFF = {BLOCK_STORAGE: 'TERMINATED',
                           FS_STORAGE:'DELETED'}
    DEFAULT_AVAILABILITY_DOMAIN="NWuj:PHX-AD-2" 
  
    def __init__(self, test_id=None, setup=False, compartment_id=None, check_oci=False, k8Resources=None):
        '''@param test_id: Id of currently running test
        @type test_id: C{Str}
        @param setup: Flag that indicated whether the provisioner needs to be setup on the cluster
        @type setup: C{Bool}
        @param compartment_id: Compartment Id to use to creaate/delete resources
        @type compartment_id: C{Str}
        @param check_oci: 'Check with OCI that the volumes have been created/destroyed (requires --setup)'
        @type check_oci: C{Bool}
        '''
        self._test_id = test_id if test_id else "demotest"
        self._setup = setup
        self._compartment_id = compartment_id
        self._region = self._get_region()
        self._check_oci = check_oci
        self._oci_config = self._get_oci_config()
        self._terraform_env = self._get_terraform_env()
        self._k8sResources = k8Resources if k8Resources else []

    def run(self):
        if self._setup:
            # Cleanup in case any existing state exists in the cluster
            self.cleanup(display_errors=False)
            utils.log("Setting up the volume provisioner", as_banner=True)
            utils.kubectl("-n kube-system create secret generic oci-volume-provisioner " + \
                          "--from-file=config.yaml=" + self._get_oci_config_file(),
                          exit_on_error=False)
            for _res in self._k8sResources:
                utils.kubectl("create -f " + _res, exit_on_error=False)
            pod_name, _, _ = self._wait_for_pod_status("Running", self.POD_VOLUME)
            self._compartment_id = self._get_compartment_id(pod_name)
        

    def cleanup(self, exit_on_error=False, display_errors=True):
        for _res in self._k8sResources: 
            utils.kubectl("delete -f " + _res, exit_on_error, display_errors)

    @staticmethod
    def _get_region():
        nodes_json = utils.kubectl("get nodes -o json", log_stdout=False)
        nodes = json.loads(nodes_json)
        for node in nodes['items']:
            return node['metadata']['labels']['failure-domain.beta.kubernetes.io/zone']
        utils.log("Region lookup failed")
        utils.finish_with_exit_code(1)

    def _get_oci_config_file(self):
        return os.environ['OCICONFIG'] if "OCICONFIG" in os.environ else self.TMP_OCICONFIG

    def _get_oci_config(self):
        config = dict(oci.config.DEFAULT_CONFIG)
        oci_config_file = self._get_oci_config_file()
        with open(oci_config_file, 'r') as stream:
            try:
                cnf = yaml.load(stream)
                config["user"] = cnf['auth']['user']
                config["tenancy"] = cnf['auth']['tenancy']
                config["fingerprint"] = cnf['auth']['fingerprint']
                config["key_file"] = self.TMP_OCI_API_KEY_FILE
                config["region"] = cnf['auth']['region']
                return config
            except yaml.YAMLError:
                utils.log("Error. Failed to parse oci config file " + oci_config_file)
                utils.finish_with_exit_code(1)

    def _get_terraform_env(self):
        timestamp = datetime.datetime.now().strftime('%Y%m%d%H%M%S%f')
        return "TF_VAR_test_id=" + timestamp

    def _terraform(self, action, cwd):
        '''Execute terraform command'''
        (stdout, _, returncode) = utils.run_command(self._terraform_env + " terraform " + action, cwd)
        if returncode != 0:
            utils.log("Error running terraform")
            sys.exit(1)
        return stdout

    def _get_terraform_output_var(self, var_name):
        '''Retrieve variable value from terraform output from state file
        @param var_name: Name of variable to retrieve from output
        @type var_name: C{Str}
        @return: Value of variable
        @rtype: C{Str}'''
        output = self._terraform("output -json", self.TERRAFORM_DIR,)
        jsn = json.loads(output)
        return jsn[var_name]["value"]

    def _get_volume_name(self):
        '''Retrieve volume name from terraform status output
        @param terraform_env: Terraform test id
        @type terraform_env: C{Str}
        @return: Volume OCID
        @rtype: C{Str}'''
        _ocid = self._get_terraform_output_var(self.TERRAFORM_VOLUME_OCID).split('.')
        return _ocid[len(_ocid)-1]

    def _get_volume(self, volume_name):
        stdout = utils.kubectl("get PersistentVolumeClaim -o wide")
        for line in stdout.split("\n"):
            line_array = line.split()
            if len(line_array) >= 3:
                name = line_array[0]
                status = line_array[1]
                if name == volume_name and status == self.BOUND:
                    return line_array[2]

    def _get_volume_and_wait(self, volume_name):
        num_polls = 0
        volume = self._get_volume(volume_name)
        while not volume:
            utils.log("    waiting...")
            time.sleep(1)
            num_polls += 1
            if num_polls == self.TIMEOUT:
                return False
            volume = self._get_volume(volume_name)
        return volume

    @staticmethod
    def _get_json_doc(response):
        decoder = json.JSONDecoder()
        try:
            doc = decoder.decode(response)
        except (ValueError, UnicodeError) as _:
            utils.log('Invalid JSON in response: %s' % str(response))
            utils.finish_with_exit_code(1)  
        return doc

    def _volume_exists(self, volume, state, compartment_id=None, backup=False, storageType=BLOCK_STORAGE, availability_domain=None):
        '''Verify whether the volume is available or not
        @param storageType: Storage type to search for volumes in
        @type storageType: C{Str}
        @param availability_domain: Availability domain to look in for
        @type availability_domain: C{Str}'''
        if storageType == self.BLOCK_STORAGE:
            utils.log("Retrieving block volumes")
            client = oci.core.blockstorage_client.BlockstorageClient(self._oci_config)
            if backup:
                volumes= oci.pagination.list_call_get_all_results(client.list_volume_backups, compartment_id)
            else:
                volumes = oci.pagination.list_call_get_all_results(client.list_volumes, compartment_id)
        else:
            utils.log("Retrieving file systems")
            client = oci.file_storage.FileStorageClient(self._oci_config)
            volumes = oci.pagination.list_call_get_all_results(client.list_file_systems, compartment_id,
                                                               availability_domain)
        utils.log("Getting status for volume %s" % volume)
        for vol in self._get_json_doc(str(volumes.data)):
            if vol['id'].endswith(volume) and vol['lifecycle_state'] == state:
                return True
        return False

    def _wait_for_volume(self, volume, state, compartment_id=None, backup=False, storageType=BLOCK_STORAGE, availability_domain=None):
        num_polls = 0
        while not self._volume_exists(volume, state,  compartment_id=compartment_id, backup=backup, storageType=storageType,
                                      availability_domain=availability_domain,):
            utils.log("    waiting...")
            time.sleep(1)
            num_polls += 1
            if num_polls == self.TIMEOUT:
                return False
        return True

    def _wait_for_volume_to_create(self, volume, compartment_id=None, backup=False, storageType=BLOCK_STORAGE, availability_domain=None):
        compartment_id = compartment_id if compartment_id else self._compartment_id
        return self._wait_for_volume(volume, self.LIFECYCLE_STATE_ON[storageType], backup, storageType=storageType, 
                                availability_domain=availability_domain)


    def _wait_for_volume_to_delete(self, volume, compartment_id=None, backup=False, storageType=BLOCK_STORAGE, availability_domain=None):
        compartment_id = compartment_id if compartment_id else self._compartment_id
        return self._wait_for_volume(volume, self.LIFECYCLE_STATE_OFF[storageType], backup, storageType=storageType,
                                     availability_domain=availability_domain)

    def _test_create_volume(self, claim_target, claim_volume_name, availability_domain=None, verify_func=None, storageType=BLOCK_STORAGE):
        '''Test making a volume claim from a configuration file
        @param backup_ocid: Verify whether the volume created from a backup contains backup info
        @type backup_ocid: C{Str}'''  
        utils.kubectl("create -f " + claim_target, exit_on_error=False)

        volume = self._get_volume_and_wait(claim_volume_name)
        utils.log("Created volume with name: %s" % str(volume))

        if self._check_oci:
            utils.log("Querying the OCI api to make sure a volume with this name exists...")
            if not self._wait_for_volume_to_create(volume, storageType=storageType, 
                                                    availability_domain=availability_domain):
                utils.log("Failed to find volume with name: " + volume)
                utils.finish_with_exit_code(1)
            utils.log("Volume: " + volume + " is present and available")

        if verify_func:
            verify_func(self._test_id, availability_domain, volume)
    
        utils.log("Delete the volume claim")
        utils.kubectl("delete -f " + claim_target, exit_on_error=False)

        if self._check_oci:
            utils.log("Querying the OCI api to make sure a volume with this name now doesnt exist...")
            self._wait_for_volume_to_delete(volume, storageType=storageType,
                                    availability_domain=availability_domain)
            if not self._volume_exists(volume, self.LIFECYCLE_STATE_OFF[storageType], storageType=storageType,
                                availability_domain=availability_domain):
                utils.log("Volume with name: " + volume + " still exists")
                utils.finish_with_exit_code(1)
            utils.log("Volume: " + volume + " has now been terminated")
  
    def _create_rc_or_pod(self, config, availability_domain, volume_name="default_volume"):
        '''Create replication controller or pod and wait for it to start
        @param rc_config: Replication controller configuration file to patch
        @type rc_config: C{Str}
        @param availability_domain: Availability domain to start rc in
        @type availability_domain: C{Str}
        @param volume_name: Volume name used by the replication controller
        @type volume_name: C{Str}
        @return: Tuple containing the name of the created rc and its config file
        @rtype: C{Tuple}'''
        _config = PopulateYaml(config, self._test_id, volume_name=volume_name, availability_domain=availability_domain).generateFile()
        utils.log("Starting the replication controller (creates a single nginx pod).")
        utils.kubectl("delete -f " + _config, exit_on_error=False, display_errors=False)
        utils.kubectl("create -f " + _config)
        utils.log("Waiting for the pod to start.")
        _name, _, _ = self._wait_for_pod_status("Running", self.POD_CONTROLLER)
        return _name, _config

    def _wait_for_pod_status(self, desired_status, pod_type):
        '''Wait until the pod gets to the desired status
        @param desired_status: Status to wait for
        @type desired_status: C{Str}
        @param pod_type: Pod type to query
        @type pod_type: C{Str}
        @return: Tuple containing the name of the resource, its status and the 
        node it's running on
        @rtype: C{Tuple}'''
        infos = self._get_pod_infos(pod_type)
        num_polls = 0
        while not any(i[1] == desired_status for i in infos):
            for i in infos:
                utils.log("    - pod: " + i[0] + ", status: " + i[1] + ", node: " + i[2])
            time.sleep(1)
            num_polls += 1
            if num_polls == self.TIMEOUT:
                for i in infos:
                    utils.log("Error: Pod: " + i[0] + " " +
                        "failed to achieve status: " + desired_status + "." +
                        "Final status was: " + i[1])
                sys.exit(1)
            infos = self._get_pod_infos(pod_type)
        for i in infos:
            if i[1] == desired_status:
                return (i[0], i[1], i[2])
        # Should never get here.
        return (None, None, None)

    def _get_pod_infos(self, pod_type):
        '''Retrieve pod information from kube-system
        @param pod_type: Pod type to search for
        @type pod_type: C{Str}
        @return: Tuple containing the name of the resource, its status and the 
        node it's running on
        @rtype: C{Tuple}'''
        _namespace = "-n kube-system" if pod_type == self.POD_VOLUME else ""
        stdout = utils.kubectl(_namespace + " get pods -o wide")
        infos = []
        for line in stdout.split("\n"):
            line_array = line.split()
            if len(line_array) > 0:
                name = line_array[0]
                if name.startswith('oci-volume-provisioner') and pod_type == self.POD_VOLUME:
                    status = line_array[2]
                    node = line_array[6]
                    infos.append((name, status, node))
                if re.match(r"nginx-controller-" + self._test_id + ".*", line) and pod_type == self.POD_CONTROLLER:
                    name = line_array[0]
                    status = line_array[2]
                    node = line_array[6]
                    infos.append((name, status, node))
                if re.match(r"demooci-fss-pod-" + self._test_id + ".*", line) and pod_type == self.POD_CONTROLLER:
                    name = line_array[0]
                    status = line_array[2]
                    node = line_array[6]
                    infos.append((name, status, node))
        return infos

    def _get_compartment_id(self, pod_name):
        '''Gets the oci compartment_id from the oci-volume-provisioner pod host.
        This is where oci volume resources will be created.'''
        result = utils.kubectl("-n kube-system exec %s -- curl -s http://169.254.169.254/opc/v1/instance/" % pod_name,
                                exit_on_error=False, log_stdout=False)
        result_json = self._get_json_doc(str(result))
        compartment_id = result_json["compartmentId"]
        return compartment_id

    @staticmethod
    def _create_file_via_replication_controller(rc_name, file_name="hello.txt"):
        '''Create file via the replication controller
        @param rcName: Name of the replication controller to write data to
        @type rcName: C{Str}
        @param fileName: Name of file to create
        @type fileName: C{Str}'''
        utils.kubectl("exec " + rc_name + " -- touch /usr/share/nginx/html/" + file_name)

    @staticmethod
    def _verify_file_existance_via_replication_controller(rc_name, file_name="hello.txt"):
        '''Verify whether file exists via the replication controller
        @param rcName: Name of the replication controller to verify
        @type rcName: C{Str}
        @param fileName: Name of file to create
        @type fileName: C{Str}'''
        utils.log("Does the new file exist?")
        stdout = utils.kubectl("exec " + rc_name + " -- ls /usr/share/nginx/html")
        if file_name not in stdout.split("\n"):
            utils.log("Error: Failed to find file %s in mounted volume" % file_name)
            sys.exit(1)
        utils.log("Yes it does!")