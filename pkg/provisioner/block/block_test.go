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
	"fmt"
	"testing"
	"time"

	"github.com/oracle/oci-volume-provisioner/pkg/oci/client"
	"github.com/oracle/oci-volume-provisioner/pkg/oci/instancemeta"

	"github.com/kubernetes-incubator/external-storage/lib/controller"
	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/core"
	"github.com/oracle/oci-go-sdk/identity"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
)

var (
	volumeBackupID = "dummyVolumeBackupId"
	defaultAD      = identity.AvailabilityDomain{Name: common.String("PHX-AD-1"), CompartmentId: common.String("ocid1.compartment.oc1")}
)

func TestValidateFsType(t *testing.T) {
	err := validateFSType("ext3")
	if err != nil {
		t.Fatalf("Unexpected validation error for fsType: '%s'", "ext3")
	}
	err = validateFSType("ext4")
	if err != nil {
		t.Fatalf("Unexpected validation error for fsType: '%s'", "ext4")
	}
	err = validateFSType("abc3")
	if err == nil {
		t.Fatalf("Unexpected validation pass for fsType: '%s'", "abc3")
	}
}

func TestResolveFSTypeWhenNotConfigured(t *testing.T) {
	options := controller.VolumeOptions{Parameters: make(map[string]string)}
	// test default fsType of 'ext4' is always returned.
	fst, _ := resolveFSType(options)
	if fst != "ext4" {
		t.Fatalf("Unexpected filesystem type: '%s'.", fst)
	}
}

func TestResolveFSTypeWhenConfigured(t *testing.T) {
	// test default fsType of 'ext3' is always returned when configured.
	parameters := map[string]string{"fsType": "ext3"}
	options := controller.VolumeOptions{Parameters: parameters}
	fst, _ := resolveFSType(options)
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
	options := controller.VolumeOptions{
		PVName: "dummyVolumeOptions",
		PVC: &v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					ociVolumeBackupID: volumeBackupID,
				},
			},
			Spec: v1.PersistentVolumeClaimSpec{
				StorageClassName: common.String("oci"),
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceName(v1.ResourceStorage): resource.MustParse("50Gi"),
					},
				},
			},
		}}

	block := blockProvisioner{
		client: NewClientProvisioner(nil),
		metadata: instancemeta.NewMock(&instancemeta.InstanceMetadata{
			CompartmentOCID: "",
			Region:          "phx",
		})}
	provisionedVolume, err := block.Provision(options, &defaultAD)
	if err != nil {
		t.Fatalf("Failed to provision volume from block storage: %v", err)
	}
	if provisionedVolume.Annotations[ociVolumeID] != volumeBackupID {
		t.Fatalf("Failed to assign the id of the blockID: %s, assigned %s instead", volumeBackupID,
			provisionedVolume.Annotations[ociVolumeID])
	}
}

func TestVolumeRoundingLogic(t *testing.T) {
	var volumeRoundingTests = []struct {
		requestedStorage string
		enabled          bool
		minVolumeSize    resource.Quantity
		expected         string
	}{
		{"20Gi", true, resource.MustParse("50Gi"), "50Gi"},
		{"30Gi", true, resource.MustParse("25Gi"), "30Gi"},
		{"50Gi", true, resource.MustParse("50Gi"), "50Gi"},
		{"20Gi", false, resource.MustParse("50Gi"), "20Gi"},
	}
	for i, tt := range volumeRoundingTests {
		t.Run(fmt.Sprintf("test-%d", i), func(t *testing.T) {
			volumeOptions := controller.VolumeOptions{
				PVC: createPVC(tt.requestedStorage),
			}
			metadata := instancemeta.NewMock(&instancemeta.InstanceMetadata{
				CompartmentOCID: "",
				Region:          "phx",
			})
			block := NewBlockProvisioner(NewClientProvisioner(nil), metadata, tt.enabled, tt.minVolumeSize)
			provisionedVolume, err := block.Provision(volumeOptions, &defaultAD)
			if err != nil {
				t.Fatalf("Expected no error but got %s", err)
			}

			expectedCapacity := resource.MustParse(tt.expected)
			actualCapacity := provisionedVolume.Spec.Capacity[v1.ResourceName(v1.ResourceStorage)]

			actual := actualCapacity.String()
			expected := expectedCapacity.String()
			if actual != expected {
				t.Fatalf("Expected volume to be %s but got %s", expected, actual)
			}
		})
	}
}

func createPVC(size string) *v1.PersistentVolumeClaim {
	return &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{},
		Spec: v1.PersistentVolumeClaimSpec{
			StorageClassName: common.String("oci"),
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceName(v1.ResourceStorage): resource.MustParse(size),
				},
			},
		},
	}
}
