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

package client

import (
	"strings"
	"testing"
)

func TestLoadClientConfigShouldFailWhenNoConfigProvided(t *testing.T) {
	_, err := LoadConfig(nil)
	if err == nil {
		t.Fatalf("should fail with when given no config")
	}
}

const validConfig = `
auth:
  region: us-phoenix-1
  tenancy: ocid1.tenancy.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
  compartment: ocid1.compartment.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
  user: ocid1.user.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
  key: |
    -----BEGIN RSA PRIVATE KEY-----
    -----END RSA PRIVATE KEY-----
  fingerprint: aa:bb:cc:dd:ee:ff:gg:hh:ii:jj:kk:ll:mm:nn:oo:pp
`
const validConfigNoRegion = `
auth:
  tenancy: ocid1.tenancy.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
  user: ocid1.user.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
  key: |
    -----BEGIN RSA PRIVATE KEY-----
    -----END RSA PRIVATE KEY-----
  fingerprint: aa:bb:cc:dd:ee:ff:gg:hh:ii:jj:kk:ll:mm:nn:oo:pp
`

func TestLoadClientConfigShouldSucceedWhenProvidedValidConfig(t *testing.T) {
	_, err := LoadConfig(strings.NewReader(validConfig))
	if err != nil {
		t.Fatalf("expected no error but got '%+v'", err)
	}
}

func TestLoadClientConfigShouldHaveNoDefaultRegionIfNoneSpecified(t *testing.T) {
	config, err := LoadConfig(strings.NewReader(validConfigNoRegion))
	if err != nil {
		t.Fatalf("expected no error but got '%+v'", err)
	}
	if config.Auth.Region != "" {
		t.Errorf("expected no region but got %s", config.Auth.Region)
	}
}

func TestLoadConfigShouldHaveCompartment(t *testing.T) {
	config, _ := LoadConfig(strings.NewReader(validConfig))
	expected := "ocid1.compartment.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	actual := config.Auth.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
	if actual != expected {
		t.Errorf("expected compartment %s but found %s", expected, actual)
	}
}
