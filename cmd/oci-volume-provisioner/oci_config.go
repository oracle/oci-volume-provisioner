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
	"os"

	baremetal "github.com/oracle/bmcs-go-sdk"
	gcfg "gopkg.in/gcfg.v1"
)

// Config holds the OCI cloud-provider config passed to Kubernetes compontents
// via the --cloud-config option.
type Config struct {
	Global struct {
		UserOCID       string `gcfg:"user"`
		TenancyOCID    string `gcfg:"tenancy"`
		Fingerprint    string `gcfg:"fingerprint"`
		PrivateKeyFile string `gcfg:"key-file"`
	}
}

func LoadClientConfig(path string) (cfg *Config, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	cfg = &Config{}
	err = gcfg.ReadInto(cfg, f)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func ClientFromConfig(cfg *Config) (client *baremetal.Client, err error) {
	privateKeyFile := baremetal.PrivateKeyFilePath(cfg.Global.PrivateKeyFile)
	ociClient, err := baremetal.NewClient(
		cfg.Global.UserOCID,
		cfg.Global.TenancyOCID,
		cfg.Global.Fingerprint,
		privateKeyFile)
	if err != nil {
		return nil, err
	}
	return ociClient, nil
}
