package resources

import (
	"context"
	"fmt"

	"github.com/stakater/hestia-operator/api/v1alpha1"
	"github.com/stakater/hestia-operator/internal/constants"
	v1 "k8s.io/api/batch/v1"
	v12 "k8s.io/api/core/v1"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type JobResource struct {
	runner   *v1alpha1.Runner
	resource *v1.Job
	scheme   *runtime.Scheme
	JobStatus
}

func NewJobResource(runner *v1alpha1.Runner, config *v12.ConfigMap, scheme *runtime.Scheme) *JobResource {
	return &JobResource{
		runner: runner,
		scheme: scheme,
		JobStatus: JobStatus{
			config: config,
		},
	}
}

func (r *JobResource) removeOldJobs(ctx context.Context, c client.Client) (*v1.Job, error) {
	var existing *v1.Job
	err := r.RemoveAll(ctx, c, func(job v1.Job) bool {
		if job.Labels[constants.VersionLabel] == r.config.ResourceVersion {
			existing = &job
			return false
		}

		return true
	})

	if err != nil {
		return nil, err
	}

	return existing, nil
}

func (r *JobResource) SyncJob(ctx context.Context, c client.Client) error {
	existing, err := r.removeOldJobs(ctx, c)
	if err != nil {
		return err
	}

	if existing != nil {
		r.resource = existing
		return nil
	}

	r.resource = &v1.Job{
		ObjectMeta: v13.ObjectMeta{
			Namespace:    r.runner.Namespace,
			GenerateName: fmt.Sprintf("%s-", r.runner.Name),
			Labels:       r.defaultJobLabels(),
		},
		Spec: v1.JobSpec{
			Template:                r.runner.Spec.Template,
			TTLSecondsAfterFinished: &[]int32{30 * 60}[0],
			BackoffLimit:            &[]int32{1}[0],
		},
	}

	if r.runner.Spec.DeadlineSeconds != 0 {
		r.resource.Spec.ActiveDeadlineSeconds = &r.runner.Spec.DeadlineSeconds
	}

	err = controllerutil.SetControllerReference(r.config, r.resource, r.scheme)
	if err != nil {
		return err
	}

	return c.Create(ctx, r.resource)
}

func (r *JobResource) RemoveAll(ctx context.Context, c client.Client, shouldDeleteFnc func(obj v1.Job) bool) error {
	jobList := &v1.JobList{}
	selector := labels.SelectorFromSet(map[string]string{
		constants.OwnerLabel:          r.runner.Name,
		constants.OwnerNamespaceLabel: r.runner.Namespace,
	})
	err := c.List(ctx, jobList, &client.ListOptions{
		LabelSelector: selector,
	})

	if err != nil {
		return err
	}

	for _, job := range jobList.Items {
		if !shouldDeleteFnc(job) {
			continue
		}

		foregroundDelete := v13.DeletePropagationForeground
		err = c.Delete(ctx, &job, &client.DeleteOptions{
			PropagationPolicy: &foregroundDelete,
		})

		if client.IgnoreNotFound(err) != nil {
			return err
		}
	}

	return nil
}
