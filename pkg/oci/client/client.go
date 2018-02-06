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
	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/core"
	"github.com/oracle/oci-go-sdk/identity"
	"github.com/pkg/errors"
)

type ProvisionerClient struct {
	BlockStorage *core.BlockstorageClient
	Identity     *identity.IdentityClient
}

// FromConfig creates an oci client from the given configuration.
func FromConfig(cfg *Config) (*ProvisionerClient, error) {
	config, err := newConfigurationProvider(cfg)
	if err != nil {
		return nil, err
	}
	blockStorage, err := core.NewBlockstorageClientWithConfigurationProvider(config)
	if err != nil {
		return nil, err
	}
	identity, err := identity.NewIdentityClientWithConfigurationProvider(config)
	if err != nil {
		return nil, err
	}
	client := ProvisionerClient{&blockStorage, &identity}
	return &client, nil
}

func newConfigurationProvider(cfg *Config) (common.ConfigurationProvider, error) {
	var conf common.ConfigurationProvider
	if conf == nil {
		conf = common.DefaultConfigProvider()
	} else {
		err := cfg.Validate()
		if err != nil {
			return nil, errors.Wrap(err, "invalid client config")
		}
		conf = common.NewRawConfigurationProvider(
			cfg.Auth.TenancyOCID,
			cfg.Auth.UserOCID,
			cfg.Auth.Region,
			cfg.Auth.Fingerprint,
			cfg.Auth.PrivateKey,
			common.String(cfg.Auth.PrivateKeyPassphrase))
	}
	return conf, nil
}
