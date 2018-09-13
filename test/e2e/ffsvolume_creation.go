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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/oracle/oci-volume-provisioner/pkg/provisioner/core"
	"github.com/oracle/oci-volume-provisioner/pkg/provisioner/fss"
	"github.com/oracle/oci-volume-provisioner/pkg/provisioner/plugin"
	"github.com/oracle/oci-volume-provisioner/test/e2e/framework"
)

var _ = Describe("FSS Volume Creation", func() {
	f := framework.NewDefaultFramework("fss-volume")

	It("Should be possible to create a persistent volume claim (PVC) for a FSS with a mnt target specified. ["+CMFssMnt+"]", func() {
		pvcJig := framework.NewPVCTestJig(f.ClientSet, "volume-provisioner-e2e-tests-pvc")
		By("Creating PVC that will dynamically provision a FSS")
		pvc := pvcJig.CreateAndAwaitPVCOrFail(f.Namespace.Name, framework.VolumeFss, func(pvc *v1.PersistentVolumeClaim) {
			pvc.Spec.Selector = &metav1.LabelSelector{MatchLabels: map[string]string{
				plugin.LabelZoneFailureDomain: f.CheckEnvVar(framework.AD)}}

			pvcJig.StorageClassName = framework.ClassOCIMntFss
			pvcJig.CheckSCorCreate(pvcJig.StorageClassName, core.ProvisionerNameFss, map[string]string{
				fss.MntTargetID: f.CheckEnvVar(framework.MntTargetOCID)})
			pvc.Spec.StorageClassName = &pvcJig.StorageClassName

		})
		By("Creating a Pod and waiting till attaches to the volume")
		pvcJig.CreateAndAwaitNginxPodOrFail(f.Namespace.Name, pvc)

		pvcJig.DeleteStorageClass(pvcJig.StorageClassName)

	})

	It("Should be possible to create a persistent volume claim (PVC) for a FSS with a subnet id specified. ["+CMFssSubnet+"]", func() {
		pvcJig := framework.NewPVCTestJig(f.ClientSet, "volume-provisioner-e2e-tests-pvc")
		By("Creating PVC that will dynamically provision a FSS")
		pvc := pvcJig.CreateAndAwaitPVCOrFail(f.Namespace.Name, framework.VolumeFss, func(pvc *v1.PersistentVolumeClaim) {
			pvc.Spec.Selector = &metav1.LabelSelector{MatchLabels: map[string]string{
				plugin.LabelZoneFailureDomain: f.CheckEnvVar(framework.AD)}}

			pvcJig.StorageClassName = framework.ClassOCISubnetFss
			pvcJig.CheckSCorCreate(pvcJig.StorageClassName, core.ProvisionerNameFss, map[string]string{
				fss.SubnetID: f.CheckEnvVar(framework.SubnetOCID)})
			pvc.Spec.StorageClassName = &pvcJig.StorageClassName

		})
		By("Creating a Pod and waiting till attaches to the volume")
		pvcJig.CreateAndAwaitNginxPodOrFail(f.Namespace.Name, pvc)

		pvcJig.DeleteStorageClass(pvcJig.StorageClassName)

	})

	It("Should be possible to create a persistent volume claim (PVC) for a FSS no mnt target or subnet id specified. ["+CMFssNoParam+"]", func() {
		pvcJig := framework.NewPVCTestJig(f.ClientSet, "volume-provisioner-e2e-tests-pvc")
		By("Creating PVC that will dynamically provision a FSS")
		pvc := pvcJig.CreateAndAwaitPVCOrFail(f.Namespace.Name, framework.VolumeFss, func(pvc *v1.PersistentVolumeClaim) {
			pvc.Spec.Selector = &metav1.LabelSelector{MatchLabels: map[string]string{
				plugin.LabelZoneFailureDomain: f.CheckEnvVar(framework.AD)}}
			pvcJig.StorageClassName = framework.ClassOCINoParamFss
			pvcJig.CheckSCorCreate(pvcJig.StorageClassName, core.ProvisionerNameFss, nil)
			pvc.Spec.StorageClassName = &pvcJig.StorageClassName
		})
		By("Creating a Pod and waiting till attaches to the volume")
		pvcJig.CreateAndAwaitNginxPodOrFail(f.Namespace.Name, pvc)

		pvcJig.DeleteStorageClass(pvcJig.StorageClassName)
	})

})
