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
	"k8s.io/apimachinery/pkg/util/uuid"
	clientset "k8s.io/client-go/kubernetes"
)

// StorageClassTestJig is a jig to help create storage classes to be used when creating PVC tests.
type StorageClassTestJig struct {
	ID     string
	Name   string
	Labels map[string]string

	KubeClient clientset.Interface
}

// NewStorageClassTestJig allocates and inits a new StorageClassTestJig
func NewStorageClassTestJig(kubeClient clientset.Interface, name string) *StorageClassTestJig {
	id := string(uuid.NewUUID())
	return &StorageClassTestJig{
		ID:   id,
		Name: name,
		Labels: map[string]string{
			"testID":   id,
			"testName": name,
		},
		KubeClient: kubeClient,
	}
}

// NewStorageClassTemplate returns the default template for this jig, but
// does not actually create the storage class. The default storage class has the same name
// as the jig
func (j *StorageClassTestJig) NewStorageClassTemplate(namespace string, provisionerType string, parameters map[string]string) *storagev1beta1.StorageClass {
	return &storagev1beta1.StorageClass{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StorageClass",
			APIVersion: "storage.k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   namespace,
			Name:        j.Name + "-" + j.ID,
			Labels:      j.Labels,
			Annotations: nil,
		},
		Provisioner: provisionerType,
		Parameters:  parameters,
	}
}

// CreateStorageClassOrFail creates a new storage class based on the jig's defaults.
func (j *StorageClassTestJig) CreateStorageClassOrFail(namespace string, provisionerType string, parameters map[string]string) *storagev1beta1.StorageClass {
	class := j.NewStorageClassTemplate(namespace, provisionerType, parameters)

	result, err := j.KubeClient.StorageV1beta1().StorageClasses().Create(class)
	if err != nil {
		Failf("Failed to create storage class %q: %v", j.Name, err)
	}
	return result
}

// DeleteStorageClass will delete a class
func (j *StorageClassTestJig) DeleteStorageClass(scName string) error {
	if j.KubeClient != nil && len(scName) > 0 {
		Logf("Deleting Storage Class %q", scName)
		err := j.KubeClient.StorageV1beta1().StorageClasses().Delete(scName, nil)
		if err != nil && !apierrs.IsNotFound(err) {
			return fmt.Errorf("Storage Class Delete API error: %v", err)
		}
	}
	return nil
}
