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
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/core"
	"github.com/oracle/oci-go-sdk/identity"
	"github.com/oracle/oci-volume-provisioner/pkg/oci/instancemeta"
)

// ProvisionerClient wraps the OCI sub-clients required for volume provisioning.
type provisionerClient struct {
	cfg          *Config
	blockStorage *core.BlockstorageClient
	identity     *identity.IdentityClient
	context      context.Context
	timeout      time.Duration
	metadata     *instancemeta.InstanceMetadata
}

// ProvisionerClient is passed to all sub clients to provision a volume
type ProvisionerClient interface {
	BlockStorage() *core.BlockstorageClient
	Identity() *identity.IdentityClient
	Context() context.Context
	Timeout() time.Duration
	CompartmentOCID() string
	TenancyOCID() string
}

func (p *provisionerClient) BlockStorage() *core.BlockstorageClient {
	return p.blockStorage
}

func (p *provisionerClient) Identity() *identity.IdentityClient {
	return p.identity
}

func (p *provisionerClient) Context() context.Context {
	return p.context
}

func (p *provisionerClient) Timeout() time.Duration {
	return p.timeout
}

func (p *provisionerClient) CompartmentOCID() (compartmentOCID string) {
	if p.cfg.Auth.CompartmentOCID == "" {
		if p.metadata == nil {
			log.Fatalf("Unable to get compartment OCID. Please provide this via config")
			return
		}
		glog.Infof("'CompartmentID' not given. Using compartment OCID %s from instance metadata", p.metadata.CompartmentOCID)
		compartmentOCID = p.metadata.CompartmentOCID
	} else {
		compartmentOCID = p.cfg.Auth.CompartmentOCID
	}
	return
}

func (p *provisionerClient) TenancyOCID() string {
	return p.cfg.Auth.TenancyOCID
}

// FromConfig creates an OCI client from the given configuration.
func FromConfig(cfg *Config) (ProvisionerClient, error) {
	config, err := newConfigurationProvider(cfg)
	if err != nil {
		return nil, err
	}

	blockStorage, err := core.NewBlockstorageClientWithConfigurationProvider(config)
	if err != nil {
		return nil, err
	}
	err = configureCustomTransport(&blockStorage.BaseClient)
	if err != nil {
		return nil, err
	}

	identity, err := identity.NewIdentityClientWithConfigurationProvider(config)
	if err != nil {
		return nil, err
	}
	err = configureCustomTransport(&identity.BaseClient)
	if err != nil {
		return nil, err
	}

	metadata, err := instancemeta.New().Get()
	if err != nil {
		glog.Warning("Unable to retrieve instance metadata: %v", err)
	}

	return &provisionerClient{
		cfg:          cfg,
		blockStorage: &blockStorage,
		identity:     &identity,
		timeout:      3 * time.Minute,
		context:      context.Background(),
		metadata:     metadata,
	}, nil
}

func newConfigurationProvider(cfg *Config) (common.ConfigurationProvider, error) {
	var conf common.ConfigurationProvider
	if cfg != nil {
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
	} else {
		conf = common.DefaultConfigProvider()
	}
	return conf, nil
}

func configureCustomTransport(baseClient *common.BaseClient) error {

	httpClient := baseClient.HTTPClient.(*http.Client)

	var transport *http.Transport
	if httpClient.Transport == nil {
		transport = &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
	} else {
		transport = httpClient.Transport.(*http.Transport)
	}

	ociProxy := os.Getenv("OCI_PROXY")
	if ociProxy != "" {
		proxyURL, err := url.Parse(ociProxy)
		if err != nil {
			return fmt.Errorf("failed to parse OCI proxy url: %s, err: %v", ociProxy, err)
		}
		transport.Proxy = func(req *http.Request) (*url.URL, error) {
			return proxyURL, nil
		}
	}

	trustedCACertPath := os.Getenv("TRUSTED_CA_CERT_PATH")
	if trustedCACertPath != "" {
		glog.Infof("configuring OCI client with a new trusted ca: %s", trustedCACertPath)
		trustedCACert, err := ioutil.ReadFile(trustedCACertPath)
		if err != nil {
			return fmt.Errorf("failed to read root certificate: %s, err: %v", trustedCACertPath, err)
		}
		caCertPool := x509.NewCertPool()
		ok := caCertPool.AppendCertsFromPEM(trustedCACert)
		if !ok {
			return fmt.Errorf("failed to parse root certificate: %s", trustedCACertPath)
		}
		transport.TLSClientConfig = &tls.Config{RootCAs: caCertPool}
	}
	httpClient.Transport = transport
	return nil
}
