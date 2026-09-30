package status

import (
	ocp "github.com/openshift/api/apps/v1"
	"github.com/redhat-cop/operator-utils/pkg/util/apis"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/batch/v1"
	v13 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/stakater/hestia-operator/api/v1alpha1"
	"github.com/stakater/hestia-operator/internal/constants"
)

var (
	readyCheckRegistry = map[string]func(obj unstructured.Unstructured) bool{
		"Deployment": func(obj unstructured.Unstructured) bool {
			deployment := &v1.Deployment{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, deployment)
			if err != nil {
				return false
			}

			return IsDeploymentReady(deployment)
		},
		"StatefulSet": func(obj unstructured.Unstructured) bool {
			sts := &v1.StatefulSet{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, sts)
			if err != nil {
				return false
			}

			return IsStatefulSetReady(sts)
		},
		"Runner": func(obj unstructured.Unstructured) bool {
			runner := &v1alpha1.Runner{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, runner)
			if err != nil {
				return false
			}

			return IsRunnerReady(runner)
		},
		"Job": func(obj unstructured.Unstructured) bool {
			job := &v12.Job{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, job)
			if err != nil {
				return false
			}

			return IsJobReady(job)
		},
		"DeploymentConfig": func(obj unstructured.Unstructured) bool {
			dc := &ocp.DeploymentConfig{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, dc)
			if err != nil {
				return false
			}

			return IsDeploymentConfigReady(dc)
		},
		"DaemonSet": func(obj unstructured.Unstructured) bool {
			daemonset := &v1.DaemonSet{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, daemonset)
			if err != nil {
				return false
			}
			return IsDaemonSetReady(daemonset)
		},
	}
)

func IsRunnerReady(runner *v1alpha1.Runner) bool {
	if runner == nil {
		return false
	}

	lastRunCondition, ok := apis.GetCondition(constants.JobStatusType, runner.Status.Conditions.Conditions)
	if !ok {
		return false
	}

	return lastRunCondition.Reason == constants.SuccessfulRunReason
}

// desiredReplicas returns spec.replicas, which the API server defaults to 1 when unset
func desiredReplicas(replicas *int32) int32 {
	if replicas == nil {
		return 1
	}

	return *replicas
}

// IsDeploymentReady compares against the desired replicas: a new Deployment's first status reports 0 of 0
func IsDeploymentReady(deployment *v1.Deployment) bool {
	if deployment == nil {
		return false
	}

	desired := desiredReplicas(deployment.Spec.Replicas)
	return deployment.Status.ObservedGeneration >= deployment.Generation &&
		deployment.Status.Replicas == desired &&
		deployment.Status.UpdatedReplicas == desired &&
		deployment.Status.AvailableReplicas == desired
}

// IsStatefulSetReady compares against the desired replicas, like IsDeploymentReady
func IsStatefulSetReady(sts *v1.StatefulSet) bool {
	if sts == nil {
		return false
	}

	desired := desiredReplicas(sts.Spec.Replicas)
	return sts.Status.ObservedGeneration >= sts.Generation &&
		sts.Status.Replicas == desired &&
		sts.Status.UpdatedReplicas == desired &&
		sts.Status.ReadyReplicas == desired
}

func IsDeploymentConfigReady(dc *ocp.DeploymentConfig) bool {
	isAvailable := false
	for _, condition := range dc.Status.Conditions {
		if condition.Type == ocp.DeploymentAvailable &&
			condition.Status == v13.ConditionTrue {
			isAvailable = true
			break
		}
	}

	replicasReady := dc.Spec.Replicas == dc.Status.ReadyReplicas &&
		dc.Status.ReadyReplicas == dc.Status.AvailableReplicas

	return isAvailable && replicasReady
}

func IsJobReady(job *v12.Job) bool {
	if job == nil {
		return false
	}

	for _, condition := range job.Status.Conditions {
		if condition.Type == v12.JobComplete &&
			condition.Status == v13.ConditionTrue {
			return true
		}
	}

	return false
}

func IsDaemonSetReady(ds *v1.DaemonSet) bool {
	return ds != nil &&
		ds.Status.ObservedGeneration >= ds.Generation &&
		ds.Status.DesiredNumberScheduled == ds.Status.NumberReady &&
		ds.Status.DesiredNumberScheduled == ds.Status.NumberAvailable &&
		ds.Status.DesiredNumberScheduled == ds.Status.UpdatedNumberScheduled
}

func GetKey(obj unstructured.Unstructured) string {
	return obj.GetKind()
}

func IsResourceReady(obj unstructured.Unstructured) bool {
	if checkFunc, ok := readyCheckRegistry[GetKey(obj)]; ok {
		return checkFunc(obj)
	}

	return false
}
