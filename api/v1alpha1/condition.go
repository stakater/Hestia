package v1alpha1

import (
	"github.com/redhat-cop/operator-utils/pkg/util/apis"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Conditions struct {
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

func getOpt(idx int, opts ...string) string {
	if len(opts) < idx+1 {
		return ""
	}

	return opts[idx]
}

func (s *Conditions) UpdateCondition(conditionType string, conditionStatus metav1.ConditionStatus, opts ...string) {
	s.Conditions = apis.AddOrReplaceCondition(metav1.Condition{
		Type:               conditionType,
		Status:             conditionStatus,
		LastTransitionTime: metav1.Now(),
		Reason:             getOpt(0, opts...),
		Message:            getOpt(1, opts...),
	}, s.Conditions)
}

func (s *Conditions) SetReady(conditionStatus metav1.ConditionStatus, msg ...string) {
	s.UpdateCondition(apis.ReconcileSuccess, conditionStatus, apis.ReconcileSuccessReason, getOpt(0, msg...))
}

func (s *Conditions) IsReady() bool {
	c, exists := apis.GetCondition(apis.ReconcileSuccess, s.Conditions)
	if !exists {
		return false
	}

	return c.Status == metav1.ConditionTrue
}

func (s *Conditions) ConditionMsg(ct string) string {
	c, exists := apis.GetCondition(ct, s.Conditions)
	if !exists {
		return ""
	}

	return c.Message
}
