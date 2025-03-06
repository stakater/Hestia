package controller

import (
	"context"

	"github.com/example/hestia-operator/api/v1alpha1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RunnersResource struct {
	SelectorResource
	Runners     *v1alpha1.RunnerList
	Ready       bool
	Generations map[string]int64
}

func NewRunnersResource(selector *v12.LabelSelector, ctx context.Context, client client.Client) (*RunnersResource, error) {
	r := &RunnersResource{
		SelectorResource: SelectorResource{
			Selector: selector,
			Context:  ctx,
			Client:   client,
		},
		Ready:       false,
		Generations: map[string]int64{},
	}

	if selector == nil {
		r.Ready = true
		return r, nil
	}

	// Fetch deployments
	r.Runners = &v1alpha1.RunnerList{}
	err := r.List(r.Runners)
	if err != nil {
		return nil, err
	}

	// Check ready state and generate generations
	if len(r.Runners.Items) == 0 && selector != nil {
		r.Ready = false
	} else {
		r.Ready = true
		for _, runner := range r.Runners.Items {
			if runner.Status.Pending {
				r.Ready = false
			}

			r.Generations[runner.Name] = runner.Status.ExecutionGeneration
		}

	}

	return r, nil
}
