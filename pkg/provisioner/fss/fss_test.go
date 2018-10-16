// Copyright (c) 2018, Oracle and/or its affiliates. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fss

import (
	"testing"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubernetes-incubator/external-storage/lib/controller"
	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/identity"
	"go.uber.org/zap/zaptest"

	"github.com/oracle/oci-volume-provisioner/pkg/provisioner"
)

func TestCreateVolumeWithFSS(t *testing.T) {
	fsp := filesystemProvisioner{
		client: provisioner.NewClientProvisioner(nil, nil),
		logger: zaptest.NewLogger(t).Sugar(),
		region: "phx",
	}
	_, err := fsp.Provision(
		controller.VolumeOptions{
			PVName: "dummyVolumeOptions",
			PVC: &v1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					UID: "my-uid",
				},
			},
			Parameters: map[string]string{MntTargetID: "dummyMountTargetID"},
		},
		&identity.AvailabilityDomain{
			Name:          common.String("dummyAdName"),
			CompartmentId: common.String("dummyCompartmentId"),
		},
	)
	if err != nil {
		t.Fatalf("Failed to provision volume from fss storage: %v", err)
	}
}
