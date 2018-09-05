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

	"github.com/golang/glog"
	"github.com/kubernetes-incubator/external-storage/lib/controller"
	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/core"
	"github.com/oracle/oci-go-sdk/filestorage"
	"github.com/oracle/oci-go-sdk/identity"
	"github.com/pkg/errors"

	"github.com/oracle/oci-volume-provisioner/pkg/oci/client"
	"github.com/oracle/oci-volume-provisioner/pkg/provisioner"
	"github.com/oracle/oci-volume-provisioner/pkg/provisioner/plugin"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ociVolumeID = "volume.beta.kubernetes.io/oci-volume-id"
	ociExportID = "volume.beta.kubernetes.io/oci-export-id"
	fsType      = "fsType"
	subnetID    = "subnetId"
	mntTargetID = "mntTargetId"
)

// filesystemProvisioner is the internal provisioner for OCI filesystem volumes
type filesystemProvisioner struct {
	client client.ProvisionerClient
}

var _ plugin.ProvisionerPlugin = &filesystemProvisioner{}

// NewFilesystemProvisioner creates a new file system provisioner that creates
// filesystems using OCI File System Service.
func NewFilesystemProvisioner(client client.ProvisionerClient) plugin.ProvisionerPlugin {
	return &filesystemProvisioner{
		client: client,
	}
}

// getMountTargetFromID retrieves mountTarget from given mountTargetID
func (fsp *filesystemProvisioner) getMountTargetFromID(ctx context.Context, mountTargetID string) (*filestorage.MountTarget, error) {
	ctx, cancel := context.WithTimeout(ctx, fsp.client.Timeout())
	defer cancel()

	resp, err := fsp.client.FSS().GetMountTarget(ctx, filestorage.GetMountTargetRequest{
		MountTargetId: &mountTargetID,
	})
	if err != nil {
		glog.Errorf("Failed to retrieve mount point mountTargetId=%q: %v", mountTargetID, err)
		return nil, err
	}
	return &resp.MountTarget, nil
}

// listAllMountTargets retrieves all available mount targets
func (fsp *filesystemProvisioner) listAllMountTargets(ctx context.Context, ad string) ([]filestorage.MountTargetSummary, error) {
	var (
		page         *string
		mountTargets []filestorage.MountTargetSummary
	)
	// Check if there already is a mount target in the existing compartment
	for {
		ctx, cancel := context.WithTimeout(ctx, fsp.client.Timeout())
		defer cancel()
		resp, err := fsp.client.FSS().ListMountTargets(ctx, filestorage.ListMountTargetsRequest{
			AvailabilityDomain: &ad,
			CompartmentId:      common.String(fsp.client.CompartmentOCID()),
			Page:               page,
		})
		if err != nil {
			return nil, err
		}
		mountTargets = append(mountTargets, resp.Items...)
		if page = resp.OpcNextPage; resp.OpcNextPage == nil {
			break
		}
	}
	return mountTargets, nil
}

func (fsp *filesystemProvisioner) getOrCreateMountTarget(ctx context.Context, mtID string, ad string, subnetID string) (*filestorage.MountTarget, error) {
	if mtID != "" {
		// Mount target already specified in the configuration file, find it in the list of mount targets
		return fsp.getMountTargetFromID(ctx, mtID)
	}
	mountTargets, err := fsp.listAllMountTargets(ctx, ad)
	if err != nil {
		return nil, err
	}
	if len(mountTargets) != 0 {
		glog.V(4).Infof("Found mount targets to use")
		mntTargetSummary := mountTargets[rand.Int()%len(mountTargets)]
		target, err := fsp.getMountTargetFromID(ctx, *mntTargetSummary.Id)
		return target, err
	}
	ctx, cancel := context.WithTimeout(ctx, fsp.client.Timeout())
	defer cancel()
	// Mount target not created, create a new one
	resp, err := fsp.client.FSS().CreateMountTarget(ctx, filestorage.CreateMountTargetRequest{
		CreateMountTargetDetails: filestorage.CreateMountTargetDetails{
			AvailabilityDomain: &ad,
			SubnetId:           &subnetID,
			CompartmentId:      common.String(fsp.client.CompartmentOCID()),
			DisplayName:        common.String(fmt.Sprintf("%s%s", provisioner.GetPrefix(), "mnt")),
		},
	})
	if err != nil {
		return nil, err
	}
	return &resp.MountTarget, nil
}

