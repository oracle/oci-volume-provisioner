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
	"time"

	. "github.com/onsi/ginkgo"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/oracle/oci-volume-provisioner/pkg/provisioner/block"
	"github.com/oracle/oci-volume-provisioner/pkg/provisioner/core"
	"github.com/oracle/oci-volume-provisioner/pkg/provisioner/plugin"
	"github.com/oracle/oci-volume-provisioner/test/e2e/framework"
)

var _ = Describe("Backup/Restore", func() {
	f := framework.NewBackupFramework("backup-restore")

	It("should be possible to backup a volume and restore the created backup", func() {
		pvcJig := framework.NewPVCTestJig(f.ClientSet, "volume-provisioner-e2e-tests-pvc")
		By("Provisioning volume to backup")
		pvc, backupID := pvcJig.CreatePVCAndBackupOrFail(f.BlockStorageClient, f.Namespace.Name, framework.MinVolumeBlock, func(pvc *v1.PersistentVolumeClaim) {
			pvc.Spec.Selector = &metav1.LabelSelector{MatchLabels: map[string]string{
				plugin.LabelZoneFailureDomain: f.CheckEnvVar(framework.AD)}}
			pvcJig.StorageClassName = framework.ClassOCI
			pvcJig.CheckSCorCreate(pvcJig.StorageClassName, core.ProvisionerNameDefault, nil)
			pvc.Spec.StorageClassName = &pvcJig.StorageClassName
		})
		f.BackupIDs = append(f.BackupIDs, backupID)
		framework.Logf("pvc %q has been backed up with the following id %q", pvc.Name, &backupID)
		By("Teardown volume")
		pvcJig.DeletePersistentVolumeClaim(pvc.Name, f.Namespace.Name)
		time.Sleep(30 * time.Second)
		By("Checking that the volume has been teared down")
		if pvcJig.EnsureDeletion(pvc.Name, f.Namespace.Name) {
			pvcJig.DeletePersistentVolumeClaim(pvc.Name, f.Namespace.Name)
		}
		By("Restoring the backup")
		pvcRestored := pvcJig.CreateAndAwaitPVCOrFail(f.Namespace.Name, framework.MinVolumeBlock, func(pvcRestore *v1.PersistentVolumeClaim) {
			pvcRestore.ObjectMeta.Name = pvc.Name + "-restored"
			pvcRestore.ObjectMeta.Annotations = map[string]string{
				block.OCIVolumeBackupID: backupID,
			}
			pvcRestore.Spec.Selector = &metav1.LabelSelector{MatchLabels: map[string]string{
				plugin.LabelZoneFailureDomain: f.CheckEnvVar(framework.AD)}}
			pvcJig.StorageClassName = framework.ClassOCI
			pvcJig.CheckSCorCreate(pvcJig.StorageClassName, core.ProvisionerNameDefault, nil)
			pvcRestore.Spec.StorageClassName = &pvcJig.StorageClassName
		})
		pvcJig.CreateAndAwaitNginxPodOrFail(f.Namespace.Name, pvcRestored)

		pvcJig.DeleteBackup(f.BlockStorageClient, &backupID)
	})
})
