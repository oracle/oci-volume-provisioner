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

package provisioner

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
	exportSetID    = "dummyExportSetID"
	// NilListMountTargetsADID lists no mount targets for the given AD
	NilListMountTargetsADID = "dummyNilListMountTargetsForADID"
	mountTargetID           = "dummyMountTargetID"
	// CreatedMountTargetID for dynamically created mount target
	CreatedMountTargetID = "dummyCreatedMountTargetID"
	// ServerIPs address for mount target
	ServerIPs = []string{"dummyServerIP"}
	privateIP = "127.0.0.1"
)

// MockBlockStorageClient mocks BlockStorage client implementation
type MockBlockStorageClient struct {
	VolumeState core.VolumeLifecycleStateEnum
}

// CreateVolume mocks the BlockStorage CreateVolume implementation
func (c *MockBlockStorageClient) CreateVolume(ctx context.Context, request core.CreateVolumeRequest) (response core.CreateVolumeResponse, err error) {
	return core.CreateVolumeResponse{
		Volume: core.Volume{
			Id: common.String(VolumeBackupID),
		},
	}, nil
}

// DeleteVolume mocks the BlockStorage DeleteVolume implementation
func (c *MockBlockStorageClient) DeleteVolume(ctx context.Context, request core.DeleteVolumeRequest) (response core.DeleteVolumeResponse, err error) {
	return core.DeleteVolumeResponse{}, nil
}

// GetVolume mocks the BlockStorage GetVolume implementation
func (c *MockBlockStorageClient) GetVolume(ctx context.Context, request core.GetVolumeRequest) (response core.GetVolumeResponse, err error) {
	return core.GetVolumeResponse{
		Volume: core.Volume{
			LifecycleState: c.VolumeState,
		},
	}, nil
}

// MockFileStorageClient mocks FileStorage client implementation.
type MockFileStorageClient struct{}

// CreateFileSystem mocks the FileStorage CreateFileSystem implementation.
func (c *MockFileStorageClient) CreateFileSystem(ctx context.Context, request filestorage.CreateFileSystemRequest) (response filestorage.CreateFileSystemResponse, err error) {
	return filestorage.CreateFileSystemResponse{
		FileSystem: filestorage.FileSystem{
			Id: common.String(fileSystemID),
		},
	}, nil
}

// GetFileSystem mocks the FileStorage GetFileSystem implementation.
func (c *MockFileStorageClient) GetFileSystem(ctx context.Context, request filestorage.GetFileSystemRequest) (response filestorage.GetFileSystemResponse, err error) {
	return filestorage.GetFileSystemResponse{
		FileSystem: filestorage.FileSystem{
			Id:             request.FileSystemId,
			LifecycleState: filestorage.FileSystemLifecycleStateActive,
		},
	}, nil
}

// ListFileSystems mocks the FileStorage ListFileSystems implementation.
func (c *MockFileStorageClient) ListFileSystems(ctx context.Context, request filestorage.ListFileSystemsRequest) (response filestorage.ListFileSystemsResponse, err error) {
	return filestorage.ListFileSystemsResponse{
		Items: []filestorage.FileSystemSummary{{
			Id:             common.String(fileSystemID),
			DisplayName:    request.DisplayName,
			LifecycleState: filestorage.FileSystemSummaryLifecycleStateActive,
		}},
	}, nil
}

// DeleteFileSystem mocks the FileStorage DeleteFileSystem implementation
func (c *MockFileStorageClient) DeleteFileSystem(ctx context.Context, request filestorage.DeleteFileSystemRequest) (response filestorage.DeleteFileSystemResponse, err error) {
	return filestorage.DeleteFileSystemResponse{}, nil
}

// CreateExport mocks the FileStorage CreateExport implementation
func (c *MockFileStorageClient) CreateExport(ctx context.Context, request filestorage.CreateExportRequest) (response filestorage.CreateExportResponse, err error) {
	return filestorage.CreateExportResponse{
		Export: filestorage.Export{
			Id: common.String(exportID),
		},
	}, nil
}

// GetExport mocks the FileStorage CreateExport implementation.
func (c *MockFileStorageClient) GetExport(ctx context.Context, request filestorage.GetExportRequest) (response filestorage.GetExportResponse, err error) {
	return filestorage.GetExportResponse{
		Export: filestorage.Export{
			Id:             common.String(exportID),
			FileSystemId:   &fileSystemID,
			ExportSetId:    &exportSetID,
			LifecycleState: filestorage.ExportLifecycleStateActive,
			Path:           common.String("/" + fileSystemID),
		},
	}, nil
}

// ListExports mocks the FileStorage ListExports implementation.
func (c *MockFileStorageClient) ListExports(ctx context.Context, request filestorage.ListExportsRequest) (response filestorage.ListExportsResponse, err error) {
	return filestorage.ListExportsResponse{
		Items: []filestorage.ExportSummary{{
			Id:             common.String(exportID),
			ExportSetId:    request.ExportSetId,
			FileSystemId:   request.FileSystemId,
			LifecycleState: filestorage.ExportSummaryLifecycleStateActive,
		}},
	}, nil
}

// DeleteExport mocks the FileStorage DeleteExport implementation
func (c *MockFileStorageClient) DeleteExport(ctx context.Context, request filestorage.DeleteExportRequest) (response filestorage.DeleteExportResponse, err error) {
	return filestorage.DeleteExportResponse{}, nil
}

// GetMountTarget mocks the FileStorage GetMountTarget implementation
func (c *MockFileStorageClient) GetMountTarget(ctx context.Context, request filestorage.GetMountTargetRequest) (response filestorage.GetMountTargetResponse, err error) {
	return filestorage.GetMountTargetResponse{
		MountTarget: filestorage.MountTarget{
			PrivateIpIds:   ServerIPs,
			Id:             &CreatedMountTargetID,
			LifecycleState: filestorage.MountTargetLifecycleStateActive,
			ExportSetId:    &exportSetID,
		},
	}, nil
}

// MockVirtualNetworkClient mocks VirtualNetwork client implementation
type MockVirtualNetworkClient struct {
}

// GetPrivateIp mocks the VirtualNetwork GetPrivateIp implementation
func (c *MockVirtualNetworkClient) GetPrivateIp(ctx context.Context, request core.GetPrivateIpRequest) (response core.GetPrivateIpResponse, err error) {
	return core.GetPrivateIpResponse{
		PrivateIp: core.PrivateIp{
			IpAddress: common.String(privateIP),
		},
	}, nil
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
	Storage *MockBlockStorageClient
}

// BlockStorage mocks client BlockStorage implementation
func (p *MockProvisionerClient) BlockStorage() client.BlockStorage {
	return p.Storage
}

// FSS mocks client FileStorage implementation
func (p *MockProvisionerClient) FSS() client.FSS {
	return &MockFileStorageClient{}
}

// VCN mocks client VirtualNetwork implementation
func (p *MockProvisionerClient) VCN() client.VCN {
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
func NewClientProvisioner(pcData client.ProvisionerClient, storage *MockBlockStorageClient) client.ProvisionerClient {
	return &MockProvisionerClient{Storage: storage}
}
