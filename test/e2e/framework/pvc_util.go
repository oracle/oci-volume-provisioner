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

	v1 "k8s.io/api/core/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	coreOCI "github.com/oracle/oci-go-sdk/core"
)

// PVCTestJig is a jig to help create PVC tests.
type PVCTestJig struct {
	ID     string
	Name   string
	Labels map[string]string

	CustomStorageClass bool
	StorageClassName   string

	BlockStorageClient *coreOCI.BlockstorageClient
	KubeClient         clientset.Interface
}

// NewPVCTestJig allocates and inits a new PVCTestJig.
func NewPVCTestJig(kubeClient clientset.Interface, name string) *PVCTestJig {
	id := string(uuid.NewUUID())
	return &PVCTestJig{
		ID:   id,
		Name: name,
		Labels: map[string]string{
			"testID":   id,
			"testName": name,
		},
		CustomStorageClass: false,
		KubeClient:         kubeClient,
	}
}

// newPVCTemplate returns the default template for this jig, but
// does not actually create the PVC.  The default PVC has the same name
// as the jig
func (j *PVCTestJig) newPVCTemplate(namespace string, volumeSize string) *v1.PersistentVolumeClaim {
	return &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    namespace,
			GenerateName: j.Name,
			Labels:       j.Labels,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.ReadWriteOnce,
			},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceName(v1.ResourceStorage): resource.MustParse(volumeSize),
				},
			},
		},
	}
}

// CheckStorageClass verifies the storage class exists, if not creates a storage class
func (j *PVCTestJig) CheckStorageClass(name string) bool {
	list, err := j.KubeClient.StorageV1beta1().StorageClasses().List(metav1.ListOptions{})
	if err != nil {
		Failf("Error listing storage classes: %v", err)
	}

	for _, sc := range list.Items {
		if sc.Name == name {
			Logf("Storage class %q found", sc.Name)
			return true
		}
	}

	return false
}

// NewStorageClassTemplate returns the default template for this jig, but
// does not actually create the storage class. The default storage class has the same name
// as the jig
func (j *PVCTestJig) newStorageClassTemplate(name string, provisionerType string, parameters map[string]string) *storagev1beta1.StorageClass {
	return &storagev1beta1.StorageClass{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StorageClass",
			APIVersion: "storage.k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: j.Labels,
		},
		Provisioner: provisionerType,
		Parameters:  parameters,
	}
}

// CreateStorageClassOrFail creates a new storage class based on the jig's defaults.
func (j *PVCTestJig) CreateStorageClassOrFail(name string, provisionerType string, parameters map[string]string) string {
	class := j.newStorageClassTemplate(name, provisionerType, parameters)

	result, err := j.KubeClient.StorageV1beta1().StorageClasses().Create(class)
	if err != nil {
		Failf("Failed to create storage class %q: %v", j.Name, err)
	}
	j.CustomStorageClass = true
	return result.Name
}

// CreatePVCorFail creates a new claim based on the jig's
// defaults. Callers can provide a function to tweak the claim object
// before it is created.
func (j *PVCTestJig) CreatePVCorFail(namespace string, volumeSize string, tweak func(pvc *v1.PersistentVolumeClaim)) *v1.PersistentVolumeClaim {
	pvc := j.newPVCTemplate(namespace, volumeSize)
	if tweak != nil {
		tweak(pvc)
	}

	name := types.NamespacedName{Namespace: namespace, Name: j.Name}
	By(fmt.Sprintf("Creating a PVC %q of volume size %q", name, volumeSize))

	result, err := j.KubeClient.CoreV1().PersistentVolumeClaims(namespace).Create(pvc)
	if err != nil {
		Failf("Failed to create persistent volume claim %q: %v", name, err)
	}
	return result

}

// CreateAndAwaitPVCOrFail creates a new PVC based on the
// jig's defaults, waits for it to become ready, and then sanity checks it and
// its dependant resources. Callers can provide a function to tweak the
// PVC object before it is created.
func (j *PVCTestJig) CreateAndAwaitPVCOrFail(namespace string, volumeSize string, tweak func(pvc *v1.PersistentVolumeClaim)) *v1.PersistentVolumeClaim {
	pvc := j.CreatePVCorFail(namespace, volumeSize, tweak)
	pvc = j.waitForConditionOrFail(namespace, pvc.Name, DefaultTimeout, "to be dynamically provisioned", func(pvc *v1.PersistentVolumeClaim) bool {
		err := j.WaitForPVCPhase(v1.ClaimBound, namespace, pvc.Name)
		if err != nil {
			Failf("PVC %q did not become Bound: %v", pvc.Name, err)
			return false
		}
		return true
	})
	j.SanityCheckPV(pvc)
	return pvc
}

