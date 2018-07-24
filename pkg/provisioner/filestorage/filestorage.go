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

package filestorage

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/kubernetes-incubator/external-storage/lib/controller"
	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/filestorage"
	"github.com/oracle/oci-go-sdk/identity"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"

	"github.com/oracle/oci-volume-provisioner/pkg/oci/client"
	"github.com/oracle/oci-volume-provisioner/pkg/provisioner/plugin"
)

const (
	ociVolumeID            = "ociVolumeID"
	ociExportID            = "ociExportID"
	volumePrefixEnvVarName = "OCI_VOLUME_NAME_PREFIX"
	fsType                 = "fsType"
	subnetID               = "subnetId"
	mntTargetID            = "mntTargetId"
)

// filesystemProvisioner is the internal provisioner for OCI filesystem volumes
type filesystemProvisioner struct {
	client client.ProvisionerClient
}

var _ plugin.ProvisionerPlugin = &filesystemProvisioner{}

// NewFilesystemProvisioner creates a new file system provisioner that creates
// filsystems using OCI file system service.
func NewFilesystemProvisioner(client client.ProvisionerClient) plugin.ProvisionerPlugin {
	return &filesystemProvisioner{
		client: client,
	}
}

// getMountTargetFromID retrieves mountTarget from given mountTargetID
func getMountTargetFromID(ctx context.Context, mountTargetID string, fileStorageClient client.FileStorage) *filestorage.MountTarget {
	responseMnt, err := fileStorageClient.GetMountTarget(ctx, filestorage.GetMountTargetRequest{
		MountTargetId: common.String(mountTargetID),
	})
	if err != nil {
		glog.Errorf("Failed to retrieve mount point: %s", err)
		return nil
	}
	return &responseMnt.MountTarget
}

func (filesystem *filesystemProvisioner) Provision(
	options controller.VolumeOptions,
	availabilityDomain *identity.AvailabilityDomain) (*v1.PersistentVolume, error) {

	ctx, cancel := context.WithTimeout(filesystem.client.Context(), filesystem.client.Timeout())
	defer cancel()

	fileStorageClient := filesystem.client.FileStorage()
	response, err := fileStorageClient.CreateFileSystem(ctx, filestorage.CreateFileSystemRequest{
		CreateFileSystemDetails: filestorage.CreateFileSystemDetails{
			AvailabilityDomain: availabilityDomain.Name,
			CompartmentId:      common.String(filesystem.client.CompartmentOCID()),
			DisplayName:        common.String(fmt.Sprintf("%s%s", os.Getenv(volumePrefixEnvVarName), options.PVC.Name)),
		},
	})
	if err != nil {
		glog.Errorf("Failed to create a file system storage:%#v, %s", options, err)
		return nil, err
	}

	mntTargetResp := filestorage.MountTarget{}
	if options.Parameters[mntTargetID] == "" {
		// Check if there there already is a mount target in the existing compartment
		glog.Infof("Looking up existing mount targets")
		responseListMnt, err := fileStorageClient.ListMountTargets(ctx, filestorage.ListMountTargetsRequest{
			AvailabilityDomain: availabilityDomain.Name,
			CompartmentId:      common.String(filesystem.client.CompartmentOCID()),
		})
		if err != nil {
			glog.Errorf("Failed to list mount targets:%#v, %s", options, err)
			return nil, err
		}
		if len(responseListMnt.Items) != 0 {
			glog.Infof("Found mount targets to use")
			rand.Seed(time.Now().Unix())
			mntTargetSummary := responseListMnt.Items[rand.Int()%len(responseListMnt.Items)]
			mntTargetResp = *getMountTargetFromID(ctx, *mntTargetSummary.Id, fileStorageClient)
		} else {
			// Mount target not created, create a new one
			responseMnt, err := fileStorageClient.CreateMountTarget(ctx, filestorage.CreateMountTargetRequest{
				CreateMountTargetDetails: filestorage.CreateMountTargetDetails{
					AvailabilityDomain: availabilityDomain.Name,
					SubnetId:           common.String(options.Parameters[subnetID]),
					CompartmentId:      common.String(filesystem.client.CompartmentOCID()),
					DisplayName:        common.String(fmt.Sprintf("%s%s", os.Getenv(volumePrefixEnvVarName), "mnt")),
				},
			})
			if err != nil {
				glog.Errorf("Failed to create a mount target:%#v, %s", options, err)
				return nil, err
			}
			mntTargetResp = responseMnt.MountTarget
		}
	} else {
		// Mount target already specified in the configuration file, find it in the list of mount targets
		mntTargetResp = *getMountTargetFromID(ctx, options.Parameters[mntTargetID], fileStorageClient)
	}

	glog.Infof("Creating export set")
	createExportResponse, err := fileStorageClient.CreateExport(ctx, filestorage.CreateExportRequest{
		CreateExportDetails: filestorage.CreateExportDetails{
			ExportSetId:  mntTargetResp.ExportSetId,
			FileSystemId: response.FileSystem.Id,
			Path:         common.String("/"),
		},
	})

	if err != nil {
		glog.Errorf("Failed to create export:%s", err)
		return nil, err
	}
	mntTargetSubnetIDPtr := ""
	if mntTargetResp.SubnetId != nil {
		mntTargetSubnetIDPtr = *mntTargetResp.SubnetId
	}
	glog.Infof("Creating persistent volume")
	return &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: *response.FileSystem.Id,
			Annotations: map[string]string{
				ociVolumeID: *response.FileSystem.Id,
				ociExportID: *createExportResponse.Id,
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
					Server:   mntTargetSubnetIDPtr,
					Path:     "/",
					ReadOnly: false,
				},
			},
		},
	}, nil
}

// Delete destroys a OCI volume created by Provision
func (filesystem *filesystemProvisioner) Delete(volume *v1.PersistentVolume) error {
	exportID, ok := volume.Annotations[ociExportID]
	if !ok {
		return errors.New("Export ID annotation not found on PV")
	}
	filesystemID, ok := volume.Annotations[ociVolumeID]
	if !ok {
		return errors.New("Filesystem ID annotation not found on PV")
	}
	ctx, cancel := context.WithTimeout(filesystem.client.Context(), filesystem.client.Timeout())
	defer cancel()
	glog.Infof("Deleting export for filesystemID %v", filesystemID)
	_, err := filesystem.client.FileStorage().DeleteExport(ctx,
		filestorage.DeleteExportRequest{
			ExportId: &exportID,
		})
	if err != nil {
		glog.Errorf("Failed to delete export:%s, %s", exportID, err)
		return err
	}
	glog.Infof("Deleting volume %v with filesystemID %v", volume, filesystemID)
	_, err = filesystem.client.FileStorage().DeleteFileSystem(ctx,
		filestorage.DeleteFileSystemRequest{
			FileSystemId: &filesystemID,
		})
	return err
}
