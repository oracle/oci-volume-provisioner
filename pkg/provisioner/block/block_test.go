// Copyright (c) 2017, Oracle and/or its affiliates. All rights reserved.
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

package block

import (
	"context"
	"testing"
	"time"

	"github.com/oracle/oci-volume-provisioner/pkg/oci/client"

	"github.com/kubernetes-incubator/external-storage/lib/controller"
	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/core"
	"github.com/oracle/oci-go-sdk/identity"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	volumeBackupID = "dummyVolumeBackupId"
)

func TestResolveFSTypeWhenNotConfigured(t *testing.T) {
	options := controller.VolumeOptions{Parameters: make(map[string]string)}
	// test default fsType of 'ext4' is always returned.
	fst := resolveFSType(options)
	if fst != "ext4" {
		t.Fatalf("Unexpected filesystem type: '%s'.", fst)
	}
}

func TestResolveFSTypeWhenConfigured(t *testing.T) {
	// test default fsType of 'ext3' is always returned when configured.
	options := controller.VolumeOptions{Parameters: map[string]string{fsType: "ext3"}}
	fst := resolveFSType(options)
	if fst != "ext3" {
		t.Fatalf("Unexpected filesystem type: '%s'.", fst)
	}
}

type mockBlockStorageClient struct {
}

func (c *mockBlockStorageClient) CreateVolume(ctx context.Context, request core.CreateVolumeRequest) (response core.CreateVolumeResponse, err error) {
	return core.CreateVolumeResponse{Volume: core.Volume{Id: common.String(volumeBackupID)}}, nil
}

func (c *mockBlockStorageClient) DeleteVolume(ctx context.Context, request core.DeleteVolumeRequest) (response core.DeleteVolumeResponse, err error) {
	return core.DeleteVolumeResponse{}, nil
}

type mockIdentityClient struct {
	common.BaseClient
}

func (client mockIdentityClient) ListAvailabilityDomains(ctx context.Context, request identity.ListAvailabilityDomainsRequest) (response identity.ListAvailabilityDomainsResponse, err error) {
	return
}

type mockProvisionerClient struct {
}

func (p *mockProvisionerClient) BlockStorage() client.BlockStorage {
	return &mockBlockStorageClient{}
}

func (p *mockProvisionerClient) Identity() client.Identity {
	return &mockIdentityClient{}
}

func (p *mockProvisionerClient) Context() context.Context {
	return context.Background()
}

func (p *mockProvisionerClient) Timeout() time.Duration {
	return 30 * time.Second
}

func (p *mockProvisionerClient) CompartmentOCID() (compartmentOCID string) {
	return ""
}

func (p *mockProvisionerClient) TenancyOCID() string {
	return "ocid1.tenancy.oc1..aaaaaaaatyn7scrtwtqedvgrxgr2xunzeo6uanvyhzxqblctwkrpisvke4kq"
}

// NewClientProvisioner creates an OCI client from the given configuration.
func NewClientProvisioner(pcData client.ProvisionerClient) client.ProvisionerClient {
	return &mockProvisionerClient{}
}

func TestCreateVolumeFromBackup(t *testing.T) {
	// test creating a volume from an existing backup
	options := controller.VolumeOptions{PVName: "dummyVolumeOptions",
		PVC: &v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					ociVolumeBackupID: volumeBackupID,
				},
			},
		}}
	ad := identity.AvailabilityDomain{Name: common.String("dummyAdName"), CompartmentId: common.String("dummyCompartmentId")}
	block := blockProvisioner{client: NewClientProvisioner(nil)}
	provisionedVolume, err := block.Provision(options, &ad)
	if err != nil {
		t.Fatalf("Failed to provision volume from block storage: %v", err)
	}
	if provisionedVolume.Annotations[ociVolumeID] != volumeBackupID {
		t.Fatalf("Failed to assign the id of the blockID: %s, assigned %s instead", volumeBackupID,
			provisionedVolume.Annotations[ociVolumeID])
	}
}
