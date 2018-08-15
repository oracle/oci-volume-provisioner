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
	"context"
	"reflect"
	"testing"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubernetes-incubator/external-storage/lib/controller"
	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/identity"

	"github.com/oracle/oci-volume-provisioner/pkg/provisioner"
)

func TestGetMountTargetFromID(t *testing.T) {
	// test retrieving a mount target from given ID
	var ctx = context.Background()
	fss := filesystemProvisioner{client: provisioner.NewClientProvisioner(nil, nil)}
	resp, err := fss.getMountTargetFromID(ctx, "mtOCID")
	if err != nil {
		t.Fatalf("Failed to retrieve mount target from ID: %v", err)
	}
	if !reflect.DeepEqual(resp.PrivateIpIds, provisioner.ServerIPs) {
		t.Fatalf("Incorrect response for retrieving mount target from ID")
	}
}

func TestListAllMountTargets(t *testing.T) {
	// test listing all mount targets
	var ctx = context.Background()
	fss := filesystemProvisioner{client: provisioner.NewClientProvisioner(nil, nil)}
	resp, err := fss.listAllMountTargets(ctx, "adOCID")
	if err != nil {
		t.Fatalf("Failed to retrieve list mount targets: %v", err)
	}
	if !reflect.DeepEqual(resp, provisioner.MountTargetItems) {
		t.Fatalf("Incorrect response for listing mount targets")
	}
}

func TestGetOrCreateMountTarget(t *testing.T) {
	// test get or create mount target
	var ctx = context.Background()
	fss := filesystemProvisioner{client: provisioner.NewClientProvisioner(nil, nil)}
	resp, err := fss.getOrCreateMountTarget(ctx, "", provisioner.NilListMountTargetsADID, "subnetID")
	if err != nil {
		t.Fatalf("Failed to retrieve or create mount target: %v", err)
	}
	if *resp.Id != provisioner.CreatedMountTargetID {
		t.Fatalf("Failed to create mount target")
	}

}
func TestCreateVolumeWithFSS(t *testing.T) {
	// test creating a volume on a file system storage
	options := controller.VolumeOptions{
		PVName: "dummyVolumeOptions",
		PVC: &v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{},
		}}
	ad := identity.AvailabilityDomain{Name: common.String("dummyAdName"), CompartmentId: common.String("dummyCompartmentId")}
	fss := filesystemProvisioner{client: provisioner.NewClientProvisioner(nil, nil)}
	_, err := fss.Provision(options, &ad)
	if err != nil {
		t.Fatalf("Failed to provision volume from fss storage: %v", err)
	}
}
