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
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/oracle/oci-volume-provisioner/pkg/oci/client"
	"github.com/oracle/oci-volume-provisioner/pkg/oci/instancemeta"

	"github.com/kubernetes-incubator/external-storage/lib/controller"
	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/core"
	"github.com/oracle/oci-go-sdk/identity"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
)

// LoadConfig loads confuration data from a given path
func loadConfig(configFilePath string, t *testing.T) *client.ProvisionerClient {
	f, err := os.Open(configFilePath)
	if err != nil {
		t.Fatalf("Unable to load volume provisioner configuration file: %v", configFilePath)
	}
	defer f.Close()

	cfg, err := client.LoadConfig(f)
	if err != nil {
		t.Fatalf("Unable to load volume provisioner client: %v", err)
	}
	pc, err := client.FromConfig(cfg)
	if err != nil {
		t.Fatalf("Unable to load volume provisioner client details: %v", err)
	}
	return &pc
}

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
	common.BaseClient
	config *common.ConfigurationProvider
}

func (c *mockBlockStorageClient) CreateVolume(ctx context.Context, request core.CreateVolumeRequest) (response core.CreateVolumeResponse, err error) {
	volID := "dummyVolumeId"
	createVolumeResp := &core.CreateVolumeResponse{Volume: core.Volume{Id: &volID}}
	return *createVolumeResp, nil
}

func (c *mockBlockStorageClient) DeleteVolume(ctx context.Context, request core.DeleteVolumeRequest) (response core.DeleteVolumeResponse, err error) {
	deleteVolumeResp := &core.DeleteVolumeResponse{}
	return *deleteVolumeResp, nil
}

type mockIdentityClient struct {
	common.BaseClient
	config *common.ConfigurationProvider
}

func (client mockIdentityClient) ListAvailabilityDomains(ctx context.Context, request identity.ListAvailabilityDomainsRequest) (response identity.ListAvailabilityDomainsResponse, err error) {
	httpRequest, err := common.MakeDefaultHTTPRequestWithTaggedStruct(http.MethodGet, "/availabilityDomains/", request)
	if err != nil {
		return
	}

	httpResponse, err := client.Call(ctx, &httpRequest)
	defer common.CloseBodyIfValid(httpResponse)
	response.RawResponse = httpResponse
	if err != nil {
		return
	}

	err = common.UnmarshalResponse(httpResponse, &response)
	return
}

type mockProvisionerClient struct {
	cfg          *client.Config
	blockStorage *core.BlockstorageClient
	identity     *identity.IdentityClient
	context      context.Context
	timeout      time.Duration
	metadata     *instancemeta.InstanceMetadata
}

func (p *mockProvisionerClient) BlockStorage() client.BlockStorage {
	return &mockBlockStorageClient{}
}

func (p *mockProvisionerClient) Identity() client.Identity {
	return &mockIdentityClient{}
}

func (p *mockProvisionerClient) Context() context.Context {
	return p.context
}

func (p *mockProvisionerClient) Timeout() time.Duration {
	return p.timeout
}

func (p *mockProvisionerClient) CompartmentOCID() (compartmentOCID string) {
	return ""
}

func (p *mockProvisionerClient) TenancyOCID() string {
	return p.cfg.Auth.TenancyOCID
}

// NewClientProvisioner creates an OCI client from the given configuration.
func NewClientProvisioner(pcData client.ProvisionerClient) client.ProvisionerClient {
	return &mockProvisionerClient{
		context: pcData.Context(),
		timeout: pcData.Timeout()}
}

func TestCreateVolumeFromBackup(t *testing.T) {
	// test creating a volume from an existing backup
	options := controller.VolumeOptions{PVName: "dummyVolumeOptions"}
	volumeBackupID := "dummyVolumeBackupId"
	pv := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				ociVolumeBackupID: "dummy",
			},
		},
	}
	options.PVC = pv
	adName := "dummyAdName"
	adCompartmentID := "dummyCompartmentId"
	availabilityDomain := identity.AvailabilityDomain{Name: &adName, CompartmentId: &adCompartmentID}
	configFilePath := "config/blockProvision.yaml"
	pcData := loadConfig(configFilePath, t)
	pc := NewClientProvisioner(*pcData)
	block := blockProvisioner{client: pc}
	volID := "dummyVolumeId"
	provisionedVolume, err := block.Provision(options, &availabilityDomain)
	if err != nil {
		t.Fatalf("Failed to provision volume from block storage: %v", err)
	}
	if provisionedVolume.Annotations[ociVolumeID] != volID {
		t.Fatalf("Failed to assign the id of the blockID: %s, assigned %s instead", volumeBackupID,
			provisionedVolume.Annotations[ociVolumeID])
	}
}
