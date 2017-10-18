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

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/golang/glog"
	"github.com/kubernetes-incubator/external-storage/lib/controller"
	baremetal "github.com/oracle/bmcs-go-sdk"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
)

const (
	resyncPeriod              = 15 * time.Second
	provisionerName           = "oracle.com/oci"
	exponentialBackOffOnError = false
	failedRetryThreshold      = 5
	leasePeriod               = controller.DefaultLeaseDuration
	retryPeriod               = controller.DefaultRetryPeriod
	renewDeadline             = controller.DefaultRenewDeadline
	termLimit                 = controller.DefaultTermLimit
	ociProvisionerIdentity    = "ociProvisionerIdentity"
	ociVolumeID               = "ociVolumeID"
	ociAvailabilityDomain     = "ociAvailabilityDomain"
	ociCompartment            = "ociCompartment"
)

type ociProvisioner struct {
	client baremetal.Client

	// Identity of this ociProvisioner, set to node's name. Used to identify "this" provisioner's PVs.
	identity string

	tenancyID string
}

// NewOCIProvisioner creates a new oci provisioner.
func NewOCIProvisioner() controller.Provisioner {
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		glog.Fatal("env variable NODE_NAME must be set so that this provisioner can identify itself")
	}

	cfg, err := LoadClientConfig("/etc/oci/config.cfg")
	if err != nil {
		glog.Fatalf("Unable to load volume provisioner client: %v", err)
	}

	client, err := ClientFromConfig(cfg)
	if err != nil {
		glog.Fatalf("Unable to load volume provisioner client: %v", err)
	}
	return &ociProvisioner{
		client:    *client,
		identity:  nodeName,
		tenancyID: cfg.Global.TenancyOCID,
	}
}

var _ controller.Provisioner = &ociProvisioner{}

func RoundUpSize(volumeSizeBytes int64, allocationUnitBytes int64) int64 {
	return (volumeSizeBytes + allocationUnitBytes - 1) / allocationUnitBytes
}

func (p *ociProvisioner) findCompartmentIdByName(name string) (*baremetal.Compartment, error) {
	compartments, err := p.client.ListCompartments(&baremetal.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, compartment := range compartments.Compartments {
		if compartment.Name == name {
			return &compartment, nil
		}
	}
	return nil, fmt.Errorf("unable to find OCI compartment named '%s'", name)
}

func (p *ociProvisioner) findADByName(name string) (*baremetal.AvailabilityDomain, error) {
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

// Provision creates a storage asset and returns a PV object representing it.
func (p *ociProvisioner) Provision(options controller.VolumeOptions) (*v1.PersistentVolume, error) {
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

	compartmentName, ok := options.PVC.Spec.Selector.MatchLabels["oci-compartment"]
	if !ok {
		return nil, fmt.Errorf("claim Selector must specify 'oci-compartment'")
	}

	compartment, err := p.findCompartmentIdByName(compartmentName)
	if err != nil {
		return nil, err
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

	volSizeMB := int(RoundUpSize(volSizeBytes, 1024*1024))
	glog.Infof("Creating volume size=%v AD=%s compartment=%q compartmentID=%q", volSizeMB,
		availabilityDomain.Name,
		compartmentName,
		compartment.ID)

	newVolume, err := p.client.CreateVolume(availabilityDomain.Name,
		compartment.ID,
		&baremetal.CreateVolumeOptions{SizeInMBs: volSizeMB})
	if err != nil {
		return nil, err
	}

	volumeName := mapVolumeIDToName(newVolume.ID)

	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: volumeName,
			Annotations: map[string]string{
				ociProvisionerIdentity: p.identity,
				ociVolumeID:            newVolume.ID,
				ociAvailabilityDomain:  availabilityDomain.Name,
				ociCompartment:         compartment.CompartmentID,
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
					FSType: "ext4",
				},
			},
		},
	}
	return pv, nil
}

// Delete removes the storage asset that was created by Provision represented
// by the given PV.
func (p *ociProvisioner) Delete(volume *v1.PersistentVolume) error {
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

func main() {
	syscall.Umask(0)

	flag.Parse()
	flag.Set("logtostderr", "true")

	// Create an InClusterConfig and use it to create a client for the controller
	// to use to communicate with Kubernetes
	config, err := rest.InClusterConfig()
	if err != nil {
		glog.Fatalf("Failed to create config: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatalf("Failed to create client: %v", err)
	}

	// The controller needs to know what the server version is because out-of-tree
	// provisioners aren't officially supported until 1.5
	serverVersion, err := clientset.Discovery().ServerVersion()
	if err != nil {
		glog.Fatalf("Error getting server version: %v", err)
	}

	// Create the provisioner: it implements the Provisioner interface expected by
	// the controller
	ociProvisioner := NewOCIProvisioner()

	// Start the provision controller which will dynamically provision oci
	// PVs
	pc := controller.NewProvisionController(
		clientset, resyncPeriod, provisionerName, ociProvisioner,
		serverVersion.GitVersion, exponentialBackOffOnError,
		failedRetryThreshold, leasePeriod, renewDeadline,
		retryPeriod, termLimit)
	pc.Run(wait.NeverStop)
}
