// Copyright 2017 The OCI Cloud Controller Manager Authors
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
	"io/ioutil"
	"os"

	baremetal "github.com/oracle/bmcs-go-sdk"
	"gopkg.in/yaml.v2"
)

// AuthConfig holds the configuration required for communicating with the OCI
// API.
type AuthConfig struct {
	TenancyOCID string `yaml:"tenancy"`
	UserOCID    string `yaml:"user"`
	PrivateKey  string `yaml:"key"`
	Fingerprint string `yaml:"fingerprint"`
	Region      string `yaml:"region"`
}

// Config holds the OCI cloud-provider config passed to Kubernetes compontents.
type Config struct {
	Auth AuthConfig `yaml:"auth"`
}

// Validate validates the OCI config.
func (c *Config) Validate() error {
	return ValidateConfig(c).ToAggregate()
}

// ReadConfig consumes the config and constructs a Config object.
func LoadClientConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	cfg := &Config{}
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(b, &cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// ClientFromConfig creates a baremetal client from the given configuration
func ClientFromConfig(cfg *Config) (client *baremetal.Client, err error) {
	ociClient, err := baremetal.NewClient(
		cfg.Auth.UserOCID,
		cfg.Auth.TenancyOCID,
		cfg.Auth.Fingerprint,
		baremetal.PrivateKeyBytes([]byte(cfg.Auth.PrivateKey)),
		baremetal.Region(cfg.Auth.Region))
	if err != nil {
		return nil, err
	}
	return ociClient, nil
}
