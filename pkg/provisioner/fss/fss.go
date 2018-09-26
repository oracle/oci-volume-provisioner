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
	"fmt"
	"math/rand"
	"os"
	"time"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"go.uber.org/zap"

	"github.com/kubernetes-incubator/external-storage/lib/controller"
	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/core"
	"github.com/oracle/oci-go-sdk/filestorage"
	"github.com/oracle/oci-go-sdk/identity"
	"github.com/pkg/errors"

	"github.com/oracle/oci-volume-provisioner/pkg/oci/client"
	"github.com/oracle/oci-volume-provisioner/pkg/oci/instancemeta"
	"github.com/oracle/oci-volume-provisioner/pkg/provisioner"
	"github.com/oracle/oci-volume-provisioner/pkg/provisioner/plugin"

	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	ociVolumeID = "volume.beta.kubernetes.io/oci-volume-id"
	ociExportID = "volume.beta.kubernetes.io/oci-export-id"
	fsType      = "fsType"

	// SubnetID is the name of the parameter which holds the target subnet ocid.
	SubnetID = "subnetId"
	// MntTargetID is the name of the parameter which hold the target mount target ocid.
	MntTargetID = "mntTargetId"
)

const (
	defaultTimeout  = 5 * time.Minute
	defaultInterval = 5 * time.Second
)

// filesystemProvisioner is the internal provisioner for OCI filesystem volumes
type filesystemProvisioner struct {
	client   client.ProvisionerClient
	metadata instancemeta.Interface
	logger   *zap.SugaredLogger
}

var _ plugin.ProvisionerPlugin = &filesystemProvisioner{}

var (
	errNoCandidateFound = errors.New("no candidate mount targets found")
	errNotFound         = errors.New("not found")
)

// NewFilesystemProvisioner creates a new file system provisioner that creates
// filesystems using OCI File System Service.
func NewFilesystemProvisioner(logger *zap.SugaredLogger, client client.ProvisionerClient, metadata instancemeta.Interface) plugin.ProvisionerPlugin {
	return &filesystemProvisioner{
		client:   client,
		metadata: metadata,
		logger: logger.With(
			"compartmentID", client.CompartmentOCID(),
			"tenancyID", client.TenancyOCID(),
		),
	}
}

// getMountTargetFromID retrieves mountTarget from given mountTargetID
func (fsp *filesystemProvisioner) getMountTargetFromID(ctx context.Context, mountTargetID string) (*filestorage.MountTarget, error) {
	resp, err := fsp.client.FSS().GetMountTarget(ctx, filestorage.GetMountTargetRequest{
		MountTargetId: &mountTargetID,
	})
	if err != nil {
		return nil, err
	}
	return &resp.MountTarget, nil
}

// getCandidateMountTargetID retrieves all available mount targets.
func (fsp *filesystemProvisioner) getCandidateMountTargetID(ctx context.Context, ad string) (string, error) {
	var (
		page     *string
		active   []*filestorage.MountTargetSummary
		creating []*filestorage.MountTargetSummary
	)
	for {
		resp, err := fsp.client.FSS().ListMountTargets(ctx, filestorage.ListMountTargetsRequest{
			AvailabilityDomain: &ad,
			CompartmentId:      common.String(fsp.client.CompartmentOCID()),
			Page:               page,
		})
		if err != nil {
			return "", err
		}
		for _, mt := range resp.Items {
			switch mt.LifecycleState {
			case filestorage.MountTargetSummaryLifecycleStateActive:
				active = append(active, &mt)
			case filestorage.MountTargetSummaryLifecycleStateCreating:
				creating = append(creating, &mt)
			}
		}
		if page = resp.OpcNextPage; resp.OpcNextPage == nil {
			break
		}
	}

	var candidates []*filestorage.MountTargetSummary
	if len(active) != 0 {
		candidates = active
	} else {
		candidates = creating
	}

	if len(candidates) == 0 {
		return "", nil
	}

	return *(candidates[rand.Int()%len(candidates)]).Id, nil
}

