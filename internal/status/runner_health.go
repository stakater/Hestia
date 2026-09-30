package status

import (
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stakater/hestia-operator/api/v1alpha1"
	"github.com/stakater/hestia-operator/internal/constants"
)

// maxListedWorkloads caps how many not-ready workloads are named in a condition message
const maxListedWorkloads = 5

type RunnerHealthInput struct {
	// WatchWorkloads is true when the Runner has a workloadSelector; the stable-state clock only runs then
	WatchWorkloads   bool
	Workloads        []v1alpha1.WatchedResource
	JobCondition     *metav1.Condition
	Timeout          time.Duration // 0 disables the timeout
	ProgressingSince *metav1.Time
	Now              time.Time
}

type RunnerHealth struct {
	// Conditions holds the kstatus Ready, Reconciling and Stalled conditions
	Conditions       []metav1.Condition
	ProgressingSince *metav1.Time
	RequeueAfter     time.Duration
}

// EvaluateRunnerHealth derives the kstatus conditions from workload stability and the last job result
func EvaluateRunnerHealth(in RunnerHealthInput) RunnerHealth {
	if in.WatchWorkloads {
		if detail, stable := describeStability(in.Workloads); !stable {
			return progressing(in, detail)
		}
	}

	var h RunnerHealth
	switch {
	case in.JobCondition != nil && in.JobCondition.Reason == constants.SuccessfulRunReason:
		h.Conditions = kstatusConditions(metav1.ConditionTrue, false, false, constants.JobSucceededReason, in.JobCondition.Message)
	case in.JobCondition != nil && in.JobCondition.Reason == constants.FailedRunReason:
		h.Conditions = kstatusConditions(metav1.ConditionFalse, false, true, constants.JobFailedReason, in.JobCondition.Message)
	default:
		h.Conditions = kstatusConditions(metav1.ConditionFalse, true, false, constants.WaitingForJobReason, "waiting for job to complete")
	}

	return h
}

func progressing(in RunnerHealthInput, detail string) RunnerHealth {
	since := in.ProgressingSince
	if since == nil {
		// Truncate to match the second precision the timestamp is stored with
		t := metav1.NewTime(in.Now.Truncate(time.Second))
		since = &t
	}

	h := RunnerHealth{ProgressingSince: since}
	elapsed := in.Now.Sub(since.Time)

	if in.Timeout > 0 && elapsed >= in.Timeout {
		h.Conditions = kstatusConditions(metav1.ConditionFalse, false, true, constants.StableStateTimeoutReason,
			fmt.Sprintf("workloads not stable after %s: %s", in.Timeout, detail))
		return h
	}

	if in.Timeout > 0 {
		h.RequeueAfter = in.Timeout - elapsed
	}
	h.Conditions = kstatusConditions(metav1.ConditionFalse, true, false, constants.ProgressingReason,
		fmt.Sprintf("waiting for workloads to become ready: %s", detail))

	return h
}

// describeStability reports whether at least one workload matched and all are ready, with a message when not
func describeStability(workloads []v1alpha1.WatchedResource) (string, bool) {
	if len(workloads) == 0 {
		return "no workloads match workloadSelector", false
	}

	var notReady []string
	for _, w := range workloads {
		if !w.Ready {
			notReady = append(notReady, fmt.Sprintf("%s %s/%s", w.Kind, w.Namespace, w.Name))
		}
	}

	if len(notReady) == 0 {
		return "", true
	}

	if len(notReady) > maxListedWorkloads {
		return fmt.Sprintf("%s and %d more", strings.Join(notReady[:maxListedWorkloads], ", "), len(notReady)-maxListedWorkloads), false
	}

	return strings.Join(notReady, ", "), false
}

func kstatusConditions(ready metav1.ConditionStatus, reconciling, stalled bool, reason, message string) []metav1.Condition {
	return []metav1.Condition{
		{Type: constants.ReadyType, Status: ready, Reason: reason, Message: message},
		{Type: constants.ReconcilingType, Status: boolStatus(reconciling), Reason: reason, Message: message},
		{Type: constants.StalledType, Status: boolStatus(stalled), Reason: reason, Message: message},
	}
}

func boolStatus(b bool) metav1.ConditionStatus {
	if b {
		return metav1.ConditionTrue
	}

	return metav1.ConditionFalse
}
