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

	"github.com/oracle/oci-volume-provisioner/test/e2e/framework"
)

var _ = Describe("Backup/Restore", func() {
	f := framework.NewBackupFramework("backup-restore")

	It("should be possible to backup a volume and restore the created backup", func() {
		pvcJig := framework.NewPVCTestJig(f.ClientSet, "volume-provisioner-e2e-tests-pvc")
		By("Provisioning volume to backup")
		pvc, backupID := pvcJig.CreatePVCAndBackupOrFail(f.BlockStorageClient, f.Namespace.Name, "50Gi", func(pvc *v1.PersistentVolumeClaim) {
			pvcJig.StorageClassName = "oci"
			if !pvcJig.CheckStorageClass(pvcJig.StorageClassName) {
				pvcJig.StorageClassName = pvcJig.CreateStorageClassOrFail(pvcJig.StorageClassName, "oracle.com/oci", nil)
			}
			pvc.Spec.StorageClassName = &pvcJig.StorageClassName
		})
		framework.Logf("pvc %q has been backed up with the following id %q", pvc.Name, &backupID)
		By("Teardown volume")
		pvcJig.DeletePersistentVolumeClaim(pvc.Name, f.Namespace.Name)
		time.Sleep(30 * time.Second)
		By("Checking that the volume has been teared down")
		if pvcJig.EnsureDeletion(pvc.Name, f.Namespace.Name) {
			pvcJig.DeletePersistentVolumeClaim(pvc.Name, f.Namespace.Name)
		}
		By("Restoring the backup")
		pvcJig.CreateAndAwaitPVCOrFail(f.Namespace.Name, "50Gi", func(pvcRestore *v1.PersistentVolumeClaim) {
			pvcRestore.ObjectMeta.Name = pvc.Name + "-restored"
			pvcRestore.ObjectMeta.Annotations = map[string]string{
				"volume.beta.kubernetes.io/oci-volume-source": backupID,
			}
			pvcJig.StorageClassName = "oci"
			if !pvcJig.CheckStorageClass(pvcJig.StorageClassName) {
				pvcJig.StorageClassName = pvcJig.CreateStorageClassOrFail(pvcJig.StorageClassName, "oracle.com/oci", nil)
			}
			pvcRestore.Spec.StorageClassName = &pvcJig.StorageClassName
		})
		//By("Checking the volume is present and contains the correct uuid")
		//pvcJig.CheckBackupUUID(backupID, pvcRestore)
		pvcJig.DeleteBackup(f.BlockStorageClient, &backupID)
	})
})
