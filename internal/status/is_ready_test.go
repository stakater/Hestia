package status

import (
	"testing"

	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func replicas(n int32) *int32 { return &n }

func TestIsDeploymentReady(t *testing.T) {
	tests := []struct {
		name   string
		spec   *int32
		status v1.DeploymentStatus
		want   bool
	}{
		{
			name:   "first status of a new deployment reports 0 of 0 before pods exist",
			spec:   replicas(1),
			status: v1.DeploymentStatus{ObservedGeneration: 1},
			want:   false,
		},
		{
			name:   "pods created but not available",
			spec:   replicas(1),
			status: v1.DeploymentStatus{ObservedGeneration: 1, Replicas: 1, UpdatedReplicas: 1},
			want:   false,
		},
		{
			name:   "old pod still running during a surge",
			spec:   replicas(1),
			status: v1.DeploymentStatus{ObservedGeneration: 1, Replicas: 2, UpdatedReplicas: 1, AvailableReplicas: 1},
			want:   false,
		},
		{
			name:   "spec change not yet observed",
			spec:   replicas(1),
			status: v1.DeploymentStatus{ObservedGeneration: 0, Replicas: 1, UpdatedReplicas: 1, AvailableReplicas: 1},
			want:   false,
		},
		{
			name:   "rolled out",
			spec:   replicas(2),
			status: v1.DeploymentStatus{ObservedGeneration: 1, Replicas: 2, UpdatedReplicas: 2, AvailableReplicas: 2},
			want:   true,
		},
		{
			name:   "unset spec.replicas means one",
			spec:   nil,
			status: v1.DeploymentStatus{ObservedGeneration: 1, Replicas: 1, UpdatedReplicas: 1, AvailableReplicas: 1},
			want:   true,
		},
		{
			name:   "scaled to zero",
			spec:   replicas(0),
			status: v1.DeploymentStatus{ObservedGeneration: 1},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &v1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Generation: 1},
				Spec:       v1.DeploymentSpec{Replicas: tt.spec},
				Status:     tt.status,
			}
			if got := IsDeploymentReady(d); got != tt.want {
				t.Errorf("IsDeploymentReady() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsStatefulSetReady(t *testing.T) {
	tests := []struct {
		name   string
		spec   *int32
		status v1.StatefulSetStatus
		want   bool
	}{
		{
			name:   "first status of a new statefulset reports 0 of 0 before pods exist",
			spec:   replicas(3),
			status: v1.StatefulSetStatus{ObservedGeneration: 1},
			want:   false,
		},
		{
			name:   "only some pods ready",
			spec:   replicas(3),
			status: v1.StatefulSetStatus{ObservedGeneration: 1, Replicas: 3, UpdatedReplicas: 3, ReadyReplicas: 2},
			want:   false,
		},
		{
			name:   "all pods ready",
			spec:   replicas(3),
			status: v1.StatefulSetStatus{ObservedGeneration: 1, Replicas: 3, UpdatedReplicas: 3, ReadyReplicas: 3},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sts := &v1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{Generation: 1},
				Spec:       v1.StatefulSetSpec{Replicas: tt.spec},
				Status:     tt.status,
			}
			if got := IsStatefulSetReady(sts); got != tt.want {
				t.Errorf("IsStatefulSetReady() = %v, want %v", got, tt.want)
			}
		})
	}
}
