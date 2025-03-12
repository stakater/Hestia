/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type Mode string

// RunnerSpec defines the desired state of Runner
type RunnerSpec struct {
	DeploymentSelector *metav1.LabelSelector `json:"deploymentSelector,omitempty"`
	RunnerSelector     *metav1.LabelSelector `json:"runnerSelector,omitempty"`

	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Template v1.PodTemplateSpec `json:"template"`
	JobName  string             `json:"jobName,omitempty"`
}

// RunnerStatus defines the observed state of Runner
type RunnerStatus struct {
	Success bool `json:"success,omitempty"`
	Failed  bool `json:"failed,omitempty"`
	Pending bool `json:"pending,omitempty"`

	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	ResourceGeneration    int64            `json:"resourceGeneration,omitempty"`
	RunnerGeneration      map[string]int64 `json:"runnerGeneration,omitempty"`
	DeploymentsGeneration map[string]int64 `json:"deploymentsGeneration,omitempty"`
	ExecutionGeneration   int64            `json:"executionGeneration,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Runner is the Schema for the runners API
type Runner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RunnerSpec   `json:"spec,omitempty"`
	Status RunnerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RunnerList contains a list of Runner
type RunnerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Runner `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Runner{}, &RunnerList{})
}
