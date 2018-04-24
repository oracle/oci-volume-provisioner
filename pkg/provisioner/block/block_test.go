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
	"encoding/json"
	"fmt"
	"oci-volume-provisioner/pkg/oci/client"
	"oci-volume-provisioner/pkg/utils"
	"os"
	"testing"

	"github.com/kubernetes-incubator/external-storage/lib/controller"
	"github.com/oracle/oci-go-sdk/core"
	"github.com/oracle/oci-go-sdk/identity"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
)

// LoadConfig loads confuration data from a given path
func loadConfig(configFilePath string, t *testing.T) *client.ProvisionerClient {
	f, err := os.Open(configFilePath)
	if err != nil {
		t.Fatalf("Unable to load volume provisioner configuration file: %v", configFilePath)
	}
	defer f.Close()

	cfg, err := client.LoadConfig(f)
	if err != nil {
		t.Fatalf("Unable to load volume provisioner client: %v", err)
	}
	pc, err := client.FromConfig(cfg)
	if err != nil {
		t.Fatalf("Unable to load volume provisioner client details: %v", err)
	}
	return &pc
}

func TestResolveFSTypeWhenNotConfigured(t *testing.T) {
	options := controller.VolumeOptions{Parameters: make(map[string]string)}
	// test default fsType of 'ext4' is always returned.
	fst := resolveFSType(options)
	if fst != "ext4" {
		t.Fatalf("Unexpected filesystem type: '%s'.", fst)
	}
}

func TestResolveFSTypeWhenConfigured(t *testing.T) {
	// test default fsType of 'ext3' is always returned when configured.
	options := controller.VolumeOptions{Parameters: map[string]string{fsType: "ext3"}}
	fst := resolveFSType(options)
	if fst != "ext3" {
		t.Fatalf("Unexpected filesystem type: '%s'.", fst)
	}
}

func TestCreateVolumeFromBackup(t *testing.T) {
	// test creating a volume from an existing backup
	options := controller.VolumeOptions{PVName: "dummyVolumeOptions"}
	volumeBackupID := "dummyVolumeBackupId"
	pv := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				ociVolumeBackupID: "dummy",
			},
		},
	}
	options.PVC = pv
	adName := "dummyAdName"
	adCompartmentID := "dummyCompartmentId"
	availabilityDomain := identity.AvailabilityDomain{Name: &adName, CompartmentId: &adCompartmentID}
	configFilePath := "config/blockProvision.yaml"
	pc := loadConfig(configFilePath, t)
	block := blockProvisioner{client: *pc}
	volID := "dummyVolumeId"
	_createVolumeResp := &core.CreateVolumeResponse{Volume: core.Volume{Id: &volID}}
	byteResp, _ := json.Marshal(_createVolumeResp)
	resp := string(byteResp)
	m := map[string]*string{fmt.Sprintf("/%s/volumes", block.client.BlockStorage().BasePath): &resp}
	server := utils.OCIResponseStub(m)
	defer server.Close()
	block.client.BlockStorage().Host = server.URL
	provisionedVolume, err := block.Provision(options, &availabilityDomain)
	if err != nil {
		t.Fatalf("Failed to provision volume from block storage: %v", err)
	}
	if provisionedVolume.Annotations[ociVolumeID] != volID {
		t.Fatalf("Failed to assign the id of the blockID: %s, assigned %s instead", volumeBackupID,
			provisionedVolume.Annotations[ociVolumeID])
	}
}
