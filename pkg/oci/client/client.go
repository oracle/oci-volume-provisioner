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
	"github.com/oracle/oci-go-sdk/common/auth"
	"github.com/oracle/oci-go-sdk/core"
	"github.com/oracle/oci-go-sdk/filestorage"
	"github.com/oracle/oci-go-sdk/identity"
	"github.com/oracle/oci-volume-provisioner/pkg/oci/instancemeta"
)

// ProvisionerClient wraps the OCI sub-clients required for volume provisioning.
type provisionerClient struct {
	cfg            *Config
	blockStorage   *core.BlockstorageClient
	identity       *identity.IdentityClient
	fileStorage    *filestorage.FileStorageClient
	virtualNetwork *core.VirtualNetworkClient
	context        context.Context
	timeout        time.Duration
	metadata       *instancemeta.InstanceMetadata
}

// BlockStorage specifies the subset of the OCI core API utilised by the provisioner.
type BlockStorage interface {
	CreateVolume(ctx context.Context, request core.CreateVolumeRequest) (response core.CreateVolumeResponse, err error)
	DeleteVolume(ctx context.Context, request core.DeleteVolumeRequest) (response core.DeleteVolumeResponse, err error)
	GetVolume(ctx context.Context, request core.GetVolumeRequest) (response core.GetVolumeResponse, err error)
}

// Identity specifies the subset of the OCI identity API utilised by the provisioner.
type Identity interface {
	ListAvailabilityDomains(ctx context.Context, request identity.ListAvailabilityDomainsRequest) (response identity.ListAvailabilityDomainsResponse, err error)
}

// FileStorage specifies the subset of the OCI core API utilised by the provisioner.
type FileStorage interface {
	CreateFileSystem(ctx context.Context, request filestorage.CreateFileSystemRequest) (response filestorage.CreateFileSystemResponse, err error)
	DeleteFileSystem(ctx context.Context, request filestorage.DeleteFileSystemRequest) (response filestorage.DeleteFileSystemResponse, err error)
	CreateMountTarget(ctx context.Context, request filestorage.CreateMountTargetRequest) (response filestorage.CreateMountTargetResponse, err error)
	CreateExport(ctx context.Context, request filestorage.CreateExportRequest) (response filestorage.CreateExportResponse, err error)
	DeleteExport(ctx context.Context, request filestorage.DeleteExportRequest) (response filestorage.DeleteExportResponse, err error)
	GetMountTarget(ctx context.Context, request filestorage.GetMountTargetRequest) (response filestorage.GetMountTargetResponse, err error)
	ListMountTargets(ctx context.Context, request filestorage.ListMountTargetsRequest) (response filestorage.ListMountTargetsResponse, err error)
}

//VirtualNetwork specifies the subset of the OCI core API utilised by the provisioner.
type VirtualNetwork interface {
	GetPrivateIp(ctx context.Context, request core.GetPrivateIpRequest) (response core.GetPrivateIpResponse, err error)
}

// ProvisionerClient is passed to all sub clients to provision a volume
type ProvisionerClient interface {
	BlockStorage() BlockStorage
	Identity() Identity
	FileStorage() FileStorage
	VirtualNetwork() VirtualNetwork
	Context() context.Context
	Timeout() time.Duration
	CompartmentOCID() string
	TenancyOCID() string
}

func (p *provisionerClient) BlockStorage() BlockStorage {
	return p.blockStorage
}

func (p *provisionerClient) Identity() Identity {
	return p.identity
}

func (p *provisionerClient) FileStorage() FileStorage {
	return p.fileStorage
}

func (p *provisionerClient) VirtualNetwork() VirtualNetwork {
	return p.virtualNetwork
}

func (p *provisionerClient) Context() context.Context {
	return p.context
}

func (p *provisionerClient) Timeout() time.Duration {
	return p.timeout
}

func (p *provisionerClient) CompartmentOCID() (compartmentOCID string) {
	if p.cfg.CompartmentOCID == "" {
		if p.metadata == nil {
			log.Fatalf("Unable to get compartment OCID. Please provide this via config")
			return
		}
		glog.Infof("'CompartmentID' not given. Using compartment OCID %s from instance metadata", p.metadata.CompartmentOCID)
		compartmentOCID = p.metadata.CompartmentOCID
	} else {
		compartmentOCID = p.cfg.CompartmentOCID
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

	fileStorage, err := filestorage.NewFileStorageClientWithConfigurationProvider(config)
	if err != nil {
		return nil, err
	}

	virtualNetwork, err := core.NewVirtualNetworkClientWithConfigurationProvider(config)
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
		glog.Warningf("Unable to retrieve instance metadata: %s", err)
	}

	return &provisionerClient{
		cfg:            cfg,
		blockStorage:   &blockStorage,
		identity:       &identity,
		fileStorage:    &fileStorage,
		virtualNetwork: &virtualNetwork,
		timeout:        3 * time.Minute,
		context:        context.Background(),
		metadata:       metadata,
	}, nil
}

func newConfigurationProvider(cfg *Config) (common.ConfigurationProvider, error) {
	var conf common.ConfigurationProvider
	if cfg != nil {
		err := cfg.Validate()
		if err != nil {
			return nil, errors.Wrap(err, "invalid client config")
		}
		if cfg.UseInstancePrincipals {
			glog.V(2).Info("Using instance principals configuration provider")
			cp, err := auth.InstancePrincipalConfigurationProvider()
			if err != nil {
				return nil, errors.Wrap(err, "InstancePrincipalConfigurationProvider")
			}
			return cp, nil
		}
		glog.V(2).Info("Using raw configuration provider")
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
