package resources

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func workload(kind string, generation int64) unstructured.Unstructured {
	obj := unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "apps/v1", "kind": kind}}
	obj.SetName("gw")
	obj.SetNamespace("ns")
	obj.SetGeneration(generation)
	return obj
}

func TestReadinessMapStoresGenerationOfRolloutKinds(t *testing.T) {
	for _, kind := range []string{"Deployment", "StatefulSet", "DaemonSet", "DeploymentConfig"} {
		obj := workload(kind, 4)
		if got := CreateReadinessMap(map[string]string{}, obj)[getKey(obj)+".generation"]; got != "4" {
			t.Errorf("%s generation = %q, want \"4\"", kind, got)
		}
	}
}

func TestReadinessMapSkipsGenerationOfOtherKinds(t *testing.T) {
	for _, kind := range []string{"Runner", "Job"} {
		obj := workload(kind, 4)
		if _, ok := CreateReadinessMap(map[string]string{}, obj)[getKey(obj)+".generation"]; ok {
			t.Errorf("%s got a generation entry; only rolling workloads should", kind)
		}
	}
}