func (fsp *filesystemProvisioner) createMountTarget(ctx context.Context, logger *zap.SugaredLogger, ad, subnetID string) (string, error) {
	displayName := fmt.Sprintf("%s%s", provisioner.GetPrefix(), "mnt")

	logger = logger.With("mountTargetDisplayName", displayName)
	logger.Info("Creating mount target")

	resp, err := fsp.client.FSS().CreateMountTarget(ctx, filestorage.CreateMountTargetRequest{
		CreateMountTargetDetails: filestorage.CreateMountTargetDetails{
			AvailabilityDomain: &ad,
			SubnetId:           &subnetID,
			CompartmentId:      common.String(fsp.client.CompartmentOCID()),
			DisplayName:        &displayName,
		},
	})
	if err != nil {
		return "", err
	}

	return *resp.MountTarget.Id, nil
}

func (fsp *filesystemProvisioner) getOrCreateMountTarget(ctx context.Context, mtID string, ad string, subnetID string) (*filestorage.MountTarget, error) {
	logger := fsp.logger.With("subnetId", subnetID)

	if mtID != "" {
		return fsp.awaitMountTarget(ctx, logger, mtID)
	}

	var err error
	mtID, err = fsp.getCandidateMountTargetID(ctx, ad)
	if err != nil {
		if err != errNoCandidateFound {
			return nil, err
		}

		mtID, err = fsp.createMountTarget(ctx, logger, ad, subnetID)
		if err != nil {
			return nil, err
		}
	}

	return fsp.awaitMountTarget(ctx, logger, mtID)
}

func (fsp *filesystemProvisioner) awaitFileSystem(ctx context.Context, logger *zap.SugaredLogger, id string) (*filestorage.FileSystem, error) {
	var fs *filestorage.FileSystem
	err := wait.Poll(defaultInterval, defaultTimeout, func() (bool, error) {
		logger.Debug("Polling file system lifecycle state")

		resp, err := fsp.client.FSS().GetFileSystem(ctx, filestorage.GetFileSystemRequest{
			FileSystemId: &id,
		})

		if err != nil {
			return false, err
		}

		switch resp.LifecycleState {
		case filestorage.FileSystemLifecycleStateActive:
			fs = &resp.FileSystem
			logger.Infof("File system is in lifecycle state %s", resp.LifecycleState)
			return true, nil
		default:
			logger.Debugf("File system is in lifecycle state %s", resp.FileSystem.LifecycleState)
			return false, nil
		}
	})

	return fs, err
}

func (fsp *filesystemProvisioner) awaitMountTarget(ctx context.Context, logger *zap.SugaredLogger, id string) (*filestorage.MountTarget, error) {
	logger.Info("Waiting for MountTarget to be in lifecycle state ACTIVE")

	var mt *filestorage.MountTarget
	if err := wait.Poll(defaultInterval, defaultTimeout, func() (bool, error) {
		logger.Debug("Polling mount target lifecycle state")

		resp, err := fsp.client.FSS().GetMountTarget(ctx, filestorage.GetMountTargetRequest{MountTargetId: &id})
		if err != nil {
			return false, err
		}

		mt = &resp.MountTarget

		switch resp.LifecycleState {
		case filestorage.MountTargetLifecycleStateActive:
			logger.Info("Mount target is in lifecycle state ACTIVE")
			return true, nil
		case filestorage.MountTargetLifecycleStateFailed:
			logger.Error("Mount target is in lifecycle state FAILED")
			return false, fmt.Errorf("mount target %q is in lifecycle state FAILED", *resp.MountTarget.Id)
		default:
			logger.Debugf("Mount target is in lifecycle state %s", resp.MountTarget.LifecycleState)
			return false, nil
		}
	}); err != nil {
		return nil, err
	}
	return mt, nil
}

func (fsp *filesystemProvisioner) awaitExport(ctx context.Context, logger *zap.SugaredLogger, id string) (*filestorage.Export, error) {
	logger.Info("Waiting for Export to be in lifecycle state ACTIVE")

	var export *filestorage.Export
	if err := wait.Poll(defaultInterval, defaultTimeout, func() (bool, error) {
		logger.Debug("Polling export lifecycle state")

		resp, err := fsp.client.FSS().GetExport(ctx, filestorage.GetExportRequest{ExportId: &id})
		if err != nil {
			return false, err
		}

		export = &resp.Export

		switch state := resp.LifecycleState; state {
		case filestorage.ExportLifecycleStateActive:
			logger.Info("Export is in lifecycle state ACTIVE")
			return true, nil
		case filestorage.ExportLifecycleStateDeleting, filestorage.ExportLifecycleStateDeleted:
			logger.Errorf("Export is in lifecycle state %s", state)
			return false, fmt.Errorf("export %q is in lifecycle state %s", *resp.Export.Id, state)
		default:
			logger.Debugf("Export is in lifecycle state %s", resp.Export.LifecycleState)
			return false, nil
		}
	}); err != nil {
		return nil, err
	}
	return export, nil
}

