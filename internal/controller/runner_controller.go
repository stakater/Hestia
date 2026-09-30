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

package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	v13 "github.com/openshift/api/apps/v1"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/stakater/hestia-operator/api/v1alpha1"
	"github.com/stakater/hestia-operator/internal/constants"
	"github.com/stakater/hestia-operator/internal/resources"
	"github.com/stakater/hestia-operator/internal/status"
)

// RunnerReconciler reconciles a Runner object
type RunnerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
}

//+kubebuilder:rbac:groups=e2e.stakater.com,resources=runners,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=e2e.stakater.com,resources=runners/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=e2e.stakater.com,resources=runners/finalizers,verbs=update
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch
//+kubebuilder:rbac:groups=apps.openshift.io,resources=deploymentconfigs,verbs=get;list;watch
//+kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch

func (r *RunnerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.Log.WithName(fmt.Sprintf("[Runner] %s", req.NamespacedName))
	r.logger.Info("Reconciling...")

	// Fetch runner
	runner := &v1alpha1.Runner{}
	err := r.Client.Get(ctx, req.NamespacedName, runner)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		r.logger.Error(err, fmt.Sprintf("fetching runner: [%s]", req.NamespacedName))
		return ctrl.Result{}, err
	}

	// Handle delete
	if runner.GetDeletionTimestamp() != nil {
		r.logger.Info("deleted")
		return ctrl.Result{}, nil
	}

	// Get matchedDeployments selected deployments
	deploymentResource := resources.NewResourceCollector([]schema.GroupVersionKind{
		{Group: "apps", Version: "v1", Kind: "Deployment"},
		{Group: "apps", Version: "v1", Kind: "StatefulSet"},
		{Group: "apps", Version: "v1", Kind: "DaemonSet"},
		{Group: "batch", Version: "v1", Kind: "Job"},
		{Group: "apps.openshift.io", Version: "v1", Kind: "DeploymentConfig"},
	})

	matchedDeployments, err := deploymentResource.GetAll(ctx, r.Client, runner.Spec.WorkloadSelector)
	if err != nil {
		return r.HandleError(ctx, runner, err, "error fetching selected workloads")
	}

	// Get matchedDeployments selected runners
	runnerResource := resources.NewResourceCollector([]schema.GroupVersionKind{
		{Group: "e2e.stakater.com", Version: "v1alpha1", Kind: "Runner"},
	})
	matchedRunners, err := runnerResource.GetAll(ctx, r.Client, runner.Spec.RunnerSelector)
	if err != nil {
		return r.HandleError(ctx, runner, err, "error fetching selected runners")
	}

	// Setup job config
	jobConfig := resources.NewJobConfig(runner, r.Scheme)
	err = jobConfig.CreateOrUpdate(ctx, r.Client, matchedDeployments, matchedRunners)
	if err != nil {
		return r.HandleError(ctx, runner, err, "error setting up job config")
	}

	health := status.EvaluateRunnerHealth(status.RunnerHealthInput{
		WatchWorkloads:   runner.Spec.WorkloadSelector != nil,
		Workloads:        resources.CreateReadinessStatus(matchedDeployments...),
		JobCondition:     apimeta.FindStatusCondition(runner.Status.Conditions.Conditions, constants.JobStatusType),
		Timeout:          stableStateTimeout(runner),
		ProgressingSince: runner.Status.ProgressingSince,
		Now:              time.Now(),
	})

	matchedResources := append(matchedRunners, matchedDeployments...)
	return r.HandleSuccess(ctx, runner, resources.CreateReadinessStatus(matchedResources...), health)
}

func stableStateTimeout(runner *v1alpha1.Runner) time.Duration {
	if runner.Spec.StableStateTimeout == nil {
		return constants.DefaultStableStateTimeout
	}

	return runner.Spec.StableStateTimeout.Duration
}

var readyPredicateFn = predicate.Funcs{
	CreateFunc: func(e event.CreateEvent) bool {
		return false
	},
	DeleteFunc: func(e event.DeleteEvent) bool {
		return true
	},
	UpdateFunc: func(e event.UpdateEvent) bool {
		newReady, ok := isWorkloadReady(e.ObjectNew)
		if !ok {
			return false
		}
		oldReady, _ := isWorkloadReady(e.ObjectOld)

		// Ready -> not-ready transitions are passed too so the stable-state clock starts
		return newReady || oldReady != newReady
	},
}

