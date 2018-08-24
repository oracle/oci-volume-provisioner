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

package e2e

import (
	. "github.com/onsi/ginkgo"

	"github.com/oracle/oci-volume-provisioner/test/e2e/framework"
)

var _ = Describe("Block Volume Creation", func() {
	f := framework.NewDefaultFramework("block-volume")

	It("Should be possible to create a persistent volume claim for a block storage (PVC)", func() {
		classJig := framework.NewStorageClassTestJig(f.ClientSet, "oci-storageclass")
		storageClass := classJig.CreateStorageClassOrFail(f.Namespace.Name, "oracle.com/oci", nil)

		pvcJig := framework.NewPVCTestJig(f.ClientSet, storageClass, "volume-provisioner-e2e-tests-pvc")
		// TO-DO (bl) - refer to config yaml for specific ad, or specify somewhere in framework
		pvc := pvcJig.CreateAndAwaitPVCOrFail(f.Namespace.Name, "50Gi", map[string]string{
			"failure-domain.beta.kubernetes.io/zone": "PHX-AD-1"}, nil)

		if framework.DeleteNamespaceRegisterFlag() {
			pvcJig.TestCleanup(f.Namespace.Name, pvc, storageClass)
			// TO-DO (bl) - or only since pvc and pv cleaned by namespace deletion
			// TO-DO (bl) - pvcJig.DeleteStorageClass(storageClass.Name)
			// TO-DO (bl) - still need to think about secret/service account and provisioner/pod cleanup --> should be namespaced
		}
		// TO-DO (bl) - compare expected and actual

	})
	It("Should be possible to create a persistent volume claim (PVC) for a block storage of Ext3 file system ", func() {
		classJig := framework.NewStorageClassTestJig(f.ClientSet, "oci-storageclass-ext3")
		storageClass := classJig.CreateStorageClassOrFail(f.Namespace.Name, "oracle.com/oci", map[string]string{
			"fsType": "ext3",
		})

		pvcJig := framework.NewPVCTestJig(f.ClientSet, storageClass, "volume-provisioner-e2e-tests-pvc")
		// TO-DO (bl) - refer to config yaml for specific ad, or specify somewhere in framework
		pvc := pvcJig.CreateAndAwaitPVCOrFail(f.Namespace.Name, "50Gi", map[string]string{
			"failure-domain.beta.kubernetes.io/zone": "PHX-AD-1"}, nil)

		if framework.DeleteNamespaceRegisterFlag() {
			pvcJig.TestCleanup(f.Namespace.Name, pvc, storageClass)
			// TO-DO (bl) - or only since pvc and pv cleaned by namespace deletion
			// TO-DO (bl) - pvcJig.DeleteStorageClass(storageClass.Name)
			// TO-DO (bl) - still need to think about secret/service account and provisioner/pod cleanup --> should be namespaced
		}
	})

	It("Should be possible to create a persistent volume claim (PVC) for a block storage with no AD specified ", func() {
		classJig := framework.NewStorageClassTestJig(f.ClientSet, "oci-storageclass-no-ad")
		storageClass := classJig.CreateStorageClassOrFail(f.Namespace.Name, "oracle.com/oci", nil)

		pvcJig := framework.NewPVCTestJig(f.ClientSet, storageClass, "volume-provisioner-e2e-tests-pvc")
		//maybe refer to config yaml for specific ad, or specify somewhere in framework
		pvc := pvcJig.CreateAndAwaitPVCOrFail(f.Namespace.Name, "50Gi", nil, nil)

		if framework.DeleteNamespaceRegisterFlag() {
			pvcJig.TestCleanup(f.Namespace.Name, pvc, storageClass)
			// TO-DO (bl) - or only since pvc and pv cleaned by namespace deletion
			// TO-DO (bl) - pvcJig.DeleteStorageClass(storageClass.Name)
			// TO-DO (bl) - still need to think about secret/service account and provisioner/pod cleanup --> should be namespaced
		}
	})

})
