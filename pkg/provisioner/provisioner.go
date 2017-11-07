// Copyright 2017 The OCI Volume Provisioner Authors
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
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/golang/glog"
	"github.com/kubernetes-incubator/external-storage/lib/controller"
	baremetal "github.com/oracle/bmcs-go-sdk"
	"github.com/oracle/oci-volume-provisioner/pkg/oci/client"
	"github.com/oracle/oci-volume-provisioner/pkg/oci/instancemeta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
)

const (
	ociProvisionerIdentity = "ociProvisionerIdentity"
	ociVolumeID            = "ociVolumeID"
	ociAvailabilityDomain  = "ociAvailabilityDomain"
	ociCompartment         = "ociCompartment"
	configFilePath         = "/etc/oci/config.yaml"
	fsType                 = "fsType"
)

// OCIProvisioner is a dynamic volume provisioner that satisfies
// the Kubernetes external storage Provisioner controller interface
type OCIProvisioner struct {
	client baremetal.Client

	// Identity of this ociProvisioner, set to node's name. Used to identify "this" provisioner's PVs.
	identity      string
	tenancyID     string
	compartmentID string

	metadata *instancemeta.InstanceMetadata
}

// NewOCIProvisioner creates a new OCI provisioner.
func NewOCIProvisioner(nodeName string) controller.Provisioner {

	f, err := os.Open(configFilePath)
	if err != nil {
		glog.Fatalf("Unable to load volume provisioner configuration file: %v", configFilePath)
	}
	defer f.Close()

	cfg, err := client.LoadConfig(f)
	if err != nil {
		glog.Fatalf("Unable to load volume provisioner client: %v", err)
	}

	client, err := client.FromConfig(cfg)
	if err != nil {
		glog.Fatalf("Unable to create volume provisioner client: %v", err)
	}

	metadata, err := instancemeta.New().Get()
	if err != nil {
		glog.Fatalf("Unable to retrieve instance metadata: %v", err)
	}

	return &OCIProvisioner{
		client:        *client,
		identity:      nodeName,
		tenancyID:     cfg.Auth.TenancyOCID,
		compartmentID: cfg.Auth.CompartmentOCID,
		metadata:      metadata,
	}
}

var _ controller.Provisioner = &OCIProvisioner{}

func roundUpSize(volumeSizeBytes int64, allocationUnitBytes int64) int64 {
	return (volumeSizeBytes + allocationUnitBytes - 1) / allocationUnitBytes
}

func (p *OCIProvisioner) findADByName(name string) (*baremetal.AvailabilityDomain, error) {
	ads, err := p.client.ListAvailabilityDomains(p.tenancyID)
	if err != nil {
		return nil, err
	}
	for _, ad := range ads.AvailabilityDomains {
		if strings.HasSuffix(ad.Name, name) {
			return &ad, nil
		}
	}
	return nil, fmt.Errorf("error looking up availability domain '%s'", name)
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

// Provision creates a storage asset and returns a PV object representing it.
func (p *OCIProvisioner) Provision(options controller.VolumeOptions) (*v1.PersistentVolume, error) {
	for _, accessMode := range options.PVC.Spec.AccessModes {
		if accessMode != v1.ReadWriteOnce {
			return nil, fmt.Errorf("invalid access mode %v specified. Only %v is supported",
				accessMode,
				v1.ReadWriteOnce)
		}
	}

	if options.PVC.Spec.Selector == nil {
		return nil, fmt.Errorf("claim Selector must be specified")
	}
	glog.Infof("VolumeOptions.PVC.Spec.Selector %#v", *options.PVC.Spec.Selector)

	var compartmentOCID string
	if p.compartmentID == "" {
		glog.Infof("'CompartmentID' not given. Using compartment OCID %s from instance metadata", p.metadata.CompartmentOCID)
		compartmentOCID = p.metadata.CompartmentOCID
	} else {
		compartmentOCID = p.compartmentID
	}

	availabilityDomainName, ok := options.PVC.Spec.Selector.MatchLabels["oci-availability-domain"]
	if !ok {
		return nil, fmt.Errorf("claim Selector must specify 'oci-availability-domain'")
	}

	availabilityDomain, err := p.findADByName(availabilityDomainName)
	if err != nil {
		return nil, err
	}

	// Calculate the size
	volSize := options.PVC.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)]
	volSizeBytes := volSize.Value()
	glog.Infof("Volume size %#v", volSizeBytes)

	volSizeMB := int(roundUpSize(volSizeBytes, 1024*1024))

	glog.Infof("Creating volume size=%v AD=%s compartmentOCID=%q", volSizeMB, availabilityDomain.Name, compartmentOCID)

	newVolume, err := p.client.CreateVolume(availabilityDomain.Name, compartmentOCID, &baremetal.CreateVolumeOptions{SizeInMBs: volSizeMB})
	if err != nil {
		return nil, err
	}

	volumeName := mapVolumeIDToName(newVolume.ID)
	filesystemType := resolveFSType(options)

	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: volumeName,
			Annotations: map[string]string{
				ociProvisionerIdentity: p.identity,
				ociVolumeID:            newVolume.ID,
				ociAvailabilityDomain:  availabilityDomain.Name,
				ociCompartment:         compartmentOCID,
			},
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

// Delete removes the storage asset that was created by Provision represented
// by the given PV.
func (p *OCIProvisioner) Delete(volume *v1.PersistentVolume) error {
	glog.Infof("Deleting volume %v with volumeId %v", volume, volume.Annotations[ociVolumeID])

	ann, ok := volume.Annotations[ociProvisionerIdentity]
	if !ok {
		return errors.New("identity annotation not found on PV")
	}
	if ann != p.identity {
		return &controller.IgnoredError{Reason: "identity annotation on PV does not match ours"}
	}

	ann, ok = volume.Annotations[ociVolumeID]
	if !ok {
		return errors.New("volumeid annotation not found on PV")
	}

	return p.client.DeleteVolume(volume.Annotations[ociVolumeID], nil)
}
