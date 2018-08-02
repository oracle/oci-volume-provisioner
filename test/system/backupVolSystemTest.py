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

from volProvisionerSystemTest import VolumeProvisionerSystemTestInterface
import oci
import utils
import atexit
from yamlUtils import PopulateYaml

class BackupVolumeSystemTest(VolumeProvisionerSystemTestInterface):

    KUBERNETES_RESOURCES = ["../../dist/storage-class.yaml", "../../dist/storage-class-ext3.yaml",
                            "../../dist/oci-volume-provisioner.yaml",
                            "../../dist/oci-volume-provisioner-rbac.yaml"]

    def __init__(self, test_id=None, setup=False, check_oci=False):
        super(BackupVolumeSystemTest, self).__init__(test_id=test_id, setup=setup, check_oci=check_oci, k8Resources=self.KUBERNETES_RESOURCES)

    def run(self):
        super(BackupVolumeSystemTest, self).run()
        if self._check_oci:
            utils.log("Running system test: Create volume from backup", as_banner=True)
            _backup_ocid, _availability_domain = self._setup_create_volume_from_backup()
            _claim_target = PopulateYaml("../../examples/example-claim-from-backup.template", self._test_id, 
                                        region=_availability_domain.split(':')[1], backup_id=_backup_ocid).generateFile()
            self._test_create_volume(_claim_target, "demooci-from-backup-" + self._test_id,
                                     availability_domain=_availability_domain,
                                     verify_func=self._volume_from_backup_check)
            self._tear_down_create_volume_from_backup(_backup_ocid)

    def _create_backup(self, volume_ocid):
        '''Create volume backup on OCI from existing volume
        @param volume_ocid: Ocid of course volume
        @type volume_ocid: C{Str}
        @return: Tuple containing the backup id, compartment id and display name
        @rtype: C{Tuple}'''
        client = oci.core.blockstorage_client.BlockstorageClient(self._oci_config)
        _backup_details = oci.core.models.CreateVolumeBackupDetails(volume_id=volume_ocid, 
                                                                    display_name="backup_volume_system_test" + self._test_id)
        _response = client.create_volume_backup(_backup_details)
        utils.log("Response for creating backup for volume %s: %s" % (volume_ocid, _response.data))
        _res = self._get_json_doc(str(_response.data))
        return _res['id'], _res['compartment_id'], _res['display_name']

    def _delete_backup(self, backup_ocid):
        '''Delete volume backup from OCI
        @param backup_ocid: Ocid of backup volume to delete
        @type backup_ocid: C{Str}'''
        client = oci.core.blockstorage_client.BlockstorageClient(self._oci_config)
        _response = client.delete_volume_backup(backup_ocid)
        utils.log("Response for deleting volume backup %s: %s" % (backup_ocid, _response.data))

    def _create_volume_from_backup(self, backup_ocid, test_id, availability_domain, compartment_id):
        client = oci.core.blockstorage_client.BlockstorageClient(self._oci_config)
        _volume_details = oci.core.models.CreateVolumeDetails(volume_backup_id=backup_ocid, 
                                                            display_name="restored_volume_system_test" + test_id,
                                                            availability_domain=availability_domain,
                                                            compartment_id=compartment_id)
        try:
            _response = client.create_volume(_volume_details)
            utils.log("Response for creating volume from backup %s: %s %s" % (_response.data, self._get_json_doc(str(_response.data))['id'], compartment_id))
            return self._get_json_doc(str(_response.data))['id']
        except Exception as exc:
            utils.log("Failed to create volume from backup %s" % exc)

    def  _setup_create_volume_from_backup(self, storageType=VolumeProvisionerSystemTestInterface.BLOCK_STORAGE, availability_domain=None):
        '''Setup environment for creating a volume from a backup device
        @return: OCID of generated backup
        @rtype: C{Str}'''
        utils.log("Creating test volume (using terraform)", as_banner=True)
        self._terraform("init", self.TERRAFORM_DIR)
        self._terraform("apply", self.TERRAFORM_DIR)
        _availability_domain = self._get_terraform_output_var(self.TERRAFORM_AVAILABILITY_DOMAIN)
        utils.log(self._terraform("output -json", self.TERRAFORM_DIR))
        # Create replication controller and write data to the generated volume
        _rc_name, _rc_config = self._create_rc_or_pod("../../examples/example-replication-controller-with-volume-claim.template",
                                                      _availability_domain, volume_name=self._get_volume_name())
        self._create_file_via_replication_controller(_rc_name)  
        self._verify_file_existance_via_replication_controller(_rc_name)
        # Create backup from generated volume
        _backup_ocid, compartment_id, _volume_name = self._create_backup(self._get_terraform_output_var(self.TERRAFORM_VOLUME_OCID))
        if not self._wait_for_volume_to_create(_backup_ocid, compartment_id=compartment_id, backup=True, storageType=storageType,
                                               availability_domain=availability_domain):
            utils.log("Failed to find backup with name: " + _volume_name)  
        return _backup_ocid, _availability_domain

    def _tear_down_create_volume_from_backup(self, backup_ocid):
        '''Tear down create volume from backup
        @param test_id: Test id used to append to component names
        @type test_id: C{Str}
        @param backup_ocid: OCID of backup from which the test volume was created
        @type backup_ocid: C{Str}'''
        def _destroy_test_volume_atexit():
            utils.log("Destroying test volume (using terraform)", as_banner=True)
            self._terraform("destroy -force", self.TERRAFORM_DIR)
        atexit.register(_destroy_test_volume_atexit)
        self._delete_backup(backup_ocid)

    def _volume_from_backup_check(self, test_id, availability_domain, volume, file_name='hello.txt'):
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
        _rc_name, _rc_config = self._create_rc_or_pod("../../examples/example-replication-controller.template", availability_domain, _ocid)
        utils.log("Does the file from the previous backup exist?")
        stdout = utils.kubectl("exec " + _rc_name + " -- ls /usr/share/nginx/html")
        if file_name not in stdout.split("\n"):
            utils.log("Error: Failed to find file %s in mounted volume" % file_name)
        utils.log("Deleting the replication controller (deletes the single nginx pod).")
        utils.kubectl("delete -f " + _rc_config)