// CreatePVCAndBackupOrFail creates
func (j *PVCTestJig) CreatePVCAndBackupOrFail(storageClient coreOCI.BlockstorageClient, namespace string, volumeSize string, tweak func(pvc *v1.PersistentVolumeClaim)) (*v1.PersistentVolumeClaim, string) {
	pvc := j.CreateAndAwaitPVCOrFail(namespace, volumeSize, tweak)
	backupVolumeID, err := j.BackupVolume(storageClient, pvc)
	if err != nil {
		Failf("Failed to created backup for pvc %q: %v", pvc.Name, err)
	}
	return pvc, *backupVolumeID
}

// WaitForPVCPhase waits for a PersistentVolumeClaim to be in a specific phase or until timeout occurs, whichever comes first.
func (j *PVCTestJig) WaitForPVCPhase(phase v1.PersistentVolumeClaimPhase, ns string, pvcName string) error {
	Logf("Waiting up to %v for PersistentVolumeClaim %s to have phase %s", DefaultTimeout, pvcName, phase)
	for start := time.Now(); time.Since(start) < DefaultTimeout; time.Sleep(Poll) {
		pvc, err := j.KubeClient.CoreV1().PersistentVolumeClaims(ns).Get(pvcName, metav1.GetOptions{})
		if err != nil {
			Logf("Failed to get claim %q, retrying in %v. Error: %v", pvcName, Poll, err)
			continue
		} else {
			if pvc.Status.Phase == phase {
				Logf("PersistentVolumeClaim %s found and phase=%s (%v)", pvcName, phase, time.Since(start))
				return nil
			}
		}
		Logf("PersistentVolumeClaim %s found but phase is %s instead of %s.", pvcName, pvc.Status.Phase, phase)
	}
	return fmt.Errorf("PersistentVolumeClaim %s not in phase %s within %v", pvcName, phase, DefaultTimeout)
}

// SanityCheckPV checks basic properties of a given volume match
// our expectations.
func (j *PVCTestJig) SanityCheckPV(pvc *v1.PersistentVolumeClaim) {
	By("Checking the claim")
	pvc, err := j.KubeClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Get(pvc.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	// Get the bound PV
	pv, err := j.KubeClient.CoreV1().PersistentVolumes().Get(pvc.Spec.VolumeName, metav1.GetOptions{})
	if err != nil {
		Failf("Failed to get persistent volume %q: %v", pvc.Spec.VolumeName, err)
	}

	// Check sizes
	pvCapacity := pv.Spec.Capacity[v1.ResourceName(v1.ResourceStorage)]
	claimCapacity := pvc.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)]
	Expect(pvCapacity.Value()).To(Equal(claimCapacity.Value()), "pvCapacity is not equal to expectedCapacity")

	// Check PV properties
	By("checking the PV")
	expectedAccessModes := []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce}
	Expect(pv.Spec.AccessModes).To(Equal(expectedAccessModes))
	Expect(pv.Spec.ClaimRef.Name).To(Equal(pvc.ObjectMeta.Name))
	Expect(pv.Spec.ClaimRef.Namespace).To(Equal(pvc.ObjectMeta.Namespace))

	// The pv and pvc are both bound, but to each other?
	// Check that the PersistentVolume.ClaimRef matches the PVC
	if pv.Spec.ClaimRef == nil {
		Failf("PV %q ClaimRef is nil", pv.Name)
	}
	if pv.Spec.ClaimRef.Name != pvc.Name {
		Failf("PV %q ClaimRef's name (%q) should be %q", pv.Name, pv.Spec.ClaimRef.Name, pvc.Name)
	}
	if pvc.Spec.VolumeName != pv.Name {
		Failf("PVC %q VolumeName (%q) should be %q", pvc.Name, pvc.Spec.VolumeName, pv.Name)
	}
	if pv.Spec.ClaimRef.UID != pvc.UID {
		Failf("PV %q ClaimRef's UID (%q) should be %q", pv.Name, pv.Spec.ClaimRef.UID, pvc.UID)
	}
}

func (j *PVCTestJig) waitForConditionOrFail(namespace, name string, timeout time.Duration, message string, conditionFn func(*v1.PersistentVolumeClaim) bool) *v1.PersistentVolumeClaim {
	var pvc *v1.PersistentVolumeClaim
	pollFunc := func() (bool, error) {
		v, err := j.KubeClient.CoreV1().PersistentVolumeClaims(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if conditionFn(v) {
			pvc = v
			return true, nil
		}
		return false, nil
	}
	if err := wait.PollImmediate(Poll, timeout, pollFunc); err != nil {
		Failf("Timed out waiting for volume claim %q to %s", pvc.Name, message)
	}
	return pvc
}

// TestCleanup - Clean up a pv and pvc in a single pv/pvc test case.
// Note: delete errors are appended to []error so that we can attempt to delete both the pvc and pv.
func (j *PVCTestJig) TestCleanup(ns string, pvc *v1.PersistentVolumeClaim) []error {
	var errs []error

	if pvc != nil {
		err := j.DeletePersistentVolumeClaim(pvc.Name, ns)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to delete PVC %q: %v", pvc.Name, err))
		}
	} else {
		Logf("pvc is nil")
	}
	pv, err := j.KubeClient.CoreV1().PersistentVolumes().Get(pvc.Spec.VolumeName, metav1.GetOptions{})
	if err != nil {
		errs = append(errs, fmt.Errorf("Failed to get persistent volume created name by claim %q: %v", pvc.Spec.VolumeName, err))
	}
	if pv != nil {
		err := j.DeletePersistentVolume(pv.Name)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to delete PV %q: %v", pv.Name, err))
		}
	} else {
		Logf("pv is nil")
	}

	if len(j.StorageClassName) != 0 {
		err := j.DeleteStorageClass(j.StorageClassName)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to delete Storage Class %q: %v", j.StorageClassName, err))
		}
	} else {
		Logf("sc is nil")
	}
	return errs
}

