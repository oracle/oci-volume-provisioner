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
	"os"
	"time"

	"github.com/kubernetes-incubator/external-storage/lib/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"syscall"

	"fmt"
	"github.com/golang/glog"
	baremetal "github.com/oracle/bmcs-go-sdk"
	"strings"
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
)

type ociProvisioner struct {
	client baremetal.Client

	// Identity of this ociProvisioner, set to node's name. Used to identify "this" provisioner's PVs.
	identity string

	tenancyId string
}

// NewOciProvisioner creates a new oci provisioner
func NewOciProvisioner() controller.Provisioner {
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		glog.Fatal("env variable NODE_NAME must be set so that this provisioner can identify itself")
	}

	cfg, err := LoadClientConfig("/etc/oci/config.cfg")
	if err != nil {
		glog.Fatalf("Unable to load volume provisioner client:%v", err)
	}

	client, err := ClientFromConfig(cfg)
	if err != nil {
		glog.Fatalf("Unable to load volume provisioner client:%v", err)
	}
	return &ociProvisioner{
		client:    *client,
		identity:  nodeName,
		tenancyId: cfg.Global.TenancyOCID,
	}
}

var _ controller.Provisioner = &ociProvisioner{}

func RoundUpSize(volumeSizeBytes int64, allocationUnitBytes int64) int64 {
	return (volumeSizeBytes + allocationUnitBytes - 1) / allocationUnitBytes
}

func (p *ociProvisioner) findCompartmentIdByName(name string) (*baremetal.Compartment, error) {
	compartments, err := p.client.ListCompartments(nil)
	if err != nil {
		return nil, err
	}
	for _, compartment := range compartments.Compartments {
		if compartment.Name == name {
			return &compartment, nil
		}
	}
	return nil, fmt.Errorf("Unable to find OCI comparment named '%s'", name)
}

func (p *ociProvisioner) findADByName(name string) (*baremetal.AvailabilityDomain, error) {
	ADs, err := p.client.ListAvailabilityDomains(p.tenancyId)
	if err != nil {
		return nil, err
	}
	for _, AD := range ADs.AvailabilityDomains {
		if strings.HasSuffix(AD.Name, name) {
			return &AD, nil
		}
	}
	return nil, fmt.Errorf("Error looking up availability domain '%s'", name)
}

// Provision creates a storage asset and returns a PV object representing it.
func (p *ociProvisioner) Provision(options controller.VolumeOptions) (*v1.PersistentVolume, error) {
	glog.Infof("Provision:OCI: VolumeOptions %#v", options)

	for _, accessMode := range options.PVC.Spec.AccessModes {
		if accessMode != v1.ReadWriteOnce {
			return nil, fmt.Errorf("Invalid access mode %v specified. Only %v is supported",
				accessMode,
				v1.ReadWriteOnce)
		}
	}

	if options.PVC.Spec.Selector == nil {
		return nil, fmt.Errorf("OCI: claim Selector must be specified")
	}
	glog.Infof("Provision:OCI: VolumeOptions.PVC.Spec.Selector %#v", *options.PVC.Spec.Selector)

	compartmentName, ok := options.PVC.Spec.Selector.MatchLabels["oci-compartment"]
	if !ok {
		return nil, fmt.Errorf("OCI: claim Selector must specify 'oci-compartment'")
	}

	compartment, err := p.findCompartmentIdByName(compartmentName)
	if err != nil {
		return nil, err
	}

	availabilityDomainName, ok := options.PVC.Spec.Selector.MatchLabels["oci-availability-domain"]
	if !ok {
		return nil, fmt.Errorf("OCI: claim Selector must specify 'oci-availability-domain'")
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
	glog.Infof("Creating volume of size %v in AD %s in compartment %s(%#v)", volSizeMB,
		availabilityDomain.Name,
		compartmentName,
		compartment.ID)

	newVolume, err := p.client.CreateVolume(availabilityDomain.Name,
		compartment.ID,
		&baremetal.CreateVolumeOptions{
			SizeInMBs: volSizeMB,
		})
	if err != nil {
		return nil, err
	}

	volumeName := strings.Split(newVolume.ID, ".")[4]

	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: volumeName,
			Annotations: map[string]string{
				"ociProvisionerIdentity": p.identity,
				"ociVolumeId":            newVolume.ID,
				"ociAvailabilityDomain":  availabilityDomain.Name,
				"ociCompartment":         compartment.CompartmentID,
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
	glog.Infof("PV:%#v", pv)
	return pv, nil
}

// Delete removes the storage asset that was created by Provision represented
// by the given PV.
func (p *ociProvisioner) Delete(volume *v1.PersistentVolume) error {
	glog.Infof("Delete:OCI: volume %#v ", volume)
	glog.Infof("Delete:OCI: VolumeId %#v ", volume.Annotations["ociVolumeId"])

	ann, ok := volume.Annotations["ociProvisionerIdentity"]
	if !ok {
		return errors.New("identity annotation not found on PV")
	}
	if ann != p.identity {
		return &controller.IgnoredError{Reason: "identity annotation on PV does not match ours"}
	}

	ann, ok = volume.Annotations["ociVolumeId"]
	if !ok {
		return errors.New("volumeid annotation not found on PV")
	}
	glog.Infof("Deleting Volume: %#v", volume)

	return p.client.DeleteVolume(volume.Annotations["ociVolumeId"], nil)
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
	ociProvisioner := NewOciProvisioner()

	// Start the provision controller which will dynamically provision oci
	// PVs
	pc := controller.NewProvisionController(clientset, resyncPeriod, provisionerName, ociProvisioner, serverVersion.GitVersion, exponentialBackOffOnError, failedRetryThreshold, leasePeriod, renewDeadline, retryPeriod, termLimit)
	pc.Run(wait.NeverStop)
}
