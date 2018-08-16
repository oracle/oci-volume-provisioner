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
	"errors"
	"io"
	"io/ioutil"

	"gopkg.in/yaml.v2"

	"go.uber.org/zap"
)

// AuthConfig holds the configuration required for communicating with the OCI
// API.
type AuthConfig struct {
	TenancyOCID string `yaml:"tenancy"`
	UserOCID    string `yaml:"user"`
	// CompartmentOCID is DEPRECATED and should be set on the top level Config
	// struct.
	CompartmentOCID      string `yaml:"compartment"`
	PrivateKey           string `yaml:"key"`
	Fingerprint          string `yaml:"fingerprint"`
	Region               string `yaml:"region"`
	PrivateKeyPassphrase string `yaml:"passphrase"`
}

// Config holds the OCI cloud-provider config passed to Kubernetes compontents.
type Config struct {
	Auth                  AuthConfig `yaml:"auth"`
	UseInstancePrincipals bool       `yaml:"useInstancePrincipals"`
	// CompartmentOCID is the OCID of the Compartment within which the cluster
	// resides.
	CompartmentOCID string `yaml:"compartment"`
}

// Validate validates the OCI config.
func (c *Config) Validate() error {
	return ValidateConfig(c).ToAggregate()
}

// LoadConfig consumes the config Reader and constructs a Config object.
func LoadConfig(r io.Reader) (*Config, error) {
	if r == nil {
		return nil, errors.New("no configuration file given")
	}

	cfg := &Config{}

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(b, &cfg)
	if err != nil {
		return nil, err
	}

	cfg.Complete()

	return cfg, nil
}

// Complete the config applying defaults / overrides.
func (c *Config) Complete() {
	if c.CompartmentOCID == "" && c.Auth.CompartmentOCID != "" {
		zap.S().Warn("cloud-provider config: \"auth.compartment\" is DEPRECATED and will be removed in a later release. Please set \"compartment\".")
		c.CompartmentOCID = c.Auth.CompartmentOCID
	}
}