func (fsp *filesystemProvisioner) getFileSystemByDisplayName(ctx context.Context, ad, displayName string) (*filestorage.FileSystemSummary, error) {
	resp, err := fsp.client.FSS().ListFileSystems(ctx, filestorage.ListFileSystemsRequest{
		CompartmentId:      common.String(fsp.client.CompartmentOCID()),
		AvailabilityDomain: &ad,
		DisplayName:        &displayName,
	})
	if err != nil {
		return nil, err
	}

	if len(resp.Items) > 1 {
		return nil, errors.Errorf("found more than one file system with display name %q", displayName)
	}

	if len(resp.Items) == 1 {
		return &resp.Items[0], nil
	}

	return nil, errNotFound
}

func (fsp *filesystemProvisioner) getOrCreateFileSystem(ctx context.Context, logger *zap.SugaredLogger, ad, displayName string) (*filestorage.FileSystem, error) {
	fs, err := fsp.getFileSystemByDisplayName(ctx, ad, displayName)
	if err != nil && err != errNotFound {
		return nil, err
	}
	if fs != nil {
		return fsp.awaitFileSystem(ctx, logger, *fs.Id)
	}

	resp, err := fsp.client.FSS().CreateFileSystem(ctx, filestorage.CreateFileSystemRequest{
		CreateFileSystemDetails: filestorage.CreateFileSystemDetails{
			AvailabilityDomain: &ad,
			CompartmentId:      common.String(fsp.client.CompartmentOCID()),
			DisplayName:        &displayName,
		},
	})
	if err != nil {
		return nil, err
	}

	logger.With("fileSystemID", *resp.FileSystem.Id).Info("Created filesystem")

	return fsp.awaitFileSystem(ctx, logger, *resp.FileSystem.Id)
}

// findExport looks for an existing export with the same filesystem ID, export set ID, and path.
// NOTE: No two non-'DELETED' export resources in the same export set can reference the same file system.
func (fsp *filesystemProvisioner) findExport(ctx context.Context, fsID, exportSetID string) (*filestorage.ExportSummary, error) {
	var page *string
	for {
		resp, err := fsp.client.FSS().ListExports(ctx, filestorage.ListExportsRequest{
			CompartmentId: common.String(fsp.client.CompartmentOCID()),
			FileSystemId:  &fsID,
			ExportSetId:   &exportSetID,
			Page:          page,
		})
		if err != nil {
			return nil, err
		}
		for _, export := range resp.Items {
			if export.LifecycleState == filestorage.ExportSummaryLifecycleStateCreating ||
				export.LifecycleState == filestorage.ExportSummaryLifecycleStateActive {
				return &export, nil
			}
		}
		if page = resp.OpcNextPage; resp.OpcNextPage == nil {
			break
		}
	}

	return nil, errNotFound
}

func (fsp *filesystemProvisioner) getOrCreateExport(ctx context.Context, logger *zap.SugaredLogger, fsID, exportSetID string) (*filestorage.Export, error) {
	export, err := fsp.findExport(ctx, fsID, exportSetID)
	if err != nil && err != errNotFound {
		return nil, err
	}
	if export != nil {
		return fsp.awaitExport(ctx, logger, *export.Id)
	}

	path := "/" + fsID

	// If export doesn't already exist create it.
	resp, err := fsp.client.FSS().CreateExport(ctx, filestorage.CreateExportRequest{
		CreateExportDetails: filestorage.CreateExportDetails{
			ExportSetId:  &exportSetID,
			FileSystemId: &fsID,
			Path:         &path,
		},
	})
	if err != nil {
		return nil, err
	}

	logger.With("exportID", *resp.Export.Id).Info("Created Export")
	return fsp.awaitExport(ctx, logger, *resp.Export.Id)
}

