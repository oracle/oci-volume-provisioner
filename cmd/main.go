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
	"flag"
	"math/rand"
	"os"
	"syscall"
	"time"

	"github.com/golang/glog"
	"github.com/kubernetes-incubator/external-storage/lib/controller"
	"github.com/oracle/oci-volume-provisioner/pkg/provisioner"
	"github.com/oracle/oci-volume-provisioner/pkg/signals"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	resyncPeriod              = 15 * time.Second
	minResyncPeriod           = 12 * time.Hour
	provisionerName           = "oracle.com/oci"
	exponentialBackOffOnError = false
	failedRetryThreshold      = 5
	leasePeriod               = controller.DefaultLeaseDuration
	retryPeriod               = controller.DefaultRetryPeriod
	renewDeadline             = controller.DefaultRenewDeadline
	termLimit                 = controller.DefaultTermLimit
)

// informerResyncPeriod computes the time interval a shared informer waits
// before resyncing with the API server.
func informerResyncPeriod(minResyncPeriod time.Duration) func() time.Duration {
	return func() time.Duration {
		factor := rand.Float64() + 1
		return time.Duration(float64(minResyncPeriod.Nanoseconds()) * factor)
	}
}

func main() {
	syscall.Umask(0)

	flag.Parse()
	flag.Set("logtostderr", "true")

	// Set up signals so we handle the shutdown signal gracefully.
	stopCh := signals.SetupSignalHandler()

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

	// TODO (owainlewis) ensure this is clearly documented
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		glog.Fatal("env variable NODE_NAME must be set so that this provisioner can identify itself")
	}

	sharedInformerFactory := informers.NewSharedInformerFactory(clientset, informerResyncPeriod(minResyncPeriod)())

	// Create the provisioner: it implements the Provisioner interface expected by
	// the controller
	ociProvisioner := provisioner.NewOCIProvisioner(clientset, sharedInformerFactory.Core().V1().Nodes(), nodeName)

	// Start the provision controller which will dynamically provision oci
	// PVs
	pc := controller.NewProvisionController(
		clientset,
		resyncPeriod,
		provisionerName,
		ociProvisioner,
		serverVersion.GitVersion,
		exponentialBackOffOnError,
		failedRetryThreshold,
		leasePeriod,
		renewDeadline,
		retryPeriod,
		termLimit)

	go sharedInformerFactory.Start(stopCh)

	// We block waiting for Ready() after the shared informer factory has
	// started so we don't deadlock waiting for caches to sync.
	if err := ociProvisioner.Ready(stopCh); err != nil {
		glog.Fatalf("Failed to start volume provisioner: %v", err)
	}

	pc.Run(stopCh)
}
