package status

import (
	ocp "github.com/openshift/api/apps/v1"
	"github.com/redhat-cop/operator-utils/pkg/util/apis"
	"github.com/stakater/hestia-operator/api/v1alpha1"
	"github.com/stakater/hestia-operator/internal/constants"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/batch/v1"
	v13 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
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

func IsDeploymentReady(deployment *v1.Deployment) bool {
	return deployment != nil &&
		deployment.Status.ObservedGeneration >= deployment.Generation &&
		deployment.Status.UpdatedReplicas == deployment.Status.Replicas &&
		deployment.Status.AvailableReplicas == deployment.Status.Replicas
}

func IsStatefulSetReady(sts *v1.StatefulSet) bool {
	return sts != nil &&
		sts.Status.ObservedGeneration >= sts.Generation &&
		sts.Status.UpdatedReplicas == sts.Status.Replicas &&
		sts.Status.ReadyReplicas == sts.Status.Replicas
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

func GetKey(obj unstructured.Unstructured) string {
	return obj.GetKind()
}

func IsResourceReady(obj unstructured.Unstructured) bool {
	if checkFunc, ok := readyCheckRegistry[GetKey(obj)]; ok {
		return checkFunc(obj)
	}

	return false
}
