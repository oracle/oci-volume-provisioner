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

package helpers

import (
	"context"
	"time"

	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/core"
	"github.com/oracle/oci-go-sdk/filestorage"
	"github.com/oracle/oci-go-sdk/identity"
	"github.com/oracle/oci-volume-provisioner/pkg/oci/client"
)

var (
	// VolumeBackupID of backup volume
	VolumeBackupID = "dummyVolumeBackupId"
	fileSystemID   = "dummyFileSystemId"
	exportID       = "dummyExportID"
	serverIPs      = []string{"dummyServerIP"}
	privateIP      = "127.0.0.1"
)

// MockBlockStorageClient mocks BlockStorage client implementation
type MockBlockStorageClient struct {
}

// CreateVolume mocks the BlockStorage CreateVolume implementation
func (c *MockBlockStorageClient) CreateVolume(ctx context.Context, request core.CreateVolumeRequest) (response core.CreateVolumeResponse, err error) {
	return core.CreateVolumeResponse{Volume: core.Volume{Id: common.String(VolumeBackupID)}}, nil
}

// DeleteVolume mocks the BlockStorage DeleteVolume implementation
func (c *MockBlockStorageClient) DeleteVolume(ctx context.Context, request core.DeleteVolumeRequest) (response core.DeleteVolumeResponse, err error) {
	return core.DeleteVolumeResponse{}, nil
}

// MockFileStorageClient mocks FileStorage client implementation
type MockFileStorageClient struct {
}

// CreateFileSystem mocks the FileStorage CreateFileSystem implementation
func (c *MockFileStorageClient) CreateFileSystem(ctx context.Context, request filestorage.CreateFileSystemRequest) (response filestorage.CreateFileSystemResponse, err error) {
	return filestorage.CreateFileSystemResponse{FileSystem: filestorage.FileSystem{Id: common.String(fileSystemID)}}, nil
}

// DeleteFileSystem mocks the FileStorage DeleteFileSystem implementation
func (c *MockFileStorageClient) DeleteFileSystem(ctx context.Context, request filestorage.DeleteFileSystemRequest) (response filestorage.DeleteFileSystemResponse, err error) {
	return filestorage.DeleteFileSystemResponse{}, nil
}

// CreateExport mocks the FileStorage CreateExport implementation
func (c *MockFileStorageClient) CreateExport(ctx context.Context, request filestorage.CreateExportRequest) (response filestorage.CreateExportResponse, err error) {
	return filestorage.CreateExportResponse{Export: filestorage.Export{Id: common.String(exportID)}}, nil
}

// DeleteExport mocks the FileStorage DeleteExport implementation
func (c *MockFileStorageClient) DeleteExport(ctx context.Context, request filestorage.DeleteExportRequest) (response filestorage.DeleteExportResponse, err error) {
	return filestorage.DeleteExportResponse{}, nil
}

// CreateMountTarget mocks the FileStorage CreateMountTarget implementation
func (c *MockFileStorageClient) CreateMountTarget(ctx context.Context, request filestorage.CreateMountTargetRequest) (response filestorage.CreateMountTargetResponse, err error) {
	return filestorage.CreateMountTargetResponse{MountTarget: filestorage.MountTarget{PrivateIpIds: serverIPs}}, nil
}

// GetMountTarget mocks the FileStorage GetMountTarget implementation
func (c *MockFileStorageClient) GetMountTarget(ctx context.Context, request filestorage.GetMountTargetRequest) (response filestorage.GetMountTargetResponse, err error) {
	return filestorage.GetMountTargetResponse{}, nil
}

// ListMountTargets mocks the FileStorage ListMountTargets implementation
func (c *MockFileStorageClient) ListMountTargets(ctx context.Context, request filestorage.ListMountTargetsRequest) (response filestorage.ListMountTargetsResponse, err error) {
	return filestorage.ListMountTargetsResponse{}, nil
}

// MockVirtualNetworkClient mocks VirtualNetwork client implementation
type MockVirtualNetworkClient struct {
}

// GetPrivateIp mocks the VirtualNetwork GetPrivateIp implementation
func (c *MockVirtualNetworkClient) GetPrivateIp(ctx context.Context, request core.GetPrivateIpRequest) (response core.GetPrivateIpResponse, err error) {
	return core.GetPrivateIpResponse{PrivateIp: core.PrivateIp{IpAddress: common.String(privateIP)}}, nil
}

// MockIdentityClient mocks identity client structure
type MockIdentityClient struct {
	common.BaseClient
}

// ListAvailabilityDomains mocks the client ListAvailabilityDomains implementation
func (client MockIdentityClient) ListAvailabilityDomains(ctx context.Context, request identity.ListAvailabilityDomainsRequest) (response identity.ListAvailabilityDomainsResponse, err error) {
	return
}

// MockProvisionerClient mocks client structure
type MockProvisionerClient struct {
}

// BlockStorage mocks client BlockStorage implementation
func (p *MockProvisionerClient) BlockStorage() client.BlockStorage {
	return &MockBlockStorageClient{}
}

// FileStorage mocks client FileStorage implementation
func (p *MockProvisionerClient) FileStorage() client.FileStorage {
	return &MockFileStorageClient{}
}

// VirtualNetwork mocks client VirtualNetwork implementation
func (p *MockProvisionerClient) VirtualNetwork() client.VirtualNetwork {
	return &MockVirtualNetworkClient{}
}

// Identity mocks client Identity implementation
func (p *MockProvisionerClient) Identity() client.Identity {
	return &MockIdentityClient{}
}

// Context mocks client Context implementation
func (p *MockProvisionerClient) Context() context.Context {
	return context.Background()
}

// Timeout mocks client Timeout implementation
func (p *MockProvisionerClient) Timeout() time.Duration {
	return 30 * time.Second
}

// CompartmentOCID mocks client CompartmentOCID implementation
func (p *MockProvisionerClient) CompartmentOCID() (compartmentOCID string) {
	return ""
}

// TenancyOCID mocks client TenancyOCID implementation
func (p *MockProvisionerClient) TenancyOCID() string {
	return "ocid1.tenancy.oc1..aaaaaaaatyn7scrtwtqedvgrxgr2xunzeo6uanvyhzxqblctwkrpisvke4kq"
}

// NewClientProvisioner creates an OCI client from the given configuration.
func NewClientProvisioner(pcData client.ProvisionerClient) client.ProvisionerClient {
	return &MockProvisionerClient{}
}
