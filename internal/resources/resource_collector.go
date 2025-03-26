package resources

import (
	"context"
	"fmt"
	"github.com/stakater/hestia-operator/api/v1alpha1"
	"github.com/stakater/hestia-operator/internal/status"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"regexp"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"strings"
)

type ResourceCollector struct {
	gvk []schema.GroupVersionKind
}

func NewResourceCollector(kinds []schema.GroupVersionKind) *ResourceCollector {
	return &ResourceCollector{
		gvk: kinds,
	}
}

func (c *ResourceCollector) GetAll(ctx context.Context, kc client.Client, labelSelector *v1.LabelSelector) ([]unstructured.Unstructured, error) {
	var result []unstructured.Unstructured
	option := client.MatchingLabelsSelector{}

	if labelSelector == nil {
		return result, nil
	}

	selector, err := v1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		option.Selector = labels.Nothing()
	} else {
		option.Selector = selector
	}

	for _, gvk := range c.gvk {
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   gvk.Group,
			Version: gvk.Version,
			Kind:    gvk.Kind + "List",
		})

		if err = kc.List(ctx, list, option); err != nil {
			return nil, err
		}

		result = append(result, list.Items...)
	}

	return result, nil
}

func CreateReadinessMap(readinessMap map[string]string, objects ...unstructured.Unstructured) map[string]string {
	for _, obj := range objects {
		readinessMap[getKey(obj)] = strconv.FormatBool(status.IsResourceReady(obj))
	}

	return readinessMap
}

func CreateReadinessStatus(objects ...unstructured.Unstructured) []v1alpha1.WatchedResource {
	var readinessStatus []v1alpha1.WatchedResource
	for _, obj := range objects {
		readinessStatus = append(readinessStatus, v1alpha1.WatchedResource{
			Namespace: obj.GetNamespace(),
			Name:      obj.GetName(),
			Kind:      obj.GetKind(),
			Ready:     status.IsResourceReady(obj),
		})
	}

	return readinessStatus
}

func normalizeMapKey(s string) string {
	s = strings.ToLower(s)
	reg := regexp.MustCompile(`[^a-z0-9_\-.]`)
	s = reg.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")

	if s == "" {
		return "empty_key"
	}

	return s
}

func getKey(obj unstructured.Unstructured) string {
	return normalizeMapKey(fmt.Sprintf("%s-%s-%s-%s", obj.GroupVersionKind().GroupVersion(), obj.GetKind(), obj.GetNamespace(), obj.GetName()))
}
