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
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/golang/glog"
	"github.com/kubernetes-incubator/external-storage/lib/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"

	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/core"
	"github.com/oracle/oci-go-sdk/identity"
	"github.com/oracle/oci-volume-provisioner/pkg/oci/client"
	"github.com/oracle/oci-volume-provisioner/pkg/provisioner/plugin"
)

const (
	ociVolumeID            = "ociVolumeID"
	volumePrefixEnvVarName = "OCI_VOLUME_NAME_PREFIX"
	fsType                 = "fsType"
)

// blockProvisioner is the internal provisioner for OCI block volumes
type blockProvisioner struct {
	client client.ProvisionerClient
}

var _ plugin.ProvisionerPlugin = &blockProvisioner{}

// NewBlockProvisioner creates a new instance of the block storage provisioner
func NewBlockProvisioner(client client.ProvisionerClient) plugin.ProvisionerPlugin {
	return &blockProvisioner{
		client: client,
	}
}

func mapVolumeIDToName(volumeID string) string {
	return strings.Split(volumeID, ".")[4]
}

func resolveFSType(options controller.VolumeOptions) string {
	fs := "ext4" // default to ext4
	if fsType, ok := options.Parameters[fsType]; ok {
		fs = fsType
	}
	return fs
}

func newCreateVolumeDetails(adName, compartmentOCID, volumeNamePrefix, volumeName string, volSizeMB int) core.CreateVolumeDetails {
	return core.CreateVolumeDetails{
		AvailabilityDomain: common.String(adName),
		CompartmentId:      common.String(compartmentOCID),
		DisplayName:        common.String(fmt.Sprintf("%s%s", volumeNamePrefix, volumeName)),
		SizeInMBs:          common.Int(volSizeMB),
	}
}

func roundUpSize(volumeSizeBytes int64, allocationUnitBytes int64) int64 {
	return (volumeSizeBytes + allocationUnitBytes - 1) / allocationUnitBytes
}

// Provision creates an OCI block volume acording to the spec
func (block *blockProvisioner) Provision(options controller.VolumeOptions,
	availabilityDomain *identity.AvailabilityDomain) (*v1.PersistentVolume, error) {
	for _, accessMode := range options.PVC.Spec.AccessModes {
		if accessMode != v1.ReadWriteOnce {
			return nil, fmt.Errorf("invalid access mode %v specified. Only %v is supported",
				accessMode,
				v1.ReadWriteOnce)
		}
	}

	// Calculate the size
	volSize := options.PVC.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)]
	volSizeBytes := volSize.Value()
	glog.Infof("Volume size (bytes): %v", volSizeBytes)
	volSizeMB := int(roundUpSize(volSizeBytes, 1024*1024))

	glog.Infof("Creating volume size=%v AD=%s compartmentOCID=%q", volSizeMB, availabilityDomain.Name, block.client.CompartmentOCID())

	// TODO: Consider OpcRetryToken
	details := newCreateVolumeDetails(*availabilityDomain.Name, block.client.CompartmentOCID(), os.Getenv(volumePrefixEnvVarName), options.PVC.Name, volSizeMB)
	request := core.CreateVolumeRequest{CreateVolumeDetails: details}
	ctx, cancel := context.WithTimeout(block.client.Context(), block.client.Timeout())
	defer cancel()
	newVolume, err := block.client.BlockStorage().CreateVolume(ctx, request)
	if err != nil {
		return nil, err
	}

	volumeName := mapVolumeIDToName(*newVolume.Id)
	filesystemType := resolveFSType(options)

	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: volumeName,
			Annotations: map[string]string{
				ociVolumeID: *newVolume.Id,
			},
			Labels: map[string]string{},
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeReclaimPolicy: options.PersistentVolumeReclaimPolicy,
			AccessModes:                   options.PVC.Spec.AccessModes,
			Capacity: v1.ResourceList{
				v1.ResourceName(v1.ResourceStorage): options.PVC.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)],
			},
			PersistentVolumeSource: v1.PersistentVolumeSource{
				FlexVolume: &v1.FlexVolumeSource{
					Driver: "oracle/oci",
					FSType: filesystemType,
				},
			},
		},
	}

	return pv, nil
}

// Delete destroys a OCI volume created by Provision
func (block *blockProvisioner) Delete(volume *v1.PersistentVolume) error {
	glog.Infof("Deleting volume %v with volumeId %v", volume, volume.Annotations[ociVolumeID])

	volID, ok := volume.Annotations[ociVolumeID]
	if !ok {
		return errors.New("volumeid annotation not found on PV")
	}

	request := core.DeleteVolumeRequest{VolumeId: common.String(volID)}
	ctx, cancel := context.WithTimeout(block.client.Context(), block.client.Timeout())
	defer cancel()
	return block.client.BlockStorage().DeleteVolume(ctx, request)
}
