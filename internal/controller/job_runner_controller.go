package controller

import (
	"context"
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
)

// JobRunnerReconciler reconciles ConfigMaps and manages Jobs
type JobRunnerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=e2e.stakater.com,resources=runners,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop
func (r *JobRunnerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.WithName(req.NamespacedName.String()).Info("Reconciling...")

	// Check if this is a ConfigMap reconciliation
	var configMap corev1.ConfigMap
	err := r.Get(ctx, req.NamespacedName, &configMap)

	// If neither was found, it was likely deleted
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

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

		lastStatus = jobResource.LastStatus(ctx, r.Client)
	}

	return r.PatchReadyStatus(ctx, runner, lastStatus)
}

func (r *JobRunnerReconciler) PatchReadyStatus(ctx context.Context, cr v1alpha1.Runner, condition v12.Condition) (ctrl.Result, error) {
	if condition.Type == resources.JobStatusType {
		if condition.Reason == resources.SuccessfulRunReason {
			cr.Status.LastSuccessfulRun = v12.Now()
		} else if condition.Reason == resources.FailedRunReason {
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

func (r *JobRunnerReconciler) HandleSuccess(ctx context.Context, cr *v1alpha1.Runner) (ctrl.Result, error) {
	original := cr.DeepCopy()
	cr.Status.Conditions.SetReady(v12.ConditionTrue)
	return ctrl.Result{}, r.Status().Patch(ctx, original, client.MergeFrom(cr))
}

// SetupWithManager sets up the controller with the Manager.
func (r *JobRunnerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&batchv1.Job{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&batchv1.CronJob{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}
