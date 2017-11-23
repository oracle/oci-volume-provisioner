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

import baremetal "github.com/oracle/bmcs-go-sdk"

// FromConfig creates a baremetal client from the given configuration.
func FromConfig(cfg *Config) (*baremetal.Client, error) {
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