// DeleteStorageClass will delete a class
func (j *PVCTestJig) DeleteStorageClass(scName string) error {
	if j.KubeClient != nil && len(scName) > 0 {
		Logf("Deleting Storage Class %q", scName)
		err := j.KubeClient.StorageV1beta1().StorageClasses().Delete(scName, nil)
		if err != nil && !apierrs.IsNotFound(err) {
			return fmt.Errorf("Storage Class Delete API error: %v", err)
		}
	}
	return nil
}

// DeletePersistentVolume - Delete the PV.
func (j *PVCTestJig) DeletePersistentVolume(pvName string) error {
	if j.KubeClient != nil && len(pvName) > 0 {
		Logf("Deleting PersistentVolume %q", pvName)
		err := j.KubeClient.CoreV1().PersistentVolumes().Delete(pvName, nil)
		if err != nil && !apierrs.IsNotFound(err) {
			return fmt.Errorf("PV Delete API error: %v", err)
		}
	}
	return nil
}

// DeletePersistentVolumeClaim - Delete the Claim
func (j *PVCTestJig) DeletePersistentVolumeClaim(pvcName string, ns string) error {
	if j.KubeClient != nil && len(pvcName) > 0 {
		Logf("Deleting PersistentVolumeClaim %q", pvcName)
		err := j.KubeClient.CoreV1().PersistentVolumeClaims(ns).Delete(pvcName, nil)
		if err != nil && !apierrs.IsNotFound(err) {
			return fmt.Errorf("PVC Delete API error: %v", err)
		}
	}
	return nil
}

// EnsureDeletion checks that the pvc and pv have been deleted.
func (j *PVCTestJig) EnsureDeletion(pvcName string, ns string) bool {
	_, err := j.KubeClient.CoreV1().PersistentVolumeClaims(ns).Get(pvcName, metav1.GetOptions{})
	if err != nil {
		return true
	}
	return false
}

// BackupVolume creates a volume backup on OCI from an exsiting volume and returns the backup volume id
func (j *PVCTestJig) BackupVolume(storageClient coreOCI.BlockstorageClient, pvc *v1.PersistentVolumeClaim) (*string, error) {
	ctx := context.Background()
	pvc, err := j.KubeClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Get(pvc.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	pv, err := j.KubeClient.CoreV1().PersistentVolumes().Get(pvc.Spec.VolumeName, metav1.GetOptions{})
	volumeID := pv.ObjectMeta.Annotations["ociVolumeID"]
	// volumeID := pv.ObjectMeta.Name
	if err != nil {
		return nil, fmt.Errorf("Failed to get persistent volume created name by claim %q: %v", pvc.Spec.VolumeName, err)
	}

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

// DeleteBackup deletes the backup
func (j *PVCTestJig) DeleteBackup(storageClient coreOCI.BlockstorageClient, backupID *string) {
	ctx := context.Background()
	storageClient.DeleteVolumeBackup(ctx,
		coreOCI.DeleteVolumeBackupRequest{VolumeBackupId: backupID})
}

/*
// CheckBackupUUID checks
func (j *PVCTestJig) CheckBackupUUID(backupID string, pvc *v1.PersistentVolumeClaim) {
	pvcRestore, err := j.KubeClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Get(pvc.Name, metav1.GetOptions{})
	if err != nil {
		Failf("Failed to get persistent volume claim %q: %v", pvc.Name, err)
	}
	pvRestore, err := j.KubeClient.CoreV1().PersistentVolumes().Get(pvcRestore.Spec.VolumeName, metav1.GetOptions{})
	if err != nil {
		Failf("Failed to get persistent volume created name by claim %q: %v", pvcRestore.Spec.VolumeName, err)
	}
	if pvRestore.Annotations["ociVolumeID"] != backupID {
		Failf("Failed to assign the id of the blockID: %s, assigned %s instead", backupID,
			pvRestore.Annotations["ociVolumeID"])
	}
}
*/