func (fsp *filesystemProvisioner) Provision(options controller.VolumeOptions, ad *identity.AvailabilityDomain) (*v1.PersistentVolume, error) {
	ctx := context.Background()
	// Create the FileSystem.
	var fsID string
	{
		ctx, cancel := context.WithTimeout(ctx, fsp.client.Timeout())
		defer cancel()
		resp, err := fsp.client.FSS().CreateFileSystem(ctx, filestorage.CreateFileSystemRequest{
			CreateFileSystemDetails: filestorage.CreateFileSystemDetails{
				AvailabilityDomain: ad.Name,
				CompartmentId:      common.String(fsp.client.CompartmentOCID()),
				DisplayName:        common.String(fmt.Sprintf("%s%s", provisioner.GetPrefix(), options.PVC.Name)),
			},
		})
		if err != nil {
			glog.Errorf("Failed to create a file system options=%#v: %v", options, err)
			return nil, err
		}
		fsID = *resp.FileSystem.Id
	}

	target, err := fsp.getOrCreateMountTarget(ctx, options.Parameters[mntTargetID], *ad.Name, options.Parameters[subnetID])
	if err != nil {
		glog.Errorf("Failed to retrieve mount target: %s", err)
		return nil, err
	}

	glog.V(6).Infof("Creating export set")
	// Create the ExportSet.
	var exportSetID string
	{
		ctx, cancel := context.WithTimeout(ctx, fsp.client.Timeout())
		defer cancel()
		resp, err := fsp.client.FSS().CreateExport(ctx, filestorage.CreateExportRequest{
			CreateExportDetails: filestorage.CreateExportDetails{
				ExportSetId:  target.ExportSetId,
				FileSystemId: &fsID,
				Path:         common.String("/" + fsID),
			},
		})
		if err != nil {
			glog.Errorf("Failed to create export: %v", err)
			return nil, err
		}
		exportSetID = *resp.Export.Id
	}

	if len(target.PrivateIpIds) == 0 {
		glog.Errorf("Failed to find server IDs associated with the Mount Target (OCID %s) to provision a persistent volume", target.Id)
		return nil, errors.Errorf("failed to find server IDs associated with the Mount Target with OCID %q", target.Id)
	}

	// Get PrivateIP.
	var serverIP string
	{
		ctx, cancel := context.WithTimeout(ctx, fsp.client.Timeout())
		defer cancel()
		id := target.PrivateIpIds[rand.Int()%len(target.PrivateIpIds)]
		getPrivateIPResponse, err := fsp.client.VCN().GetPrivateIp(ctx, core.GetPrivateIpRequest{
			PrivateIpId: &id,
		})
		if err != nil {
			glog.Errorf("Failed to retrieve IP address for mount target privateIpID=%q: %v", id, err)
			return nil, err
		}
		serverIP = *getPrivateIPResponse.PrivateIp.IpAddress
	}

	glog.Infof("Creating persistent volume on mount target with private IP address %s", serverIP)
	return &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: fsID,
			Annotations: map[string]string{
				ociVolumeID: fsID,
				ociExportID: exportSetID,
			},
			Labels: map[string]string{},
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeReclaimPolicy: options.PersistentVolumeReclaimPolicy,
			AccessModes:                   options.PVC.Spec.AccessModes,
			//FIXME: fs storage doesn't enforce quota, capacity is meaningless here.
			Capacity: v1.ResourceList{
				v1.ResourceName(v1.ResourceStorage): options.PVC.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)],
			},
			PersistentVolumeSource: v1.PersistentVolumeSource{
				NFS: &v1.NFSVolumeSource{
					// Randomnly select IP address associated with the mount target to use for attachment
					Server:   serverIP,
					Path:     "/" + fsID,
					ReadOnly: false,
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

	glog.Infof("Deleting export for filesystemID %v", filesystemID)
	ctx, cancel := context.WithTimeout(ctx, fsp.client.Timeout())
	defer cancel()
	if _, err := fsp.client.FSS().DeleteExport(ctx, filestorage.DeleteExportRequest{
		ExportId: &exportID,
	}); err != nil {
		if !provisioner.IsNotFound(err) {
			glog.Errorf("Failed to delete export exportID=%q: %v", exportID, err)
			return err
		}
		glog.Infof("ExportID %q was not found. Unable to delete it: %v", exportID, err)
	}

	ctx, cancel = context.WithTimeout(ctx, fsp.client.Timeout())
	defer cancel()

	glog.Infof("Deleting volume %v with FileSystemID %v", volume, filesystemID)
	_, err := fsp.client.FSS().DeleteFileSystem(ctx, filestorage.DeleteFileSystemRequest{
		FileSystemId: &filesystemID,
	})
	if err != nil {
		if !provisioner.IsNotFound(err) {
			return err
		}
		glog.Infof("FileSystemID %q was not found. Unable to delete it: %v", filesystemID, err)
	}
	return nil
}
