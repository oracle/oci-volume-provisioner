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

import re
import utils

class PopulateYaml():

    TEST_ID = "{{TEST_ID}}"
    REGION = "{{REGION}}"
    BACKUP_ID = "{{BACKUP_ID}}"
    MNT_TARGET_OCID = "{{MNT_TARGET_OCID}}"
    SUBNET_OCID = "{{SUBNET_OCID}}"
    VOLUME_NAME = "{{VOLUME_NAME}}"
    AVAILABILITY_DOMAIN = "{{AVAILABILITY_DOMAIN}}"
    TEMPLATE_ELEMENTS = {'_test_id': TEST_ID, '_region': REGION,
                         '_backup_id': BACKUP_ID, '_mount_target_ocid': MNT_TARGET_OCID,
                         '_subnet_ocid': SUBNET_OCID, '_volume_name': VOLUME_NAME,
                         '_availability_domain': AVAILABILITY_DOMAIN}

    def __init__(self, template_file, test_id, region=None, backup_id=None,
                 mount_target_ocid=None, subnet_ocid=None, volume_name=None, availability_domain=None):
        '''@param template: Name of file to use as template
        @type template: C{Str}
        @param test_id: Used for tagging resources with test id
        @type test_id: C{Str}
        @param region: Used for selecting resources from specified region
        @type region: C{Str}
        @param backup_id: Backup id to create PVC from
        @type backup_id: C{Str}
        @param mount_target_ocid: Mount target OCID to populate config with
        @type mount_target_ocid: C{Str}
        @param volume_name: Name used to create volume
        @type volume_name: C{Str}
        @param availability_domain: Availability domain (used for pvc)
        @type availability_domain: C{Str}'''
        self._template_file = template_file
        self._test_id = test_id
        self._region = region
        self._backup_id = backup_id
        self._mount_target_ocid = mount_target_ocid
        self._subnet_ocid = subnet_ocid
        self._volume_name = volume_name
        # yaml config does not allow ':'
        self._availability_domain = availability_domain.replace(':', '-')  if availability_domain else None

    def generateFile(self):
        '''Generate yaml based on the given template and fill in additional details
        @return: Name of generated config file
        @rtype: C{Str}'''
        yaml_file = self._template_file + ".yaml"
        with open(self._template_file, "r") as sources:
            lines = sources.readlines()
        with open(yaml_file, "w") as sources:
            for line in lines:
                patched_line = line
                for _elem, _elemName in self.TEMPLATE_ELEMENTS.iteritems():
                    if getattr(self, _elem) is not None:
                        patched_line = re.sub(_elemName, getattr(self, _elem), patched_line)
                    elif _elemName in [self.MNT_TARGET_OCID, self.SUBNET_OCID] and _elemName in patched_line:
                        # Remove lines from config files if attribute is not specified
                        utils.log("%s not specified. Removing reference from config" % _elemName)
                        patched_line = ""
                sources.write(patched_line)
        return yaml_file



