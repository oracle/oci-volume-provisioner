#!/bin/bash

# Copyright (c) 2017 Oracle and/or its affiliates. All rights reserved.
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

DIR="$( cd "$(dirname "$0")" ; pwd -P )"
if [ "$#" -ne 1 ]; then
    echo "Invalid args: Usage: ./run-test-image.sh <image:version>"
    exit 1
fi
VERSION=$1
DOCKER_REGISTRY_USERNAME=${DOCKER_REGISTRY_USERNAME:-oracle}
TEST_ID=$(pwgen -A 8 1)

# Create the test image pod yaml
cat $DIR/run-test-image.yaml.template | \
    sed "s/{{VERSION}}/$VERSION/g" | \
    sed "s/{{DOCKER_REGISTRY_USERNAME}}/$DOCKER_REGISTRY_USERNAME/g" | \
    sed "s/{{TEST_ID}}/$TEST_ID/g" \
    > $DIR/run-test-image.yaml.$TEST_ID

if [[ -z "${KUBECONFIG}" ]]; then
    if [[ -z "${KUBECONFIG_VAR}" ]]; then
        echo "KUBECONFIG or KUBECONFIG_VAR must be set"
        exit 1
    else
       echo "$KUBECONFIG_VAR" | openssl enc -base64 -d -A > /tmp/kubeconfig
       export KUBECONFIG=/tmp/kubeconfig
   fi
fi

if [[ -z "${OCI_API_KEY}" ]]; then
    if [[ -z "${OCI_API_KEY_VAR}" ]]; then
        echo "OCI_API_KEY or OCI_API_KEY_VAR must be set"
        exit 1
    else
       echo "$OCI_API_KEY_VAR" | openssl enc -base64 -d -A > /tmp/oci_api_key.pem
       export OCI_API_KEY=/tmp/oci_api_key.pem
   fi
fi
# Starts the test image inside the cluster and waits for it to complete.
exitCodeCmd="kubectl get po volume-provisioner-system-test-$TEST_ID -o json | jq '.status.containerStatuses[0].state.terminated.exitCode'"
kubectl create -f $DIR/run-test-image.yaml.$TEST_ID
while [ "$(eval $exitCodeCmd)" == null ]; do
    echo -n "."
    sleep 10
done
kubectl logs volume-provisioner-system-test-$TEST_ID
exitCode=$(eval $exitCodeCmd)
kubectl delete -f $DIR/run-test-image.yaml.$TEST_ID
exit $exitCode
