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
	. "github.com/onsi/ginkgo"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (f *Framework) checkSecret(namespace string, secretName string) bool {
	//_, err := f.ClientSet.CoreV1().Secrets(namespace).Get(secretName, metav1.GetOptions{})
	_, err := f.ClientSet.CoreV1().ServiceAccounts(namespace).Get(secretName, metav1.GetOptions{})
	if err != nil {
		Logf("Secret %q not found: %v", secretName, err)
		return false
	}
	return true
}

func (f *Framework) installFSSProvisioner(namespace string) error {
	_, err := f.ClientSet.AppsV1beta1().Deployments(KubeSystemNS).Get("oci-volume-provisioner-fss", metav1.GetOptions{})
	if err != nil {
		Logf("Error getting oci-volume-provisioner-fss: %v", err)
		if !f.checkSecret(KubeSystemNS, "oci-volume-provisioner") {
			By("Creating oci-volume-provisioner secret")
			KubectlCmd("-n" + namespace + "create secret generic oci-volume-provisioner --from-file=config.yaml=" + f.CheckOCIConfig())
		}
		By("Installing OCI FSS volume provisioner")
		KubectlCmd("create -f " + TestContext.RepoRoot + "dist/oci-volume-provisioner-fss.yaml")
	} else {
		f.ProvisionerFSSInstalled = true
		return nil
	}
	return nil
}

func (f *Framework) installBlockProvisioner(namespace string) error {
	_, err := f.ClientSet.AppsV1beta1().Deployments(KubeSystemNS).Get("oci-volume-provisioner", metav1.GetOptions{})
	if err != nil {
		Logf("Error getting oci-volume-provisioner: %v", err)
		if !f.checkSecret(KubeSystemNS, "oci-volume-provisioner") {
			By("Creating oci-volume-provisioner secret")
			KubectlCmd("-n" + namespace + "create secret generic oci-volume-provisioner --from-file=config.yaml=" + f.CheckOCIConfig())
		}
		By("Installing OCI volume provisioner")
		KubectlCmd("create -f " + TestContext.RepoRoot + "dist/oci-volume-provisioner.yaml")
		//Install the provisioner
	} else {
		f.ProvisionerBlockInstalled = true
		return nil
	}
	return nil
}
