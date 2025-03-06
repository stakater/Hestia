package controller

import (
	"context"

	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeploymentsResource struct {
	SelectorResource
	Deployments *v1.DeploymentList
	Ready       bool
	Generations map[string]int64
}

func NewDeploymentsResource(selector *v12.LabelSelector, ctx context.Context,
	client client.Client) (*DeploymentsResource, error) {
	r := &DeploymentsResource{
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
	r.Deployments = &v1.DeploymentList{}
	err := r.List(r.Deployments)
	if err != nil {
		return nil, err
	}

	// Check ready state and generate generations
	if len(r.Deployments.Items) == 0 && selector != nil {
		r.Ready = false
	} else {
		r.Ready = true
		for _, d := range r.Deployments.Items {
			if !isDeploymentReady(&d) {
				r.Ready = false
			}

			r.Generations[d.Name] = d.GetGeneration()
		}
	}

	return r, nil
}

func isDeploymentReady(deployment *v1.Deployment) bool {
	return *deployment.Spec.Replicas > 0 && *deployment.Spec.Replicas == deployment.Status.ReadyReplicas
}
