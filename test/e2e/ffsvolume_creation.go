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

	"github.com/oracle/oci-volume-provisioner/pkg/provisioner/core"
	"github.com/oracle/oci-volume-provisioner/pkg/provisioner/fss"
	"github.com/oracle/oci-volume-provisioner/test/e2e/framework"
)

var _ = Describe("FSS Volume Creation", func() {
	f := framework.NewDefaultFramework("fss-volume")

	It("Should be possible to create a persistent volume claim (PVC) for a FSS with a mnt target specified", func() {
		pvcJig := framework.NewPVCTestJig(f.ClientSet, "volume-provisioner-e2e-tests-pvc")

		scName := f.CreateStorageClassOrFail(framework.ClassOCIMntFss, core.ProvisionerNameFss, map[string]string{
			fss.MntTargetID: framework.TestContext.MntTargetOCID}, pvcJig.Labels)

		By("Creating PVC that will dynamically provision a FSS")
		pvc := pvcJig.CreateAndAwaitPVCOrFail(f.Namespace.Name, framework.VolumeFss, scName, nil)

		By("Creating a Pod and waiting till attaches to the volume")
		pvcJig.CheckVolumeReadWrite(f.Namespace.Name, pvc)
	})

	It("Should be possible to create a persistent volume claim (PVC) for a FSS with a subnet id specified", func() {
		pvcJig := framework.NewPVCTestJig(f.ClientSet, "volume-provisioner-e2e-tests-pvc")

		scName := f.CreateStorageClassOrFail(framework.ClassOCISubnetFss, core.ProvisionerNameFss, map[string]string{
			fss.SubnetID: framework.TestContext.SubnetOCID}, pvcJig.Labels)

		By("Creating PVC that will dynamically provision a FSS")
		pvc := pvcJig.CreateAndAwaitPVCOrFail(f.Namespace.Name, framework.VolumeFss, scName, nil)

		By("Creating a Pod and waiting till attaches to the volume")
		pvcJig.CheckVolumeReadWrite(f.Namespace.Name, pvc)
	})

	It("Should be possible to create a persistent volume claim (PVC) for a FSS no mnt target or subnet id specified", func() {
		pvcJig := framework.NewPVCTestJig(f.ClientSet, "volume-provisioner-e2e-tests-pvc")

		scName := f.CreateStorageClassOrFail(framework.ClassOCINoParamFss, core.ProvisionerNameFss, nil, pvcJig.Labels)

		By("Creating PVC that will dynamically provision a FSS")
		pvc := pvcJig.CreateAndAwaitPVCOrFail(f.Namespace.Name, framework.VolumeFss, scName, nil)

		By("Creating a Pod and waiting till attaches to the volume")
		pvcJig.CheckVolumeReadWrite(f.Namespace.Name, pvc)
	})

})
