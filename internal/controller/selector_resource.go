package controller

import (
	"context"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SelectorResource struct {
	Selector *v1.LabelSelector
	Context  context.Context
	Client   client.Client
}

func (sr *SelectorResource) List(object client.ObjectList) error {
	selector, err := v1.LabelSelectorAsSelector(sr.Selector)
	if err != nil {
		return err
	}

	return sr.Client.List(sr.Context, object, &client.ListOptions{
		LabelSelector: selector,
	})
}
