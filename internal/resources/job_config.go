package resources

import (
	"context"
	"github.com/stakater/hestia-operator/api/v1alpha1"
	"github.com/stakater/hestia-operator/internal/constants"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strconv"
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

func (r *JobConfig) CreateOrUpdate(ctx context.Context, c client.Client, objects ...unstructured.Unstructured) error {
	r.resource = &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.runner.Name,
			Namespace: r.runner.Namespace,
		},
	}

	_, err := controllerutil.CreateOrPatch(ctx, c, r.resource, func() error {
		r.resource.SetLabels(map[string]string{
			constants.RunnerLabel:         strconv.FormatBool(true),
			constants.OwnerLabel:          r.runner.Name,
			constants.OwnerNamespaceLabel: r.runner.Namespace,
		})

		r.resource.Data = CreateReadinessMap(objects...)
		r.resource.Data["generation"] = strconv.FormatInt(r.runner.Generation, 10)
		r.resource.Data["schedule"] = r.runner.Spec.Schedule
		r.resource.Data["deadline"] = strconv.FormatInt(r.runner.Spec.DeadlineSeconds, 10)

		return controllerutil.SetControllerReference(r.runner, r.resource, r.scheme)
	})

	return err
}
