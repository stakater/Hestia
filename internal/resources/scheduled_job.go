package resources

import (
	"context"
	"github.com/stakater/hestia-operator/api/v1alpha1"
	v1 "k8s.io/api/batch/v1"
	v12 "k8s.io/api/core/v1"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strconv"
)

type ScheduledJob struct {
	runner   *v1alpha1.Runner
	resource *v1.CronJob
	scheme   *runtime.Scheme
	JobStatus
}

func NewScheduledJobResource(runner *v1alpha1.Runner, config *v12.ConfigMap, scheme *runtime.Scheme) *ScheduledJob {
	return &ScheduledJob{
		runner:    runner,
		scheme:    scheme,
		JobStatus: JobStatus{config: config},
	}
}

func (r *ScheduledJob) CreateOrUpdate(ctx context.Context, c client.Client) error {
	r.resource = &v1.CronJob{
		ObjectMeta: v13.ObjectMeta{
			Name:      r.runner.Name,
			Namespace: r.runner.Namespace,
		},
	}

	_, err := controllerutil.CreateOrPatch(ctx, c, r.resource, func() error {
		r.resource.SetLabels(r.defaultJobLabels())

		r.resource.Spec = v1.CronJobSpec{
			Schedule:                   r.runner.Spec.Schedule,
			Suspend:                    &[]bool{!r.isReady()}[0],
			ConcurrencyPolicy:          v1.ForbidConcurrent,
			FailedJobsHistoryLimit:     &[]int32{1}[0],
			SuccessfulJobsHistoryLimit: &[]int32{1}[0],

			JobTemplate: v1.JobTemplateSpec{
				ObjectMeta: v13.ObjectMeta{
					Labels: r.defaultJobLabels(),
				},
				Spec: v1.JobSpec{
					Template:                r.runner.Spec.Template,
					TTLSecondsAfterFinished: &[]int32{30 * 60}[0],
					BackoffLimit:            &[]int32{1}[0],
					ActiveDeadlineSeconds:   &[]int64{r.runner.Spec.DeadlineSeconds}[0],
				},
			},
		}

		return controllerutil.SetControllerReference(r.config, r.resource, r.scheme)
	})

	return err
}

func (r *ScheduledJob) RemoveJob(ctx context.Context, c client.Client) error {
	err := c.Delete(ctx, &v1.CronJob{
		ObjectMeta: v13.ObjectMeta{
			Name:      r.runner.Name,
			Namespace: r.runner.Namespace,
		},
	})

	return client.IgnoreNotFound(err)
}

func (r *ScheduledJob) isReady() bool {
	for _, v := range r.config.Data {
		if v == strconv.FormatBool(false) {
			return false
		}
	}

	return true
}
