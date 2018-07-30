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
import sys
import time
import uuid
import oci
import yaml
import utils
from yaml_utils import PopulateYaml
from vol_provisioner_system_test import VolumeProvisionerSystemTestInterface
from block_system_test import BlockSystemTests
from fss_system_test import FSSSystemTests
from backup_vol_system_test import BackupVolumeSystemTest
from canary_metrics import CanaryMetrics

TEST_NAME = "volumeprovisionersystemtest"
REGION = "us-ashburn-1"
WRITE_REPORT=True
OCICONFIG = "OCICONFIG"
OCICONFIG_VAR = "OCICONFIG_VAR"
KUBECONFIG_VAR = "KUBECONFIG_VAR"
SUBNET_OCID = "SUBNET_OCID"
METRICS_FILE = "METRICS_FILE"
MNT_TARGET_OCID = "MNT_TARGET_OCID"


def _check_env(check_oci):
    if check_oci:
        if OCICONFIG not in os.environ and OCICONFIG_VAR not in os.environ:
            utils.log("Error. Can't find either OCICONFIG or OCICONFIG_VAR in the environment.")
            utils.finish_with_exit_code(1)

def _create_key_files(check_oci):
    utils.log("Setting environment variables")
    if OCICONFIG_VAR in os.environ:
        utils.run_command("echo \"$OCICONFIG_VAR\" | openssl enc -base64 -d -A > " + VolumeProvisionerSystemTestInterface.TMP_OCICONFIG, ".")
        utils.run_command("chmod 600 " + VolumeProvisionerSystemTestInterface.TMP_OCICONFIG, ".")
    if KUBECONFIG_VAR in os.environ:
        utils.run_command("echo \"$KUBECONFIG_VAR\" | openssl enc -base64 -d -A > " + utils.TMP_KUBECONFIG, ".")

    if check_oci:
        oci_config_file = VolumeProvisionerSystemTestInterface()._get_oci_config_file()
        with open(oci_config_file, 'r') as stream:
            try:
                cnf = yaml.load(stream)
                with open(VolumeProvisionerSystemTestInterface.TMP_OCI_API_KEY_FILE, 'w') as stream:
                    stream.write(cnf['auth']['key'])
            except yaml.YAMLError as err:
                utils.log("Error. Failed to parse oci config file %s. Error: %s " % (oci_config_file, err))
                utils.finish_with_exit_code(1)


def _destroy_key_files(check_oci):
    if OCICONFIG_VAR in os.environ:
        os.remove(VolumeProvisionerSystemTestInterface.TMP_OCICONFIG)
    if KUBECONFIG_VAR in os.environ:
        os.remove(utils.TMP_KUBECONFIG)
    if check_oci:
        os.remove(VolumeProvisionerSystemTestInterface.TMP_OCI_API_KEY_FILE)

def _get_oci_api_key_file():
    return VolumeProvisionerSystemTestInterface.TMP_OCI_API_KEY_FILE

def _get_timestamp(test_id):
    return test_id if test_id is not None else datetime.datetime.now().strftime('%Y%m%d%H%M%S%f')

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
        utils.log("If --check-oci is specified, then --setup also needs to be set.")
        utils.finish_with_exit_code(1)

    return args

def _main():
    utils.reset_debug_file()
    args = _handle_args()

    _check_env(args['check_oci'])
    _create_key_files(args['check_oci'])
    def _destroy_key_files_atexit():
        _destroy_key_files(args['check_oci'])
    atexit.register(_destroy_key_files_atexit)

    test_id = str(uuid.uuid4())[:8]
    canaryMetrics = CanaryMetrics(metrics_file=os.environ.get(METRICS_FILE))
    if args['teardown']:
        def _teardown_atexit():
            utils.log("Tearing down the volume provisioner", as_banner=True)
            # BlockSystemTests(test_id, args['setup']).cleanup()
            FSSSystemTests(test_id, args['setup']).cleanup()
            # BackupVolumeSystemTest(test_id, args['setup']).cleanup()
        atexit.register(_teardown_atexit)

    if not args['no_test']:
        BlockSystemTests(test_id=test_id, setup=args['setup'], check_oci=args['check_oci'], canaryMetrics=canaryMetrics).run()
        FSSSystemTests(subnet_ocid=os.environ.get(SUBNET_OCID), test_id=test_id, setup=args['setup'], check_oci=args['check_oci'], canaryMetrics=canaryMetrics).run()
        BackupVolumeSystemTest(test_id=test_id, setup=args['setup'], check_oci=args['check_oci'], canaryMetrics=canaryMetrics).run()
    canaryMetrics.finish_canary_metrics()
    utils.finish_with_exit_code(0)

if __name__ == "__main__":
    _main()
