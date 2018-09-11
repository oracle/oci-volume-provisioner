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

	"github.com/oracle/oci-volume-provisioner/pkg/provisioner/block"
	"github.com/oracle/oci-volume-provisioner/pkg/provisioner/core"
	"github.com/oracle/oci-volume-provisioner/pkg/provisioner/plugin"
	"github.com/oracle/oci-volume-provisioner/test/e2e/framework"
)

var _ = Describe("Block Volume Creation", func() {
	f := framework.NewDefaultFramework("block-volume")

	It("Should be possible to create a persistent volume claim for a block storage (PVC)", func() {
		pvcJig := framework.NewPVCTestJig(f.ClientSet, "volume-provisioner-e2e-tests-pvc")
		pvcJig.CreateAndAwaitPVCOrFail(f.Namespace.Name, framework.MinVolumeBlock, func(pvc *v1.PersistentVolumeClaim) {
			pvc.Spec.Selector = &metav1.LabelSelector{MatchLabels: map[string]string{
				plugin.LabelZoneFailureDomain: framework.DefaultAD}}
			pvcJig.StorageClassName = framework.ClassOCI
			pvcJig.CheckSCorCreate(pvcJig.StorageClassName, core.ProvisionerNameDefault, nil)
			pvc.Spec.StorageClassName = &pvcJig.StorageClassName
		})

		if pvcJig.CustomStorageClass && framework.DeleteNamespaceRegisterFlag() {
			pvcJig.DeleteStorageClass(pvcJig.StorageClassName)
		}
	})

	It("Should be possible to create a persistent volume claim (PVC) for a block storage of Ext3 file system ", func() {
		pvcJig := framework.NewPVCTestJig(f.ClientSet, "volume-provisioner-e2e-tests-pvc")
		pvcJig.CreateAndAwaitPVCOrFail(f.Namespace.Name, framework.MinVolumeBlock, func(pvc *v1.PersistentVolumeClaim) {
			pvc.Spec.Selector = &metav1.LabelSelector{MatchLabels: map[string]string{
				plugin.LabelZoneFailureDomain: framework.DefaultAD}}
			pvcJig.StorageClassName = framework.ClassOCIExt3
			pvcJig.CheckSCorCreate(pvcJig.StorageClassName, core.ProvisionerNameDefault, map[string]string{block.FsType: "ext3"})
			pvc.Spec.StorageClassName = &pvcJig.StorageClassName
		})

		if pvcJig.CustomStorageClass && framework.DeleteNamespaceRegisterFlag() {
			pvcJig.DeleteStorageClass(pvcJig.StorageClassName)
		}
	})

	It("Should be possible to create a persistent volume claim (PVC) for a block storage with no AD specified ", func() {
		pvcJig := framework.NewPVCTestJig(f.ClientSet, "volume-provisioner-e2e-tests-pvc")
		pvcJig.CreateAndAwaitPVCOrFail(f.Namespace.Name, framework.MinVolumeBlock, func(pvc *v1.PersistentVolumeClaim) {
			pvcJig.StorageClassName = framework.ClassOCI
			pvcJig.CheckSCorCreate(pvcJig.StorageClassName, core.ProvisionerNameDefault, nil)
			pvc.Spec.StorageClassName = &pvcJig.StorageClassName
		})

		if pvcJig.CustomStorageClass && framework.DeleteNamespaceRegisterFlag() {
			pvcJig.DeleteStorageClass(pvcJig.StorageClassName)
		}
	})

})
