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
	"time"

	. "github.com/onsi/ginkgo"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/pkg/client/conditions"
)

// CreateAndAwaitNginxPodOrFail creates a pod with a dymincally provisioned volume
func CreateAndAwaitNginxPodOrFail(client clientset.Interface, namespace string, nodeSelector map[string]string, pvc *v1.PersistentVolumeClaim) {
	pod := MakeNginxPod(namespace, nodeSelector, pvc)
	By("Creating a pod with the dynmically provisioned volume")
	pod, err := client.CoreV1().Pods(namespace).Create(pod)
	if err != nil {
		Failf("pod Create API error: %v", err)
	}
	// Waiting for pod to be running
	err = WaitForPodNameRunningInNamespace(client, pod.Name, namespace)
	if err != nil {
		Failf("pod %q is not Running: %v", pod.Name, err)
	}
	// Get fresh pod info
	pod, err = client.CoreV1().Pods(namespace).Get(pod.Name, metav1.GetOptions{})
	if err != nil {
		Failf("pod Get API error: %v", err)
	}
}

// MakeNginxPod returns a pod definition based on the namespace using nginx image
func MakeNginxPod(ns string, nodeSelector map[string]string, pvc *v1.PersistentVolumeClaim) *v1.Pod {
	podSpec := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "pvc-pod-tester-" + pvc.Labels["testID"],
			Namespace:    ns,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "write-pod",
					Image: "nginx",
					Ports: []v1.ContainerPort{
						{
							Name:          "http-server",
							ContainerPort: 80,
						},
					},
					VolumeMounts: []v1.VolumeMount{
						{
							Name:      "nginx-storage",
							MountPath: "/usr/share/nginx/html/",
						},
					},
				},
			},
			Volumes: []v1.Volume{
				{
					Name: "nginx-storage",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvc.Name,
						},
					},
				},
			},
		},
	}

	if nodeSelector != nil {
		podSpec.Spec.NodeSelector = nodeSelector
	}
	return podSpec
}

// WaitForPodNameRunningInNamespace aits default amount of time (PodStartTimeout) for the specified pod to become running.
// Returns an error if timeout occurs first, or pod goes in to failed state.
func WaitForPodNameRunningInNamespace(c clientset.Interface, podName, namespace string) error {
	return WaitTimeoutForPodRunningInNamespace(c, podName, namespace, slowPodStartTimeout)
}

// WaitTimeoutForPodRunningInNamespace comment
func WaitTimeoutForPodRunningInNamespace(c clientset.Interface, podName, namespace string, timeout time.Duration) error {
	return wait.PollImmediate(Poll, timeout, podRunning(c, podName, namespace))
}

func podRunning(c clientset.Interface, podName, namespace string) wait.ConditionFunc {
	return func() (bool, error) {
		pod, err := c.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		switch pod.Status.Phase {
		case v1.PodRunning:
			return true, nil
		case v1.PodFailed, v1.PodSucceeded:
			return false, conditions.ErrPodCompleted
		}
		return false, nil
	}
}
