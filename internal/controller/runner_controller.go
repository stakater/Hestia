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
	"github.com/example/hestia-operator/api/v1alpha1"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const RunnerLabel = "runner.stakater.com/enable"

// RunnerReconciler reconciles a Runner object
type RunnerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=e2e.stakater.com,resources=runners,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=e2e.stakater.com,resources=runners/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=e2e.stakater.com,resources=runners/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch,resources=jobs/status,verbs=get

func (r *RunnerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.Log.WithName(fmt.Sprintf("[%s]", req.Name))

	// Fetch runner
	runner := &v1alpha1.Runner{}
	err := r.Client.Get(ctx, req.NamespacedName, runner)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		logger.Error(err, fmt.Sprintf("fetching runner: [%s]", req.NamespacedName))
		return ctrl.Result{}, err
	}

	// Handle delete
	if runner.GetDeletionTimestamp() != nil {
		logger.Info("deleted")
		return ctrl.Result{}, nil
	}

	deploymentsHelper, err := NewDeploymentsResource(runner.Spec.DeploymentSelector, ctx, r.Client)
	if err != nil {
		logger.Error(err, "fetching deployments")
		return ctrl.Result{}, err
	}

	runnersHelper, err := NewRunnersResource(runner.Spec.RunnerSelector, ctx, r.Client)
	if err != nil {
		logger.Error(err, "fetching runners")
		return ctrl.Result{}, err
	}

	// fetch existing job
	job := &v1.Job{}
	err = r.Get(ctx, client.ObjectKey{
		Namespace: runner.Namespace,
		Name:      runner.Name,
	}, job)

	if err != nil {
		if errors.IsNotFound(err) {
			if !deploymentsHelper.Ready || !runnersHelper.Ready {
				logger.Info(fmt.Sprintf("is waiting for conditions deployments: %v & runners: %v",
					deploymentsHelper.Ready,
					runnersHelper.Ready))
				return ctrl.Result{}, nil
			}

			job = &v1.Job{
				ObjectMeta: v12.ObjectMeta{
					Namespace: runner.Namespace,
					Name:      runner.Name,
				},
				Spec: v1.JobSpec{
					Template: runner.Spec.Template,
					//TTLSecondsAfterFinished: &[]int32{30 * 60}[0],
					BackoffLimit: &[]int32{1}[0],
				},
			}

			err = controllerutil.SetOwnerReference(runner, job, r.Scheme)
			if err != nil {
				logger.Error(err, fmt.Sprintf("failed to set job owner ref: %s", job.Name))
				return ctrl.Result{}, err
			}

			err = r.Client.Create(ctx, job)
			if err != nil {
				logger.Error(err, fmt.Sprintf("failed to create new job: %s", job.Name))
				return ctrl.Result{}, err
			}

			runner.Status.ResourceGeneration = runner.GetGeneration()
			runner.Status.RunnerGeneration = runnersHelper.Generations
			runner.Status.DeploymentsGeneration = deploymentsHelper.Generations
			logger.Info(fmt.Sprintf("created new job: %s", job.Name))
		} else {
			logger.Info(fmt.Sprintf("failed fetching job [%s]", job.Name))
			return ctrl.Result{}, err
		}
	}

	// delete no longer valid job
	if runner.Status.ResourceGeneration != runner.GetGeneration() ||
		runner.Status.DeploymentsGeneration != nil && !reflect.DeepEqual(runner.Status.DeploymentsGeneration,
			deploymentsHelper.Generations) ||
		runner.Status.RunnerGeneration != nil && !reflect.DeepEqual(runner.Status.RunnerGeneration, runnersHelper.Generations) {

		logger.Info(fmt.Sprintf("deleted expired job: %s", job.Name))

		options := &client.DeleteOptions{
			PropagationPolicy: new(v12.DeletionPropagation),
		}
		*options.PropagationPolicy = v12.DeletePropagationBackground
		return ctrl.Result{Requeue: true}, r.Delete(ctx, job, options)
	} else {
		// update job status
		status := getJobStatus(job)
		runner.Status.Pending = status[v1.JobComplete] != v12.ConditionTrue
		runner.Status.Success = status[v1.JobComplete] == v12.ConditionTrue
		runner.Status.Failed = status[v1.JobFailed] == v12.ConditionTrue

		// Update runner status
		err = r.Client.Status().Update(ctx, runner)
		if err != nil {
			logger.Error(err, "failed to update status")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RunnerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Runner{}).
		Watches(&v1.Job{}, handler.EnqueueRequestForOwner(mgr.GetScheme(), mgr.GetRESTMapper(), &v1alpha1.Runner{}),
			builder.WithPredicates(predicate.Funcs{
				CreateFunc: func(createEvent event.CreateEvent) bool {
					return false
				},
				DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
					return true
				},
				GenericFunc: func(genericEvent event.GenericEvent) bool {
					return false
				},
				UpdateFunc: func(updateEvent event.UpdateEvent) bool {
					old := getJobStatus(updateEvent.ObjectOld.(*v1.Job))
					recent := getJobStatus(updateEvent.ObjectNew.(*v1.Job))

					return old[v1.JobComplete] != v12.ConditionTrue && recent[v1.JobComplete] == v12.ConditionTrue
				},
			})).
		Watches(&v1alpha1.Runner{}, handler.EnqueueRequestsFromMapFunc(enqueueForWatchingRunners(r)),
			builder.WithPredicates(predicate.Funcs{
				CreateFunc: func(createEvent event.CreateEvent) bool {
					return false
				},
				DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
					return false
				},
				GenericFunc: func(genericEvent event.GenericEvent) bool {
					return false
				},
				UpdateFunc: func(updateEvent event.UpdateEvent) bool {
					old := updateEvent.ObjectOld.(*v1alpha1.Runner)
					recent := updateEvent.ObjectNew.(*v1alpha1.Runner)

					return !old.Status.Success && recent.Status.Success
				},
			})).
		Watches(&apps.Deployment{}, handler.EnqueueRequestsFromMapFunc(enqueueForDeployment(r)),
			builder.WithPredicates(predicate.Funcs{
				CreateFunc: func(event event.CreateEvent) bool {
					return false
				},
				DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
					return true
				},
				GenericFunc: func(genericEvent event.GenericEvent) bool {
					return false
				},
				UpdateFunc: func(updateEvent event.UpdateEvent) bool {
					old := updateEvent.ObjectOld.(*apps.Deployment)
					recent := updateEvent.ObjectNew.(*apps.Deployment)

					label := map[string]string{}
					label[RunnerLabel] = "true"
					if labels.Conflicts(label, recent.Labels) {
						return false
					}

					return isDeploymentReady(old) != isDeploymentReady(recent)
				},
			})).
		Complete(r)
}

