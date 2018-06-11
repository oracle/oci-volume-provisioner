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
	"net/http"
	"os"
	"strings"

	"github.com/golang/glog"
	"github.com/kubernetes-incubator/external-storage/lib/controller"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"

	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/core"
	"github.com/oracle/oci-go-sdk/identity"
	"github.com/oracle/oci-volume-provisioner/pkg/oci/client"
	"github.com/oracle/oci-volume-provisioner/pkg/provisioner/plugin"

	"github.com/oracle/oci-volume-provisioner/pkg/oci/instancemeta"
)

const (
	ociVolumeID            = "ociVolumeID"
	ociVolumeBackupID      = "volume.beta.kubernetes.io/oci-volume-source"
	volumePrefixEnvVarName = "OCI_VOLUME_NAME_PREFIX"
	fsType                 = "fsType"
)

// blockProvisioner is the internal provisioner for OCI block volumes
type blockProvisioner struct {
	client                client.ProvisionerClient
	metadata              instancemeta.Interface
	volumeRoundingEnabled bool
	minVolumeSizeMB       int
}

var _ plugin.ProvisionerPlugin = &blockProvisioner{}

// NewBlockProvisioner creates a new instance of the block storage provisioner
func NewBlockProvisioner(client client.ProvisionerClient, metadata instancemeta.Interface, volumeRoundingEnabled bool, minVolumeSizeMB int) plugin.ProvisionerPlugin {
	return &blockProvisioner{
		client:                client,
		metadata:              metadata,
		volumeRoundingEnabled: volumeRoundingEnabled,
		minVolumeSizeMB:       minVolumeSizeMB,
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

func roundUpSize(volumeSizeBytes int64, allocationUnitBytes int64) int64 {
	return (volumeSizeBytes + allocationUnitBytes - 1) / allocationUnitBytes
}

// Provision creates an OCI block volume
func (block *blockProvisioner) Provision(options controller.VolumeOptions, ad *identity.AvailabilityDomain) (*v1.PersistentVolume, error) {
	for _, accessMode := range options.PVC.Spec.AccessModes {
		if accessMode != v1.ReadWriteOnce {
			return nil, fmt.Errorf("invalid access mode %v specified. Only %v is supported", accessMode, v1.ReadWriteOnce)
		}
	}

	// Calculate the volume size
	capacity, ok := options.PVC.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)]
	if !ok {
		return nil, fmt.Errorf("could not determine volume size for PVC")
	}

	volSizeMB := int(roundUpSize(capacity.Value(), 1024*1024))
	glog.Infof("Volume size: %dMB", volSizeMB)

	if block.volumeRoundingEnabled && volSizeMB < block.minVolumeSizeMB {
		glog.Warningf("PVC requested storage less than %dMi. Rounding up to ensure volume creation", block.minVolumeSizeMB)
		newVolumeSize, err := resource.ParseQuantity(fmt.Sprintf("%dMi", block.minVolumeSizeMB))
		if err != nil {
			return nil, err
		}

		volSizeMB = block.minVolumeSizeMB
		capacity = newVolumeSize
	}

	glog.Infof("Creating volume size=%v AD=%s compartmentOCID=%q", volSizeMB, *ad.Name, block.client.CompartmentOCID())

	volumeDetails := core.CreateVolumeDetails{
		AvailabilityDomain: ad.Name,
		CompartmentId:      common.String(block.client.CompartmentOCID()),
		DisplayName:        common.String(fmt.Sprintf("%s%s", os.Getenv(volumePrefixEnvVarName), options.PVC.Name)),
		SizeInMBs:          common.Int(volSizeMB),
	}

	if value, ok := options.PVC.Annotations[ociVolumeBackupID]; ok {
		glog.Infof("Creating volume from backup ID %s", value)
		volumeDetails.SourceDetails = &core.VolumeSourceFromVolumeBackupDetails{Id: &value}
	}

	ctx, cancel := context.WithTimeout(block.client.Context(), block.client.Timeout())
	defer cancel()
	prefix := strings.TrimSpace(os.Getenv(volumePrefixEnvVarName))
	if prefix != "" && !strings.HasSuffix(prefix, "-") {
		prefix = fmt.Sprintf("%s%s", prefix, "-")
	}

	newVolume, err := block.client.BlockStorage().CreateVolume(ctx, core.CreateVolumeRequest{
		CreateVolumeDetails: volumeDetails,
	})
	if err != nil {
		return nil, err
	}

	filesystemType := resolveFSType(options)

	region, ok := os.LookupEnv("OCI_SHORT_REGION")
	if !ok {
		metadata, err := block.metadata.Get()
		if err != nil {
			return nil, err
		}
		region = metadata.Region
	}

	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: *newVolume.Id,
			Annotations: map[string]string{
				ociVolumeID: *newVolume.Id,
			},
			Labels: map[string]string{
				plugin.LabelZoneRegion:        region,
				plugin.LabelZoneFailureDomain: *ad.Name,
			},
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeReclaimPolicy: options.PersistentVolumeReclaimPolicy,
			AccessModes:                   options.PVC.Spec.AccessModes,
			Capacity: v1.ResourceList{
				v1.ResourceName(v1.ResourceStorage): capacity,
			},
			PersistentVolumeSource: v1.PersistentVolumeSource{
				FlexVolume: &v1.FlexVolumeSource{
					Driver: plugin.OCIProvisionerName,
					FSType: filesystemType,
				},
			},
		},
	}

	return pv, nil
}

// Delete destroys a OCI volume created by Provision
func (block *blockProvisioner) Delete(volume *v1.PersistentVolume) error {
	volID, ok := volume.Annotations[ociVolumeID]
	if !ok {
		return errors.New("volumeid annotation not found on PV")
	}
	glog.Infof("Deleting volume %v with volumeId %v", volume, volID)

	request := core.DeleteVolumeRequest{VolumeId: common.String(volID)}
	ctx, cancel := context.WithTimeout(block.client.Context(), block.client.Timeout())
	defer cancel()

	response, err := block.client.BlockStorage().DeleteVolume(ctx, request)
	// If the volume does not exists (perhaps a user deleted it) then stop retrying the delete
	// Note that we cannot differentiate between a volume that no longer exists and an authentication failure.
	if response.RawResponse != nil && response.RawResponse.StatusCode == http.StatusNotFound {
		return nil
	}

	return err
}
