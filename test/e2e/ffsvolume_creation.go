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

var _ = Describe("FSS Volume Creation", func() {
	f := framework.NewDefaultFramework("fss-volume")

	It("Should be possible to create a persistent volume claim (PVC) for a FSS with a mnt target specified ", func() {
		classJig := framework.NewStorageClassTestJig(f.ClientSet, "oci-storageclass-fss-mnt")
		storageClass := classJig.CreateStorageClassOrFail(f.Namespace.Name, "oracle.com/oci-fss", map[string]string{
			"mntTargetId": "ocid1.mounttarget.oc1.phx.aaaaaa4np2snlxveobuhqllqojxwiotqnb4c2ylefuzaaaaa",
		})

		pvcJig := framework.NewPVCTestJig(f.ClientSet, storageClass, "volume-provisioner-e2e-tests-pvc")
		// TO-DO (bl) - refer to config yaml for specific ad, or specify somewhere in framework
		// TO-DO (bl) - need to add a tweak function to check the if the a pod is attached
		pvc := pvcJig.CreateAndAwaitPVCOrFail(f.Namespace.Name, "1Gi", map[string]string{
			"failure-domain.beta.kubernetes.io/zone": "PHX-AD-2"}, nil)

		framework.CreateAndAwaitNginxPodOrFail(f.ClientSet, f.Namespace.Name, nil, pvc)

		if framework.DeleteNamespaceRegisterFlag() {
			pvcJig.TestCleanup(f.Namespace.Name, pvc, storageClass)
			// TO-DO (bl) - or only since pvc and pv cleaned by namespace deletion
			// TO-DO (bl) - pvcJig.DeleteStorageClass(storageClass.Name)
			// TO-DO (bl) - still need to think about secret/service account and provisioner/pod cleanup --> should be namespaced
		}
	})

	It("Should be possible to create a persistent volume claim (PVC) for a FSS with a subnet id specified", func() {
		classJig := framework.NewStorageClassTestJig(f.ClientSet, "oci-storageclass-fss-subnet-id")
		storageClass := classJig.CreateStorageClassOrFail(f.Namespace.Name, "oracle.com/oci-fss", map[string]string{
			"subnetId": "ocid1.subnet.oc1.phx.aaaaaaaanlsnbcixkkchz6n6eznusplxui3xwgb7bsaeucqy4zpehohcb3ra",
		})

		pvcJig := framework.NewPVCTestJig(f.ClientSet, storageClass, "volume-provisioner-e2e-tests-pvc")
		// TO-DO (bl) - maybe refer to config yaml for specific ad, or specify somewhere in framework
		// TO-DO (bl) - need to add a tweak function to check the if the a pod is attached
		pvc := pvcJig.CreateAndAwaitPVCOrFail(f.Namespace.Name, "1Gi", map[string]string{
			"failure-domain.beta.kubernetes.io/zone": "PHX-AD-2"}, nil)

		framework.CreateAndAwaitNginxPodOrFail(f.ClientSet, f.Namespace.Name, nil, pvc)

		if framework.DeleteNamespaceRegisterFlag() {
			pvcJig.TestCleanup(f.Namespace.Name, pvc, storageClass)
			// TO-DO (bl) - or only since pvc and pv cleaned by namespace deletion
			// TO-DO (bl) - pvcJig.DeleteStorageClass(storageClass.Name)
			// TO-DO (bl) - still need to think about secret/service account and provisioner/pod cleanup
		}
	})
})
