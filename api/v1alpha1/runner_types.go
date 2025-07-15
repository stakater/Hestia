package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RunnerSpec defines the desired state of Runner
type RunnerSpec struct {
	WorkloadSelector *metav1.LabelSelector `json:"workloadSelector,omitempty"`
	RunnerSelector   *metav1.LabelSelector `json:"runnerSelector,omitempty"`

	JobName string `json:"jobName,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Template v1.PodTemplateSpec `json:"template"`

	// Cron schedule eg "* * * * *"
	Schedule string `json:"schedule,omitempty"`

	// Job deadline
	DeadlineSeconds int64 `json:"deadlineSeconds,omitempty"`
}

type WatchedResource struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Kind      string `json:"kind"`
	Ready     bool   `json:"ready"`
}

// RunnerStatus defines the observed state of Runner
type RunnerStatus struct {
	Conditions        `json:",inline"`
	LastSuccessfulRun metav1.Time       `json:"lastSuccessfulRunTime,omitempty"`
	LastFailedRun     metav1.Time       `json:"lastFailedRunTime,omitempty"`
	WatchedResources  []WatchedResource `json:"watchedResources,omitempty"`
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
