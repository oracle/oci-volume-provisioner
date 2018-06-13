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
	"os"

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
	volumePrefixEnvVarName = "OCI_VOLUME_NAME_PREFIX"
	fsType                 = "fsType"
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

func (filesystem *filesystemProvisioner) Provision(
	options controller.VolumeOptions,
	availabilityDomain *identity.AvailabilityDomain) (*v1.PersistentVolume, error) {

	ctx, cancel := context.WithTimeout(filesystem.client.Context(), filesystem.client.Timeout())
	defer cancel()

	response, err := filesystem.client.FileStorage().CreateFileSystem(ctx, filestorage.CreateFileSystemRequest{
		CreateFileSystemDetails: filestorage.CreateFileSystemDetails{
			AvailabilityDomain: availabilityDomain.Name,
			CompartmentId:      common.String(filesystem.client.CompartmentOCID()),
			DisplayName:        common.String(fmt.Sprintf("%s%s", os.Getenv(volumePrefixEnvVarName), options.PVC.Name)),
		},
	})
	if err != nil {
		glog.Errorf("Failed to create a volume:%#v, %s", options, err)
		return nil, err
	}

	return &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: *response.FileSystem.Id,
			Annotations: map[string]string{
				ociVolumeID: *response.FileSystem.Id,
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
				FlexVolume: &v1.FlexVolumeSource{
					Driver: plugin.OCIProvisionerName,
					FSType: "NFSv3",
				},
			},
		},
	}, nil
}

// Delete destroys a OCI volume created by Provision
func (filesystem *filesystemProvisioner) Delete(volume *v1.PersistentVolume) error {
	filesystemID, ok := volume.Annotations[ociVolumeID]
	if !ok {
		return errors.New("filesystemid annotation not found on PV")
	}
	glog.Infof("Deleting volume %v with filesystemID %v", volume, filesystemID)
	ctx, cancel := context.WithTimeout(filesystem.client.Context(), filesystem.client.Timeout())
	defer cancel()
	_, err := filesystem.client.FileStorage().DeleteFileSystem(ctx,
		filestorage.DeleteFileSystemRequest{
			FileSystemId: &filesystemID,
		})
	return err
}
