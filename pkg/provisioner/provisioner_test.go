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

package provisioner

import (
	"testing"

	"github.com/kubernetes-incubator/external-storage/lib/controller"
)

// func TestGetOptionsForVolume(t *testing.T) {
// 	volume := getOptionsForVolume(100, "", "myvolume")
// 	if volume.DisplayName != "myvolume" {
// 		t.Fatalf("Incorrect display name. Expecting %s but got %s", "myvolume", volume.DisplayName)
// 	}
// }

// func TestGetGetOptionsForVolumeCustomDisplayName(t *testing.T) {
// 	volume := getOptionsForVolume(100, "XXX", "myvolume")

// 	if volume.DisplayName != "XXXmyvolume" {
// 		t.Fatalf("Incorrect display name. Expecting %s but got %s", "XXXmyvolume", volume.DisplayName)
// 	}
// }

func TestNewCreateVolumeDetails(t *testing.T) {
	volume := newCreateVolumeDetails("test-ad", "test-ocid", "", "myvolume", 100)
	if *volume.DisplayName != "myvolume" {
		t.Fatalf("Incorrect display name. Expecting %s but got %s", "myvolume", *volume.DisplayName)
	}
}

func TestNewCreateVolumeDetailsForVolumeCustomDisplayName(t *testing.T) {
	volume := newCreateVolumeDetails("test-ad", "test-ocid", "XXX", "myvolume", 100)

	if *volume.DisplayName != "XXXmyvolume" {
		t.Fatalf("Incorrect display name. Expecting %s but got %s", "XXXmyvolume", *volume.DisplayName)
	}
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
