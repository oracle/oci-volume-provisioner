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
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	coreOCI "github.com/oracle/oci-go-sdk/core"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/oracle/oci-volume-provisioner/pkg/provisioner/block"
)

// CreatePVCAndBackupOrFail calls CreateAndAwaitPVCOrFail creates a new PVC based on the
// jig's defaults, waits for it to become ready. The volume is then backed up.
func (j *PVCTestJig) CreatePVCAndBackupOrFail(storageClient coreOCI.BlockstorageClient, namespace string, volumeSize string, tweak func(pvc *v1.PersistentVolumeClaim)) (*v1.PersistentVolumeClaim, string) {
	pvc := j.CreateAndAwaitPVCOrFail(namespace, volumeSize, tweak)
	backupVolumeID, err := j.BackupVolume(storageClient, pvc)
	if err != nil {
		Failf("Failed to created backup for pvc %q: %v", pvc.Name, err)
	}
	return pvc, *backupVolumeID
}

// BackupVolume creates a volume backup on OCI from an exsiting volume and returns the backup volume id
func (j *PVCTestJig) BackupVolume(storageClient coreOCI.BlockstorageClient, pvc *v1.PersistentVolumeClaim) (*string, error) {
	By("Creating backup of the volume")
	ctx := context.Background()
	pvc, err := j.KubeClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Get(pvc.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	pv, err := j.KubeClient.CoreV1().PersistentVolumes().Get(pvc.Spec.VolumeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("Failed to get persistent volume created name by claim %q: %v", pvc.Spec.VolumeName, err)
	}
	volumeID := pv.ObjectMeta.Annotations[block.OCIVolumeID]
	backupVolume, err := storageClient.CreateVolumeBackup(ctx, coreOCI.CreateVolumeBackupRequest{
		CreateVolumeBackupDetails: coreOCI.CreateVolumeBackupDetails{
			VolumeId:    &volumeID,
			DisplayName: &j.Name,
			Type:        coreOCI.CreateVolumeBackupDetailsTypeFull,
		},
	})
	if err != nil {
		return backupVolume.Id, fmt.Errorf("Failed to backup volume with ocid %q: %v", volumeID, err)
	}

	err = j.waitForVolumeAvailable(ctx, storageClient, backupVolume.Id, DefaultTimeout)
	if err != nil {
		// Delete the volume if it failed to get in a good state for us
		ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
		defer cancel()

		_, _ = storageClient.DeleteVolumeBackup(ctx,
			coreOCI.DeleteVolumeBackupRequest{VolumeBackupId: backupVolume.Id})

		return backupVolume.Id, err
	}
	return backupVolume.Id, nil
}

func (j *PVCTestJig) waitForVolumeAvailable(ctx context.Context, storageClient coreOCI.BlockstorageClient, volumeID *string, timeout time.Duration) error {
	isVolumeReady := func() (bool, error) {
		ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
		defer cancel()

		getVolumeResponse, err := storageClient.GetVolumeBackup(ctx,
			coreOCI.GetVolumeBackupRequest{VolumeBackupId: volumeID})
		if err != nil {
			return false, err
		}

		state := getVolumeResponse.LifecycleState
		Logf("State: %q", state)
		switch state {
		case coreOCI.VolumeBackupLifecycleStateCreating:
			return false, nil
		case coreOCI.VolumeBackupLifecycleStateAvailable:
			return true, nil
		case coreOCI.VolumeBackupLifecycleStateFaulty,
			coreOCI.VolumeBackupLifecycleStateTerminated,
			coreOCI.VolumeBackupLifecycleStateTerminating:
			return false, fmt.Errorf("volume has lifecycle state %q", state)
		}
		return false, nil
	}

	return wait.PollImmediate(time.Second*5, timeout, func() (bool, error) {
		ready, err := isVolumeReady()
		if err != nil {
			return false, fmt.Errorf("failed to provision volume %q: %v", *volumeID, err)
		}
		return ready, nil
	})

}

// DeleteBackup deletes the backup after a volume has been restored.
func (j *PVCTestJig) DeleteBackup(storageClient coreOCI.BlockstorageClient, backupID *string) {
	ctx := context.Background()
	storageClient.DeleteVolumeBackup(ctx,
		coreOCI.DeleteVolumeBackupRequest{VolumeBackupId: backupID})
}
