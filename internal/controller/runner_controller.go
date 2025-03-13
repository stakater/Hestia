package controller

import (
	"context"
	"fmt"
	"time"

	app "github.com/example/hestia-operator/api/v1alpha1"
	"github.com/example/hestia-operator/internal/informers"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type RunnerReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	logger          logr.Logger
	events          chan event.TypedGenericEvent[informers.DynamicResourceEvent]
	dynamicInformer *informers.DynamicInformer
}

//+kubebuilder:rbac:groups=e2e.stakater.com,resources=runners,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=e2e.stakater.com,resources=runners/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=e2e.stakater.com,resources=runners/finalizers,verbs=update
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="*",resources="*/status",verbs=get;list;watch

func (r *RunnerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.Log.WithName(fmt.Sprintf("[%s]", req.Name))
	r.logger.Info("reconcile started...")

	// Fetch runner
	runner := &app.Runner{}
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
		r.logger.Info(fmt.Sprintf("runner %s deleted", runner.Name))
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RunnerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	dynamicClient, err := dynamic.NewForConfig(mgr.GetConfig())
	if err != nil {
		return err
	}

	r.dynamicInformer = informers.NewDynamicInformer(dynamicClient, schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "statefulsets",
	})

	r.events = make(chan event.TypedGenericEvent[informers.DynamicResourceEvent])
	r.dynamicInformer.Events = r.events
	err = mgr.Add(r.dynamicInformer)

	if err != nil {
		return err
	}

	err = mgr.Add(manager.RunnableFunc(func(ctx context.Context) error {
		rand.Seed(time.Now().UnixNano())
		randomDuration := time.Duration(rand.Intn(10)+1) * time.Second

		r.logger.Info(fmt.Sprintf("simulate shutdown runnable in %v seconds...\n", randomDuration))

		select {
		case <-time.After(randomDuration):
			r.logger.Info("RandomStopperRunnable: Stopping the informer now!")
			r.dynamicInformer.Stop()
		case <-ctx.Done():
			r.logger.Info("RandomStopperRunnable: Manager stopped, exiting...")
		}

		r.logger.Info("RandomStopperRunnable: done")
		return nil
	}))

	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named("Runner").
		For(&app.Runner{}).
		WatchesRawSource(source.TypedChannel(
			r.events,
			handler.TypedEnqueueRequestsFromMapFunc(r.HandleTyped),
		)).
		Complete(r)
}

func (r *RunnerReconciler) HandleChanges(ctx context.Context, object client.Object) []reconcile.Request {
	r.logger.Info("handle changes for: " + object.GetName())

	var requests []reconcile.Request
	runners := &app.RunnerList{}
	if err := r.List(ctx, runners); err != nil {
		return requests
	}

	for _, runner := range runners.Items {
		selector := labels.SelectorFromSet(runner.Spec.DeploymentSelector.MatchLabels)

		if !selector.Matches(labels.Set(object.GetLabels())) {
			continue
		}

		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: runner.Namespace,
				Name:      runner.Name,
			},
		})
	}

	return requests
}

func (r *RunnerReconciler) HandleTyped(ctx context.Context, event informers.DynamicResourceEvent) []reconcile.Request {
	var requests []reconcile.Request
	runners := &app.RunnerList{}
	if err := r.List(ctx, runners); err != nil {
		return requests
	}

	for _, runner := range runners.Items {
		selector := labels.SelectorFromSet(runner.Spec.DeploymentSelector.MatchLabels)

		if !selector.Matches(labels.Set(event.Object.GetLabels())) {
			continue
		}

		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: runner.Namespace,
				Name:      runner.Name,
			},
		})
	}

	return requests
}
