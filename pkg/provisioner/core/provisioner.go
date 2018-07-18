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

package core

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/golang/glog"

	"github.com/kubernetes-incubator/external-storage/lib/controller"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	informersv1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	listersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/oracle/oci-volume-provisioner/pkg/oci/client"
	"github.com/oracle/oci-volume-provisioner/pkg/oci/instancemeta"
	"github.com/oracle/oci-volume-provisioner/pkg/provisioner/block"
	"github.com/oracle/oci-volume-provisioner/pkg/provisioner/filestorage"
	"github.com/oracle/oci-volume-provisioner/pkg/provisioner/plugin"
)

const (
	ociProvisionerIdentity = "ociProvisionerIdentity"
	ociAvailabilityDomain  = "ociAvailabilityDomain"
	ociCompartment         = "ociCompartment"
	configFilePath         = "/etc/oci/config.yaml"
)

// OCIProvisioner is a dynamic volume provisioner that satisfies
// the Kubernetes external storage Provisioner controller interface
type OCIProvisioner struct {
	client client.ProvisionerClient

	kubeClient       kubernetes.Interface
	nodeLister       listersv1.NodeLister
	nodeListerSynced cache.InformerSynced

	storageClassProvisioners map[string]plugin.ProvisionerPlugin
}

// NewOCIProvisioner creates a new OCI provisioner.
func NewOCIProvisioner(kubeClient kubernetes.Interface, nodeInformer informersv1.NodeInformer, nodeName string, volumeRoundingEnabled bool, minVolumeSize resource.Quantity) *OCIProvisioner {
	configPath, ok := os.LookupEnv("CONFIG_YAML_FILENAME")
	if !ok {
		configPath = configFilePath
	}

	f, err := os.Open(configPath)
	if err != nil {
		glog.Fatalf("Unable to load volume provisioner configuration file: %v", configPath)
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

	blockProvisioner := block.NewBlockProvisioner(client, instancemeta.New(), volumeRoundingEnabled, minVolumeSize)
	return &OCIProvisioner{
		client:           client,
		kubeClient:       kubeClient,
		nodeLister:       nodeInformer.Lister(),
		nodeListerSynced: nodeInformer.Informer().HasSynced,
		storageClassProvisioners: map[string]plugin.ProvisionerPlugin{
			"oci":      blockProvisioner,
			"oci-ext3": blockProvisioner,
			"oci-fss":  filestorage.NewFilesystemProvisioner(client),
		},
	}
}

var _ controller.Provisioner = &OCIProvisioner{}

func roundUpSize(volumeSizeBytes int64, allocationUnitBytes int64) int64 {
	return (volumeSizeBytes + allocationUnitBytes - 1) / allocationUnitBytes
}

// mapAvailabilityDomainToFailureDomain maps a given Availability Domain to a
// k8s label compat. string.
func mapAvailabilityDomainToFailureDomain(AD string) string {
	parts := strings.SplitN(AD, ":", 2)
	if parts == nil {
		return ""
	}
	return parts[len(parts)-1]
}

// Provision creates a storage asset and returns a PV object representing it.
func (p *OCIProvisioner) Provision(options controller.VolumeOptions) (*v1.PersistentVolume, error) {
	availabilityDomainName, availabilityDomain, err := p.chooseAvailabilityDomain(options.PVC)
	if err != nil {
		return nil, err
	}

	provisioner, ok := p.storageClassProvisioners[*options.PVC.Spec.StorageClassName]
	if !ok {
		return nil, fmt.Errorf("Storage class '%s' not supported", *options.PVC.Spec.StorageClassName)
	}

	persistentVolume, err := provisioner.Provision(options, availabilityDomain)
	if err == nil {
		persistentVolume.ObjectMeta.Annotations[ociProvisionerIdentity] = ociProvisionerIdentity
		persistentVolume.ObjectMeta.Annotations[ociAvailabilityDomain] = availabilityDomainName
		persistentVolume.ObjectMeta.Annotations[ociCompartment] = p.client.CompartmentOCID()
		persistentVolume.ObjectMeta.Labels[metav1.LabelZoneFailureDomain] = mapAvailabilityDomainToFailureDomain(*availabilityDomain.Name)
	}
	return persistentVolume, err
}

// Delete removes the storage asset that was created by Provision represented
// by the given PV.
func (p *OCIProvisioner) Delete(volume *v1.PersistentVolume) error {
	identity, ok := volume.Annotations[ociProvisionerIdentity]
	if !ok {
		return errors.New("identity annotation not found on PV")
	}
	if identity != ociProvisionerIdentity {
		return &controller.IgnoredError{Reason: "identity annotation on PV does not match ours"}
	}

	provisioner, ok := p.storageClassProvisioners[volume.Spec.StorageClassName]
	if !ok {
		return fmt.Errorf("Storage class '%s' not supported", volume.Spec.StorageClassName)
	}

	return provisioner.Delete(volume)
}

// Ready waits unitl the the nodeLister has been synced.
func (p *OCIProvisioner) Ready(stopCh <-chan struct{}) error {
	if !cache.WaitForCacheSync(stopCh, p.nodeListerSynced) {
		return errors.New("unable to sync caches for OCI Volume Provisioner")
	}
	return nil
}
