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
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// PVCTestJig is a jig to help create PVC tests.
type PVCTestJig struct {
	ID     string
	Name   string
	Labels map[string]string

	StorageClass *storagev1beta1.StorageClass

	KubeClient clientset.Interface
}

// NewPVCTestJig allocates and inits a new PVCTestJig.
func NewPVCTestJig(kubeClient clientset.Interface, storageClass *storagev1beta1.StorageClass, name string) *PVCTestJig {
	id := storageClass.Labels["testID"]
	return &PVCTestJig{
		ID:   id,
		Name: name,
		Labels: map[string]string{
			"testID":   id,
			"testName": name,
		},
		StorageClass: storageClass,
		KubeClient:   kubeClient,
	}
}

// newPVCTemplate returns the default template for this jig, but
// does not actually create the PVC.  The default PVC has the same name
// as the jig
func (j *PVCTestJig) newPVCTemplate(namespace string, annotations map[string]string, selectorLabel map[string]string, volumeSize string, accessMode []v1.PersistentVolumeAccessMode) *v1.PersistentVolumeClaim {
	if len(accessMode) == 0 {
		Logf("AccessModes unspecified, default: RWO.")
		accessMode = append(accessMode, v1.ReadWriteOnce)
	}

	return &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   namespace,
			Name:        j.Name + "-" + j.ID,
			Labels:      j.Labels,
			Annotations: annotations,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			// TODO (bl) - refer to oci-config
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabel,
			},
			AccessModes: accessMode,
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceName(v1.ResourceStorage): resource.MustParse(volumeSize),
				},
			},
			StorageClassName: &j.StorageClass.Name,
		},
	}
}

// CreatePVCorFail creates a new claim based on the jig's
// defaults. Callers can provide a function to tweak the claim object
// before it is created.
func (j *PVCTestJig) CreatePVCorFail(namespace string, volumeSize string, selectorLabel map[string]string, tweak func(pvc *v1.PersistentVolumeClaim)) *v1.PersistentVolumeClaim {
	pvc := j.newPVCTemplate(namespace, nil, selectorLabel, volumeSize, nil)
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
func (j *PVCTestJig) CreateAndAwaitPVCOrFail(namespace string, volumeSize string, selectorLabel map[string]string, tweak func(cluster *v1.PersistentVolumeClaim)) *v1.PersistentVolumeClaim {
	pvc := j.CreatePVCorFail(namespace, volumeSize, selectorLabel, tweak)
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
	By("checking the claim")
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
func (j *PVCTestJig) TestCleanup(ns string, pvc *v1.PersistentVolumeClaim, sc *storagev1beta1.StorageClass) []error {
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
	if sc != nil {
		err := j.DeleteStorageClass(sc.Name)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to delete Storage Class %q: %v", sc.Name, err))
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
