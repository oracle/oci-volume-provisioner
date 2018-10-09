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
	"time"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"go.uber.org/zap"

	"github.com/kubernetes-incubator/external-storage/lib/controller"
	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/core"
	fss "github.com/oracle/oci-go-sdk/filestorage"
	"github.com/oracle/oci-go-sdk/identity"
	"github.com/pkg/errors"

	"github.com/oracle/oci-volume-provisioner/pkg/oci/client"
	"github.com/oracle/oci-volume-provisioner/pkg/provisioner"
	"github.com/oracle/oci-volume-provisioner/pkg/provisioner/plugin"

	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	ociVolumeID = "volume.beta.kubernetes.io/oci-volume-id"
	ociExportID = "volume.beta.kubernetes.io/oci-export-id"
	// AnnotationMountTargetID configures the mount target to use when
	// provisioning a FSS volume
	AnnotationMountTargetID = "volume.beta.kubernetes.io/oci-mount-target-id"

	// MntTargetID is the name of the parameter which hold the target mount target ocid.
	MntTargetID = "mntTargetId"
)

const (
	defaultTimeout  = 5 * time.Minute
	defaultInterval = 5 * time.Second
)

// filesystemProvisioner is the internal provisioner for OCI filesystem volumes
type filesystemProvisioner struct {
	client client.ProvisionerClient

	// region is the oci region in which the kubernetes cluster is located.
	region string

	logger *zap.SugaredLogger
}

var _ plugin.ProvisionerPlugin = &filesystemProvisioner{}

var (
	errNoCandidateFound = errors.New("no candidate mount targets found")
	errNotFound         = errors.New("not found")
)

// NewFilesystemProvisioner creates a new file system provisioner that creates
// filesystems using OCI File System Service.
func NewFilesystemProvisioner(logger *zap.SugaredLogger, client client.ProvisionerClient, region string) plugin.ProvisionerPlugin {
	return &filesystemProvisioner{
		client: client,
		region: region,
		logger: logger.With(
			"compartmentID", client.CompartmentOCID(),
			"tenancyID", client.TenancyOCID(),
		),
	}
}

func (fsp *filesystemProvisioner) awaitFileSystem(ctx context.Context, logger *zap.SugaredLogger, id string) (*fss.FileSystem, error) {
	logger.Infof("Waiting for FileSystem to be in lifecycle state %q", fss.FileSystemLifecycleStateActive)

	var fs *fss.FileSystem
	err := wait.Poll(defaultInterval, defaultTimeout, func() (bool, error) {
		logger.Debug("Polling FileSystem lifecycle state")

		resp, err := fsp.client.FSS().GetFileSystem(ctx, fss.GetFileSystemRequest{
			FileSystemId: &id,
		})
		if err != nil {
			return false, err
		}

		fs = &resp.FileSystem

		switch state := resp.LifecycleState; state {
		case fss.FileSystemLifecycleStateActive:
			logger.Infof("FileSystem is in lifecycle state %q", state)
			return true, nil
		case fss.FileSystemLifecycleStateDeleting, fss.FileSystemLifecycleStateDeleted:
			return false, errors.Errorf("file system %q is in lifecycle state %q", *fs.Id, state)
		default:
			logger.Debugf("FileSystem is in lifecycle state %q", state)
			return false, nil
		}
	})
	if err != nil {
		return nil, err
	}

	return fs, nil
}

func (fsp *filesystemProvisioner) awaitMountTarget(ctx context.Context, logger *zap.SugaredLogger, id string) (*fss.MountTarget, error) {
	logger.Infof("Waiting for MountTarget to be in lifecycle state %q", fss.MountTargetLifecycleStateActive)

	var mt *fss.MountTarget
	if err := wait.Poll(defaultInterval, defaultTimeout, func() (bool, error) {
		logger.Debug("Polling MountTarget lifecycle state")

		resp, err := fsp.client.FSS().GetMountTarget(ctx, fss.GetMountTargetRequest{MountTargetId: &id})
		if err != nil {
			return false, err
		}

		mt = &resp.MountTarget

		switch state := resp.LifecycleState; state {
		case fss.MountTargetLifecycleStateActive:
			logger.Infof("Mount target is in lifecycle state %q")
			return true, nil
		case fss.MountTargetLifecycleStateFailed,
			fss.MountTargetLifecycleStateDeleting,
			fss.MountTargetLifecycleStateDeleted:
			logger.With("lifecycleState", state).Error("MountTarget will not become ACTIVE")
			return false, fmt.Errorf("mount target %q is in lifecycle state %q and will not become ACTIVE", *resp.MountTarget.Id, state)
		default:
			logger.Debugf("Mount target is in lifecycle state %q", state)
			return false, nil
		}
	}); err != nil {
		return nil, err
	}
	return mt, nil
}

