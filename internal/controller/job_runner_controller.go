package controller

import (
	"context"
	"fmt"
	"github.com/stakater/hestia-operator/api/v1alpha1"
	"github.com/stakater/hestia-operator/internal/constants"
	"github.com/stakater/hestia-operator/internal/resources"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"strconv"
)

// JobRunnerReconciler reconciles ConfigMaps and manages Jobs
type JobRunnerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=e2e.stakater.com,resources=runners,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch,resources=jobs/finalizers,verbs=update
//+kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop
func (r *JobRunnerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.Log.WithName(fmt.Sprintf("[Jobrunner] %s", req.NamespacedName))
	logger.Info("Reconciling...")

	// Fetch job config
	var configMap corev1.ConfigMap
	err := r.Get(ctx, req.NamespacedName, &configMap)

	// If not found return
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	// Ensure the configMap registers is ready for all boolean keys
	for _, v := range configMap.Data {
		if v == strconv.FormatBool(false) {
			return ctrl.Result{}, nil
		}
	}

	// Fetch related runner
	var runner v1alpha1.Runner
	err = r.Get(ctx, client.ObjectKey{
		Namespace: configMap.Labels[constants.OwnerNamespaceLabel],
		Name:      configMap.Labels[constants.OwnerLabel],
	}, &runner)

	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}

	jobResource := resources.NewJobResource(&runner, &configMap, r.Scheme)
	cronJobResource := resources.NewScheduledJobResource(&runner, &configMap, r.Scheme)
	var lastStatus v12.Condition

	// Handle a non-scheduled job
	if runner.Spec.Schedule == "" {
		err = cronJobResource.RemoveJob(ctx, r.Client)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = jobResource.SyncJob(ctx, r.Client)
		if err != nil {
			return ctrl.Result{}, err
		}

		lastStatus = jobResource.LastStatus(ctx, r.Client)
	} else {
		// Handle a scheduled job
		err = jobResource.RemoveAll(ctx, r.Client, func(obj batchv1.Job) bool {
			for _, reference := range obj.OwnerReferences {
				if reference.Kind == "CronJob" {
					return false
				}
			}

			return true
		})
		if err != nil {
			return ctrl.Result{}, err
		}

		err = cronJobResource.CreateOrUpdate(ctx, r.Client)
		if err != nil {
			return ctrl.Result{}, err
		}

		lastStatus = cronJobResource.LastStatus(ctx, r.Client)
	}

	// Update job-run status
	return r.PatchRunnerJobStatus(ctx, runner, lastStatus)
}

// PatchRunnerJobStatus updates runner with latest job status based on executed final conditions
func (r *JobRunnerReconciler) PatchRunnerJobStatus(ctx context.Context, cr v1alpha1.Runner, condition v12.Condition) (ctrl.Result, error) {
	if condition.Type == constants.JobStatusType {
		if condition.Reason == constants.SuccessfulRunReason {
			cr.Status.LastSuccessfulRun = v12.Now()
		} else if condition.Reason == constants.FailedRunReason {
			cr.Status.LastFailedRun = v12.Now()
		}
	}

	cr.Status.UpdateCondition(condition.Type, condition.Status, condition.Reason, condition.Message)

	err := r.Status().Update(ctx, &cr)
	if err != nil {
		return ctrl.Result{Requeue: true}, nil
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *JobRunnerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}, builder.WithPredicates(predicate.ResourceVersionChangedPredicate{})).
		Owns(&batchv1.Job{}).
		Complete(r)
}
