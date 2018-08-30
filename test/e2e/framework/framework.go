// Copyright 2018 Oracle and/or its affiliates. All rights reserved.
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

package framework

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/common/auth"
	coreOCI "github.com/oracle/oci-go-sdk/core"
	"github.com/oracle/oci-volume-provisioner/pkg/oci/client"
)

const (
	configFilePath string = "/etc/oci/config.yaml"
)

// Framework is used in the execution of e2e tests.
type Framework struct {
	BaseName             string
	ProvisionerInstalled bool

	ClientSet          clientset.Interface
	BlockStorageClient coreOCI.BlockstorageClient

	Namespace          *v1.Namespace   // Every test has at least one namespace unless creation is skipped
	namespacesToDelete []*v1.Namespace // Some tests have more than one.

	// To make sure that this framework cleans up after itself, no matter what,
	// we install a Cleanup action before each test and clear it after.  If we
	// should abort, the AfterSuite hook should run all Cleanup actions.
	cleanupHandle CleanupActionHandle
}

// NewDefaultFramework constructs a new e2e test Framework with default options.
func NewDefaultFramework(baseName string) *Framework {
	f := NewFramework(baseName, nil)
	return f
}

// NewFramework constructs a new e2e test Framework.
func NewFramework(baseName string, client clientset.Interface) *Framework {
	f := &Framework{
		BaseName:  baseName,
		ClientSet: client,
	}

	BeforeEach(f.BeforeEach)
	AfterEach(f.AfterEach)

	return f
}

// NewBackupFramework constrycts a new e2e test Framework for the backup initialising a storage client to used to create a backup
func NewBackupFramework(baseName string) *Framework {
	f := NewFramework(baseName, nil)
	f.BlockStorageClient = createStorageClient()
	return f
}

// CreateNamespace creates a e2e test namespace.
func (f *Framework) CreateNamespace(baseName string, labels map[string]string) (*v1.Namespace, error) {
	if labels == nil {
		labels = map[string]string{}
	}

	namespaceObj := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("volume-provisioner-e2e-tests-%v-", baseName),
			Namespace:    "",
			Labels:       labels,
		},
		Status: v1.NamespaceStatus{},
	}

	// Be robust about making the namespace creation call.
	var got *v1.Namespace
	if err := wait.PollImmediate(Poll, 30*time.Second, func() (bool, error) {
		var err error
		got, err = f.ClientSet.CoreV1().Namespaces().Create(namespaceObj)
		if err != nil {
			Logf("Unexpected error while creating namespace: %v", err)
			return false, nil
		}
		return true, nil
	}); err != nil {
		return nil, err
	}

	if got != nil {
		f.namespacesToDelete = append(f.namespacesToDelete, got)
	}

	return got, nil
}

// DeleteNamespace deletes a given namespace and waits until its contents are
// deleted.
func (f *Framework) DeleteNamespace(namespace string, timeout time.Duration) error {
	startTime := time.Now()
	if err := f.ClientSet.CoreV1().Namespaces().Delete(namespace, nil); err != nil {
		if apierrors.IsNotFound(err) {
			Logf("Namespace %v was already deleted", namespace)
			return nil
		}
		return err
	}

	// wait for namespace to delete or timeout.
	err := wait.PollImmediate(Poll, timeout, func() (bool, error) {
		if _, err := f.ClientSet.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{}); err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			Logf("Error while waiting for namespace to be terminated: %v", err)
			return false, nil
		}
		return false, nil
	})

	// Namespace deletion timed out.
	if err != nil {
		return fmt.Errorf("namespace %v was not deleted with limit: %v", namespace, err)
	}

	Logf("namespace %v deletion completed in %s", namespace, time.Now().Sub(startTime))
	return nil
}

// InstallProvisioner installs the Volume Provisioner as a deployment into the given namespace
func (f *Framework) InstallProvisioner(namespace string) error {
	return nil

}