func (fsp *filesystemProvisioner) awaitExport(ctx context.Context, logger *zap.SugaredLogger, id string) (*fss.Export, error) {
	logger.Info("Waiting for Export to be in lifecycle state ACTIVE")

	var export *fss.Export
	if err := wait.Poll(defaultInterval, defaultTimeout, func() (bool, error) {
		logger.Debug("Polling export lifecycle state")

		resp, err := fsp.client.FSS().GetExport(ctx, fss.GetExportRequest{ExportId: &id})
		if err != nil {
			return false, err
		}

		export = &resp.Export

		switch state := resp.LifecycleState; state {
		case fss.ExportLifecycleStateActive:
			logger.Infof("Export is in lifecycle state %q", state)
			return true, nil
		case fss.ExportLifecycleStateDeleting, fss.ExportLifecycleStateDeleted:
			logger.Errorf("Export is in lifecycle state %q", state)
			return false, fmt.Errorf("export %q is in lifecycle state %q", *resp.Export.Id, state)
		default:
			logger.Debugf("Export is in lifecycle state %q", state)
			return false, nil
		}
	}); err != nil {
		return nil, err
	}
	return export, nil
}

func (fsp *filesystemProvisioner) getFileSystemByDisplayName(ctx context.Context, ad, displayName string) (*fss.FileSystemSummary, error) {
	resp, err := fsp.client.FSS().ListFileSystems(ctx, fss.ListFileSystemsRequest{
		CompartmentId:      common.String(fsp.client.CompartmentOCID()),
		AvailabilityDomain: &ad,
		DisplayName:        &displayName,
	})
	if err != nil {
		return nil, err
	}

	switch count := len(resp.Items); {
	case count == 1:
		return &resp.Items[0], nil
	case count > 1:
		return nil, errors.Errorf("found more than one file system with display name %q", displayName)
	}

	return nil, errNotFound
}

func (fsp *filesystemProvisioner) getOrCreateFileSystem(ctx context.Context, logger *zap.SugaredLogger, ad, displayName string) (*fss.FileSystem, error) {
	fs, err := fsp.getFileSystemByDisplayName(ctx, ad, displayName)
	if err != nil && err != errNotFound {
		return nil, err
	}
	if fs != nil {
		return fsp.awaitFileSystem(ctx, logger, *fs.Id)
	}

	resp, err := fsp.client.FSS().CreateFileSystem(ctx, fss.CreateFileSystemRequest{
		CreateFileSystemDetails: fss.CreateFileSystemDetails{
			CompartmentId:      common.String(fsp.client.CompartmentOCID()),
			AvailabilityDomain: &ad,
			DisplayName:        &displayName,
		},
	})
	if err != nil {
		return nil, err
	}

	logger.With("fileSystemID", *resp.FileSystem.Id).Info("Created FileSystem")

	return fsp.awaitFileSystem(ctx, logger, *resp.FileSystem.Id)
}

// findExport looks for an existing export with the same filesystem ID, export set ID, and path.
// NOTE: No two non-'DELETED' export resources in the same export set can reference the same file system.
func (fsp *filesystemProvisioner) findExport(ctx context.Context, fsID, exportSetID string) (*fss.ExportSummary, error) {
	var page *string
	for {
		resp, err := fsp.client.FSS().ListExports(ctx, fss.ListExportsRequest{
			CompartmentId: common.String(fsp.client.CompartmentOCID()),
			FileSystemId:  &fsID,
			ExportSetId:   &exportSetID,
			Page:          page,
		})
		if err != nil {
			return nil, err
		}
		for _, export := range resp.Items {
			if export.LifecycleState == fss.ExportSummaryLifecycleStateCreating ||
				export.LifecycleState == fss.ExportSummaryLifecycleStateActive {
				return &export, nil
			}
		}
		if page = resp.OpcNextPage; resp.OpcNextPage == nil {
			break
		}
	}

	return nil, errNotFound
}