type SpecialRequest struct {
	reconcile.Request
	Custom string `json:"custom,omitempty"`
}

func getJobStatus(job *v1.Job) map[v1.JobConditionType]v12.ConditionStatus {
	status := make(map[v1.JobConditionType]v12.ConditionStatus)

	for _, condition := range job.Status.Conditions {
		status[condition.Type] = v12.ConditionStatus(condition.Status)
	}

	return status
}

func enqueueForWatchingRunners(r *RunnerReconciler) handler.MapFunc {
	return func(ctx context.Context, object client.Object) []reconcile.Request {
		runner := object.(*v1alpha1.Runner)
		var requests []reconcile.Request

		runners := &v1alpha1.RunnerList{}
		err := r.Client.List(ctx, runners)
		if err != nil {
			return requests
		}

		for _, r := range runners.Items {
			runnerSelector, err := v12.LabelSelectorAsMap(r.Spec.RunnerSelector)
			if err != nil || labels.Conflicts(runnerSelector, runner.Labels) {
				continue
			}

			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: r.Namespace,
					Name:      r.Name,
				},
			})
		}

		return requests
	}
}

func enqueueForDeployment(r *RunnerReconciler) handler.MapFunc {
	return func(ctx context.Context, object client.Object) []reconcile.Request {
		deployment := object.(*apps.Deployment)
		requests := make([]reconcile.Request, 0)
		runners := &v1alpha1.RunnerList{}
		_ = r.Client.List(ctx, runners)

		for _, r := range runners.Items {
			selector, err := v12.LabelSelectorAsMap(r.Spec.DeploymentSelector)
			if err != nil || labels.Conflicts(selector, deployment.Labels) {
				continue
			}

			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: r.Namespace,
					Name:      r.Name,
				},
			})
		}

		return requests
	}
}