// isWorkloadReady returns the readiness of a watched workload, and false for ok when the kind is not watched
func isWorkloadReady(obj client.Object) (ready bool, ok bool) {
	switch o := obj.(type) {
	case *v1.Deployment:
		return status.IsDeploymentReady(o), true
	case *v1.StatefulSet:
		return status.IsStatefulSetReady(o), true
	case *v1.DaemonSet:
		return status.IsDaemonSetReady(o), true
	case *v13.DeploymentConfig:
		return status.IsDeploymentConfigReady(o), true
	default:
		return false, false
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *RunnerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Runner{}).
		Watches(&v1.Deployment{}, handler.EnqueueRequestsFromMapFunc(r.labelMatchingHandler), builder.WithPredicates(readyPredicateFn)).
		Watches(&v1.StatefulSet{}, handler.EnqueueRequestsFromMapFunc(r.labelMatchingHandler), builder.WithPredicates(readyPredicateFn)).
		Watches(&v1.DaemonSet{}, handler.EnqueueRequestsFromMapFunc(r.labelMatchingHandler), builder.WithPredicates(readyPredicateFn)).
		Watches(&v13.DeploymentConfig{}, handler.EnqueueRequestsFromMapFunc(r.labelMatchingHandler), builder.WithPredicates(readyPredicateFn)).
		Watches(&v1alpha1.Runner{}, handler.EnqueueRequestsFromMapFunc(r.runnerReadyHandler)).
		Complete(r)
}

func (r *RunnerReconciler) HandleError(ctx context.Context, cr *v1alpha1.Runner, err error, msg string) (ctrl.Result, error) {
	if err != nil {
		cr.Status.Conditions.SetReady(v12.ConditionFalse, fmt.Sprintf("%s. %s", msg, err.Error()))
	} else {
		cr.Status.Conditions.SetReady(v12.ConditionFalse, msg)
	}

	err = r.Status().Update(ctx, cr)
	if err != nil {
		return ctrl.Result{Requeue: true}, nil
	}
	return ctrl.Result{}, nil
}

func (r *RunnerReconciler) HandleSuccess(ctx context.Context, cr *v1alpha1.Runner, readinessStatus []v1alpha1.WatchedResource, health status.RunnerHealth) (ctrl.Result, error) {
	cr.Status.Conditions.SetReady(v12.ConditionTrue)
	cr.Status.WatchedResources = readinessStatus
	cr.Status.ObservedGeneration = cr.Generation
	cr.Status.ProgressingSince = health.ProgressingSince

	// SetStatusCondition keeps LastTransitionTime unless the status changes, as kstatus expects
	for _, c := range health.Conditions {
		c.ObservedGeneration = cr.Generation
		apimeta.SetStatusCondition(&cr.Status.Conditions.Conditions, c)
	}

	err := r.Status().Update(ctx, cr)
	if err != nil {
		return ctrl.Result{Requeue: true}, nil
	}
	return ctrl.Result{RequeueAfter: health.RequeueAfter}, nil
}

// labelMatchingHandler reconcile Runners with matching deployment selector label
func (r *RunnerReconciler) labelMatchingHandler(ctx context.Context, object client.Object) []reconcile.Request {
	var request []reconcile.Request
	runners := &v1alpha1.RunnerList{}
	err := r.Client.List(ctx, runners)
	if err != nil {
		r.logger.Error(err, "failed to list runners")
		return request
	}

	for _, runner := range runners.Items {
		selector, err := v12.LabelSelectorAsSelector(runner.Spec.WorkloadSelector)
		if err != nil {
			r.logger.Error(err, "failed to create label selector")
			return request
		}

		if selector.Matches(labels.Set(object.GetLabels())) {
			request = append(request, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: runner.GetNamespace(),
					Name:      runner.GetName(),
				},
			})
		}
	}

	return request
}

func (r *RunnerReconciler) runnerReadyHandler(ctx context.Context, object client.Object) []reconcile.Request {
	var request []reconcile.Request
	runnerObj, ok := object.(*v1alpha1.Runner)
	if !ok || !status.IsRunnerReady(runnerObj) {
		return request
	}

	runners := &v1alpha1.RunnerList{}
	err := r.Client.List(ctx, runners)
	if err != nil {
		return request
	}

	for _, runner := range runners.Items {
		selector, err := v12.LabelSelectorAsSelector(runner.Spec.RunnerSelector)
		if err != nil {
			continue
		}

		if selector.Matches(labels.Set(runnerObj.GetLabels())) {
			request = append(request, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: runner.GetNamespace(),
					Name:      runner.GetName(),
				},
			})
		}
	}

	return request
}