func (fsp *filesystemProvisioner) getOrCreateExport(ctx context.Context, logger *zap.SugaredLogger, fsID, exportSetID string) (*fss.Export, error) {
	export, err := fsp.findExport(ctx, fsID, exportSetID)
	if err != nil && err != errNotFound {
		return nil, err
	}
	if export != nil {
		return fsp.awaitExport(ctx, logger, *export.Id)
	}

	path := "/" + fsID

	// If export doesn't already exist create it.
	resp, err := fsp.client.FSS().CreateExport(ctx, fss.CreateExportRequest{
		CreateExportDetails: fss.CreateExportDetails{
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

// getMountTargetID retrieves MountTarget OCID if provided.
func getMountTargetID(opts controller.VolumeOptions) string {
	if opts.PVC != nil {
		if mtID := opts.PVC.Annotations[AnnotationMountTargetID]; mtID != "" {
			return mtID
		}
	}
	return opts.Parameters[MntTargetID]
}

// isReadOnly determines if the given slice of PersistentVolumeAccessModes
// permits mounting as read only.
func isReadOnly(modes []v1.PersistentVolumeAccessMode) bool {
	for _, mode := range modes {
		if mode == v1.ReadWriteMany || mode == v1.ReadWriteOnce {
			return false
		}
	}
	return true
}

func (fsp *filesystemProvisioner) Provision(options controller.VolumeOptions, ad *identity.AvailabilityDomain) (*v1.PersistentVolume, error) {
	ctx := context.Background()
	fsDisplayName := fmt.Sprintf("%s%s", provisioner.GetPrefix(), options.PVC.UID)
	logger := fsp.logger.With(
		"availabilityDomain", ad,
		"fileSystemDisplayName", fsDisplayName,
	)

	// Require that a user provides a MountTarget ID.
	mtID := getMountTargetID(options)
	if mtID == "" {
		return nil, errors.New("no mount target ID provided (via PVC annotation nor StorageClass option)")
	}

	logger = logger.With("mountTargetID", mtID)

	// Wait for MountTarget to be ACTIVE.
	target, err := fsp.awaitMountTarget(ctx, logger, mtID)
	if err != nil {
		logger.With(zap.Error(err)).Error("Failed to retrieve mount target")
		return nil, err
	}

	// Ensure MountTarget required fields are set.
	if len(target.PrivateIpIds) == 0 {
		logger.Error("Failed to find private IPs associated with the Mount Target")
		return nil, errors.Errorf("mount target has no associated private IPs")
	}
	if target.ExportSetId == nil {
		logger.Error("Mount target has no export set associated with it")
		return nil, errors.Errorf("mount target has no export set associated with it")
	}

	// Randomly select a MountTarget IP address to attach to.
	var ip string
	{

		id := target.PrivateIpIds[rand.Int()%len(target.PrivateIpIds)]
		logger = logger.With("privateIPID", id)
		resp, err := fsp.client.VCN().GetPrivateIp(ctx, core.GetPrivateIpRequest{PrivateIpId: &id})
		if err != nil {
			logger.With(zap.Error(err)).Error("Failed to retrieve IP address for mount target")
			return nil, err
		}
		if resp.PrivateIp.IpAddress == nil {
			logger.Error("PrivateIp has no IpAddress")
			return nil, errors.Errorf("PrivateIp %q associated with MountTarget %q has no IpAddress", id, mtID)
		}
		ip = *resp.PrivateIp.IpAddress
	}
	logger = logger.With("privateIP", ip)

	logger.Info("Creating FileSystem")
	fs, err := fsp.getOrCreateFileSystem(ctx, logger, *ad.Name, fsDisplayName)
	if err != nil {
		return nil, err
	}
	logger = logger.With("fileSystemID", *fs.Id)

	logger.Info("Creating Export")
	export, err := fsp.getOrCreateExport(ctx, logger, *fs.Id, *target.ExportSetId)
	if err != nil {
		logger.With(zap.Error(err)).Error("Failed to create export.")
		return nil, err
	}

	logger.With("exportID", *export.Id).Info("All OCI resources provisioned")

	return &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: *fs.Id,
			Annotations: map[string]string{
				ociVolumeID: *fs.Id,
				ociExportID: *export.Id,
			},
			Labels: map[string]string{plugin.LabelZoneRegion: fsp.region},
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
					Server:   ip,
					Path:     *export.Path,
					ReadOnly: isReadOnly(options.PVC.Spec.AccessModes),
				},
			},
			MountOptions: options.MountOptions,
		},
	}, nil
}

// Delete terminates the OCI resources associated with the given PVC.
func (fsp *filesystemProvisioner) Delete(volume *v1.PersistentVolume) error {
	ctx := context.Background()
	exportID := volume.Annotations[ociExportID]
	if exportID == "" {
		return errors.Errorf("%q annotation not found on PV", ociExportID)
	}

	filesystemID := volume.Annotations[ociVolumeID]
	if filesystemID == "" {
		return errors.Errorf("%q annotation not found on PV", ociVolumeID)
	}

	logger := fsp.logger.With(
		"volumeOCID", volume,
		"exportOCID", exportID,
	)

	logger.Info("Deleting export")
	if _, err := fsp.client.FSS().DeleteExport(ctx, fss.DeleteExportRequest{
		ExportId: &exportID,
	}); err != nil {
		if !provisioner.IsNotFound(err) {
			logger.With(zap.Error(err)).Error("Failed to delete export")
			return err
		}
		logger.With(zap.Error(err)).Info("Export not found. Unable to delete it")
	}

	logger.Info("Deleting File System")
	_, err := fsp.client.FSS().DeleteFileSystem(ctx, fss.DeleteFileSystemRequest{
		FileSystemId: &filesystemID,
	})
	if err != nil {
		if !provisioner.IsNotFound(err) {
			return err
		}
		logger.Info("FileSystem not found. Unable to delete it")
	}
	return nil
}
