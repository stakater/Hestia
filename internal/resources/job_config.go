package resources

import (
	"context"
	"strconv"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/stakater/hestia-operator/api/v1alpha1"
	"github.com/stakater/hestia-operator/internal/constants"
)

type JobConfig struct {
	runner   *v1alpha1.Runner
	resource *v1.ConfigMap
	scheme   *runtime.Scheme
}

func NewJobConfig(runner *v1alpha1.Runner, scheme *runtime.Scheme) *JobConfig {
	return &JobConfig{
		runner: runner,
		scheme: scheme,
	}
}

func (r *JobConfig) CreateOrUpdate(ctx context.Context, c client.Client, deployments []unstructured.Unstructured, runners []unstructured.Unstructured) error {
	r.resource = &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.runner.Name,
			Namespace: r.runner.Namespace,
		},
	}

	_, err := controllerutil.CreateOrPatch(ctx, c, r.resource, func() error {
		previous := r.resource.Data

		r.resource.SetLabels(map[string]string{
			constants.RunnerLabel:         strconv.FormatBool(true),
			constants.OwnerLabel:          r.runner.Name,
			constants.OwnerNamespaceLabel: r.runner.Namespace,
		})

		readinessMap := make(map[string]string)
		r.resource.Data = readinessMap
		if r.runner.Spec.WorkloadSelector != nil {
			CreateReadinessMap(readinessMap, deployments...)
			r.resource.Data["workloadMinimum"] = strconv.FormatBool(len(deployments) > 0)
		}

		if r.runner.Spec.RunnerSelector != nil {
			CreateReadinessMap(readinessMap, runners...)
			r.resource.Data["runnersCountMinimum"] = strconv.FormatBool(len(runners) > 0)
		}

		r.resource.Data["generation"] = strconv.FormatInt(r.runner.Generation, 10)
		r.resource.Data["schedule"] = r.runner.Spec.Schedule
		r.resource.Data["deadline"] = strconv.FormatInt(r.runner.Spec.DeadlineSeconds, 10)
		r.resource.Data[RunVersionKey] = nextRunVersion(previous, r.resource.Data)

		return controllerutil.SetControllerReference(r.runner, r.resource, r.scheme)
	})

	return err
}

// RunVersionKey is the job-config entry a job runs once for
const RunVersionKey = "runVersion"

// nextRunVersion bumps the run version only when an entry changed, so rewriting the job config with the same
// content does not rerun the job
func nextRunVersion(previous, current map[string]string) string {
	version, ok := previous[RunVersionKey]
	if !ok {
		return "1"
	}

	if !runChanged(previous, current) {
		return version
	}

	n, err := strconv.ParseUint(version, 10, 64)
	if err != nil {
		return "1"
	}
	return strconv.FormatUint(n+1, 10)
}

func runChanged(previous, current map[string]string) bool {
	for key, value := range current {
		if key == RunVersionKey {
			continue
		}
		if old, ok := previous[key]; !ok || old != value {
			return true
		}
	}

	for key := range previous {
		if _, ok := current[key]; !ok && key != RunVersionKey {
			return true
		}
	}

	return false
}