func (fsp *filesystemProvisioner) Provision(options controller.VolumeOptions, ad *identity.AvailabilityDomain) (*v1.PersistentVolume, error) {
	ctx := context.Background()
	fsDisplayName := fmt.Sprintf("%s%s", provisioner.GetPrefix(), options.PVC.UID)
	logger := fsp.logger.With(
		"availabilityDomain", ad,
		"fileSystemDisplayName", fsDisplayName,
	)

	region, ok := os.LookupEnv("OCI_SHORT_REGION")
	if !ok {
		metadata, err := fsp.metadata.Get()
		if err != nil {
			return nil, err
		}
		region = metadata.Region
	}

	target, err := fsp.getOrCreateMountTarget(ctx, options.Parameters[MntTargetID], *ad.Name, options.Parameters[SubnetID])
	if err != nil {
		logger.With(zap.Error(err)).Error("Failed to retrieve mount target")
		return nil, err
	}

	logger = logger.With("mountTargetID", *target.Id)

	if len(target.PrivateIpIds) == 0 {
		logger.Error("Failed to find private IP IDs associated with the Mount Target")
		return nil, errors.Errorf("mount target has no private IP IDs")
	}

	if target.ExportSetId == nil {
		logger.Error("Mount target has no export set associated with it")
		return nil, errors.Errorf("mount target has no export set associated with it")
	}

	// Randomnly select IP address associated with the mount target to use
	// for attachment.
	var serverIP string
	{
		id := target.PrivateIpIds[rand.Int()%len(target.PrivateIpIds)]
		getPrivateIPResponse, err := fsp.client.VCN().GetPrivateIp(ctx, core.GetPrivateIpRequest{PrivateIpId: &id})
		if err != nil {
			logger.With(zap.Error(err), "privateIPID", id).Error("Failed to retrieve IP address for mount target")
			return nil, err
		}
		serverIP = *getPrivateIPResponse.PrivateIp.IpAddress
	}

	logger.With("privateIP", serverIP).Info("Selected Mount Target Private IP.")

	logger.Info("Creating FileSystem")
	var fsID string
	{
		fs, err := fsp.getOrCreateFileSystem(ctx, logger, *ad.Name, fsDisplayName)
		if err != nil {
			return nil, err
		}
		fsID = *fs.Id

		logger = logger.With("fileSystemID", fsID)
	}

	logger.Info("Creating export set")
	export, err := fsp.getOrCreateExport(ctx, logger, fsID, *target.ExportSetId)
	if err != nil {
		logger.With(zap.Error(err)).Error("Failed to create export.")
		return nil, err
	}

	return &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: fsID,
			Annotations: map[string]string{
				ociVolumeID: fsID,
				ociExportID: *export.Id,
			},
			Labels: map[string]string{plugin.LabelZoneRegion: region},
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeReclaimPolicy: options.PersistentVolumeReclaimPolicy,
			AccessModes:                   options.PVC.Spec.AccessModes,
			//NOTE: fs storage doesn't enforce quota, capacity is meaningless here.
			Capacity: v1.ResourceList{
				v1.ResourceName(v1.ResourceStorage): options.PVC.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)],
			},
			PersistentVolumeSource: v1.PersistentVolumeSource{
				NFS: &v1.NFSVolumeSource{
					Server:   serverIP,
					Path:     *export.Path,
					ReadOnly: false, // TODO: Should this be based on the AccessModes?
				},
			},
			MountOptions: options.MountOptions,
		},
	}, nil
}

// Delete destroys a OCI volume created by Provision
func (fsp *filesystemProvisioner) Delete(volume *v1.PersistentVolume) error {
	ctx := context.Background()
	exportID, ok := volume.Annotations[ociExportID]
	if !ok {
		return errors.Errorf("%q annotation not found on PV", ociExportID)
	}

	filesystemID, ok := volume.Annotations[ociVolumeID]
	if !ok {
		return errors.Errorf("%q annotation not found on PV", ociVolumeID)
	}

	logger := fsp.logger.With(
		"volumeOCID", volume,
		"exportOCID", exportID,
	)

	logger.Info("Deleting export")
	if _, err := fsp.client.FSS().DeleteExport(ctx, filestorage.DeleteExportRequest{
		ExportId: &exportID,
	}); err != nil {
		if !provisioner.IsNotFound(err) {
			logger.With(zap.Error(err)).Error("Failed to delete export")
			return err
		}
		logger.With(zap.Error(err)).Info("ExportID not found. Unable to delete it")
	}

	logger.Info("Deleting File System.")

	_, err := fsp.client.FSS().DeleteFileSystem(ctx, filestorage.DeleteFileSystemRequest{
		FileSystemId: &filesystemID,
	})
	if err != nil {
		if !provisioner.IsNotFound(err) {
			return err
		}
		logger.Info("File System not found. Unable to delete it")
	}
	return nil
}
