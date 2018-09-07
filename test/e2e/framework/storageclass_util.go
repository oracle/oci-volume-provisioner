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

	storagev1beta1 "k8s.io/api/storage/v1beta1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

// CheckSCorCreate checks if a storage class exists, if not creates one
func (j *PVCTestJig) CheckSCorCreate(name string, provisionerType string, param map[string]string) {
	if !j.CheckStorageClass(name) {
		j.CreateStorageClassOrFail(name, provisionerType, param)
	}
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
	classTemp := j.newStorageClassTemplate(name, provisionerType, parameters)

	class, err := j.KubeClient.StorageV1beta1().StorageClasses().Create(classTemp)
	if err != nil {
		Failf("Failed to create storage class %q: %v", j.Name, err)
	}
	j.CustomStorageClass = true
	return class.Name
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