// BeforeEach gets a client and makes a namespace.
func (f *Framework) BeforeEach() {
	// The fact that we need this feels like a bug in ginkgo.
	// https://github.com/onsi/ginkgo/issues/222
	f.cleanupHandle = AddCleanupAction(f.AfterEach)

	if f.ClientSet == nil {
		By("Creating a kubernetes client")
		config, err := clientcmd.BuildConfigFromFlags("", TestContext.KubeConfig)
		Expect(err).NotTo(HaveOccurred())
		f.ClientSet, err = clientset.NewForConfig(config)
		Expect(err).NotTo(HaveOccurred())
	}

	if TestContext.Namespace == "" {
		By("Building a namespace api object")
		namespace, err := f.CreateNamespace(f.BaseName, map[string]string{
			"e2e-framework": f.BaseName,
		})
		Expect(err).NotTo(HaveOccurred())
		f.Namespace = namespace
	} else {
		By(fmt.Sprintf("Getting existing namespace %q", TestContext.Namespace))
		namespace, err := f.ClientSet.CoreV1().Namespaces().Get(TestContext.Namespace, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		f.Namespace = namespace
	}

	if !f.ProvisionerInstalled {
		err := f.InstallProvisioner(f.Namespace.Name)
		Expect(err).NotTo(HaveOccurred())
		f.ProvisionerInstalled = true
	}
}

// AfterEach deletes the namespace(s).
func (f *Framework) AfterEach() {
	RemoveCleanupAction(f.cleanupHandle)

	nsDeletionErrors := map[string]error{}

	// Whether to delete namespace is determined by 3 factors: delete-namespace flag, delete-namespace-on-failure flag and the test result
	// if delete-namespace set to false, namespace will always be preserved.
	// if delete-namespace is true and delete-namespace-on-failure is false, namespace will be preserved if test failed.
	if TestContext.DeleteNamespace && (TestContext.DeleteNamespaceOnFailure || !CurrentGinkgoTestDescription().Failed) {
		for _, ns := range f.namespacesToDelete {
			By(fmt.Sprintf("Destroying namespace %q for this suite.", ns.Name))
			if err := f.DeleteNamespace(ns.Name, 5*time.Minute); err != nil {
				nsDeletionErrors[ns.Name] = err
			}
		}
	}

	// if we had errors deleting, report them now.
	if len(nsDeletionErrors) != 0 {
		messages := []string{}
		for namespaceKey, namespaceErr := range nsDeletionErrors {
			messages = append(messages, fmt.Sprintf("Couldn't delete ns: %q: %s (%#v)", namespaceKey, namespaceErr, namespaceErr))
		}
		Failf(strings.Join(messages, ","))
	}
	f.ProvisionerInstalled = false
}

func createStorageClient() coreOCI.BlockstorageClient {
	By("Creating an OCI block storage client")
	configPath, ok := os.LookupEnv("OCICONFIG_VAR")
	if !ok {
		configPath = "/home/bdour/go/src/github.com/oracle/oci-volume-provisioner/cloud-config.yaml"
	}

	file, err := os.Open(configPath)
	if err != nil {
		glog.Fatalf("Unable to load volume provisioner configuration file: %v", configPath)
	}
	defer file.Close()
	cfg, err := client.LoadConfig(file)
	if err != nil {
		glog.Fatalf("Unable to load volume provisioner client: %v", err)
	}
	config, err := newConfigurationProvider(cfg)
	if err != nil {
		// TO-DO modify error message
		Logf("config %q, err: %v", config, err)
	}
	blockStorageClient, err := coreOCI.NewBlockstorageClientWithConfigurationProvider(config)
	if err != nil {
		// TO-DO modify error message
		Logf("config %q, err: %v", config, err)
	}
	/*By("client declared")

		config, err := clientcmd.BuildConfigFromFlags("", TestContext.KubeConfig)
		Expect(err).NotTo(HaveOccurred())
		f.BlockStorageClient, err = *coreOCI.NewBlockstorageClientWithConfigurationProvider(config)
		Expect(err).NotTo(HaveOccurred())

	conf := common.DefaultConfigProvider()
	blockStorageClient, err := coreOCI.NewBlockstorageClientWithConfigurationProvider(conf)
	Expect(err).NotTo(HaveOccurred())
	*/
	return blockStorageClient
}

func newConfigurationProvider(cfg *client.Config) (common.ConfigurationProvider, error) {
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
