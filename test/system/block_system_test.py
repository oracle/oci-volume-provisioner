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

import os
import utils
from yaml_utils import PopulateYaml
from vol_provisioner_system_test import VolumeProvisionerSystemTestInterface

class BlockSystemTests(VolumeProvisionerSystemTestInterface):

    KUBERNETES_RESOURCES = ["../../dist/storage-class.yaml", "../../dist/storage-class-ext3.yaml",
                            "../../dist/oci-volume-provisioner.yaml",
                            "../../dist/oci-volume-provisioner-rbac.yaml"]
    SIMPLE_CLAIM_TEMPLATE = "templates/example-claim.template"
    EXT_CLAIM_TEMPLATE = "templates/example-claim-ext3.template"
    NO_AD_CLAIM_TEMPLATE = "templates/example-claim-no-AD.template"
    CM_SIMPLE = "volume_provisioner_simple"
    CM_EXT3 = "volume_provisioner_ext3"
    CM_NO_AD = "volume_provisioner_no_ad"

    def __init__(self, test_id=None, setup=False, check_oci=False, canaryMetrics=None):
        super(BlockSystemTests, self).__init__(test_id=test_id, setup=setup, check_oci=check_oci,
                                               k8Resources=self.KUBERNETES_RESOURCES, canaryMetrics=canaryMetrics)

    def run(self):
        super(BlockSystemTests, self).run()
        utils.log("Running system test: Simple", as_banner=True)
        self._testSuccess = self._test_create_volume(PopulateYaml(self.SIMPLE_CLAIM_TEMPLATE, region=self._region,
                                                     test_id=self._test_id).generateFile(), "demooci-" + self._test_id,
                                                     self._check_oci, canaryMetricName=self.CM_SIMPLE)
        self._checkTestSuccess()
        utils.log("Running system test: Ext3 file system", as_banner=True)
        self._testSuccess = self._test_create_volume(PopulateYaml(self.EXT_CLAIM_TEMPLATE,
                                                     self._test_id).generateFile(), "demooci-ext3-" + self._test_id,
                                                     self._check_oci, canaryMetricName=self.CM_EXT3)
        self._checkTestSuccess()
        utils.log("Running system test: No AD specified", as_banner=True)
        self._testSuccess = self._test_create_volume(PopulateYaml(self.NO_AD_CLAIM_TEMPLATE,
                                                     self._test_id).generateFile(), "demooci-no-ad-" + self._test_id,
                                                     self._check_oci, canaryMetricName=self.CM_NO_AD)
        self._checkTestSuccess()
