package status

import (
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stakater/hestia-operator/api/v1alpha1"
	"github.com/stakater/hestia-operator/internal/constants"
)

func TestEvaluateRunnerHealth(t *testing.T) {
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	ago := func(d time.Duration) *metav1.Time { t := metav1.NewTime(now.Add(-d)); return &t }
	ready := []v1alpha1.WatchedResource{{Kind: "Deployment", Namespace: "ns", Name: "a", Ready: true}}
	notReady := []v1alpha1.WatchedResource{
		{Kind: "Deployment", Namespace: "ns", Name: "a", Ready: true},
		{Kind: "StatefulSet", Namespace: "ns", Name: "b", Ready: false},
	}
	job := func(reason string) *metav1.Condition {
		return &metav1.Condition{Type: constants.JobStatusType, Reason: reason, Message: "job says " + reason}
	}

	tests := []struct {
		name             string
		in               RunnerHealthInput
		wantReady        metav1.ConditionStatus
		wantReconciling  metav1.ConditionStatus
		wantStalled      metav1.ConditionStatus
		wantReason       string
		wantMsgContains  string
		wantSince        *metav1.Time
		wantRequeueAfter time.Duration
	}{
		{
			name:             "first not-ready starts the clock",
			in:               RunnerHealthInput{WatchWorkloads: true, Workloads: notReady, Timeout: 30 * time.Minute, Now: now},
			wantReady:        metav1.ConditionFalse,
			wantReconciling:  metav1.ConditionTrue,
			wantStalled:      metav1.ConditionFalse,
			wantReason:       constants.ProgressingReason,
			wantMsgContains:  "StatefulSet ns/b",
			wantSince:        ago(0),
			wantRequeueAfter: 30 * time.Minute,
		},
		{
			name:             "progressing within timeout requeues for the remainder",
			in:               RunnerHealthInput{WatchWorkloads: true, Workloads: notReady, Timeout: 30 * time.Minute, ProgressingSince: ago(10 * time.Minute), Now: now},
			wantReady:        metav1.ConditionFalse,
			wantReconciling:  metav1.ConditionTrue,
			wantStalled:      metav1.ConditionFalse,
			wantReason:       constants.ProgressingReason,
			wantSince:        ago(10 * time.Minute),
			wantRequeueAfter: 20 * time.Minute,
		},
		{
			name:            "timeout reached stalls",
			in:              RunnerHealthInput{WatchWorkloads: true, Workloads: notReady, Timeout: 30 * time.Minute, ProgressingSince: ago(30 * time.Minute), Now: now, JobCondition: job(constants.SuccessfulRunReason)},
			wantReady:       metav1.ConditionFalse,
			wantReconciling: metav1.ConditionFalse,
			wantStalled:     metav1.ConditionTrue,
			wantReason:      constants.StableStateTimeoutReason,
			wantMsgContains: "not stable after 30m0s: StatefulSet ns/b",
			wantSince:       ago(30 * time.Minute),
		},
		{
			name:            "timeout disabled never stalls or requeues",
			in:              RunnerHealthInput{WatchWorkloads: true, Workloads: notReady, Timeout: 0, ProgressingSince: ago(24 * time.Hour), Now: now},
			wantReady:       metav1.ConditionFalse,
			wantReconciling: metav1.ConditionTrue,
			wantStalled:     metav1.ConditionFalse,
			wantReason:      constants.ProgressingReason,
			wantSince:       ago(24 * time.Hour),
		},
		{
			name:            "zero matched workloads counts as not stable",
			in:              RunnerHealthInput{WatchWorkloads: true, Timeout: 30 * time.Minute, ProgressingSince: ago(31 * time.Minute), Now: now},
			wantReady:       metav1.ConditionFalse,
			wantReconciling: metav1.ConditionFalse,
			wantStalled:     metav1.ConditionTrue,
			wantReason:      constants.StableStateTimeoutReason,
			wantMsgContains: "no workloads match workloadSelector",
			wantSince:       ago(31 * time.Minute),
		},
		{
			name:            "stable clears the clock and waits for the job",
			in:              RunnerHealthInput{WatchWorkloads: true, Workloads: ready, Timeout: 30 * time.Minute, ProgressingSince: ago(40 * time.Minute), Now: now, JobCondition: job(constants.PendingReason)},
			wantReady:       metav1.ConditionFalse,
			wantReconciling: metav1.ConditionTrue,
			wantStalled:     metav1.ConditionFalse,
			wantReason:      constants.WaitingForJobReason,
		},
		{
			name:            "stable without a job condition waits for the job",
			in:              RunnerHealthInput{WatchWorkloads: true, Workloads: ready, Timeout: 30 * time.Minute, Now: now},
			wantReady:       metav1.ConditionFalse,
			wantReconciling: metav1.ConditionTrue,
			wantStalled:     metav1.ConditionFalse,
			wantReason:      constants.WaitingForJobReason,
		},
		{
			name:            "stable and job succeeded is ready",
			in:              RunnerHealthInput{WatchWorkloads: true, Workloads: ready, Timeout: 30 * time.Minute, Now: now, JobCondition: job(constants.SuccessfulRunReason)},
			wantReady:       metav1.ConditionTrue,
			wantReconciling: metav1.ConditionFalse,
			wantStalled:     metav1.ConditionFalse,
			wantReason:      constants.JobSucceededReason,
			wantMsgContains: "job says Successful",
		},
		{
			name:            "stable and job failed stalls",
			in:              RunnerHealthInput{WatchWorkloads: true, Workloads: ready, Timeout: 30 * time.Minute, Now: now, JobCondition: job(constants.FailedRunReason)},
			wantReady:       metav1.ConditionFalse,
			wantReconciling: metav1.ConditionFalse,
			wantStalled:     metav1.ConditionTrue,
			wantReason:      constants.JobFailedReason,
		},
		{
			name:            "no workloadSelector ignores workloads and the clock",
			in:              RunnerHealthInput{WatchWorkloads: false, Timeout: 30 * time.Minute, ProgressingSince: ago(time.Hour), Now: now, JobCondition: job(constants.SuccessfulRunReason)},
			wantReady:       metav1.ConditionTrue,
			wantReconciling: metav1.ConditionFalse,
			wantStalled:     metav1.ConditionFalse,
			wantReason:      constants.JobSucceededReason,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluateRunnerHealth(tt.in)

			want := map[string]metav1.ConditionStatus{
				constants.ReadyType:       tt.wantReady,
				constants.ReconcilingType: tt.wantReconciling,
				constants.StalledType:     tt.wantStalled,
			}
			if len(got.Conditions) != len(want) {
				t.Fatalf("got %d conditions, want %d", len(got.Conditions), len(want))
			}
			for _, c := range got.Conditions {
				if c.Status != want[c.Type] {
					t.Errorf("%s = %s, want %s", c.Type, c.Status, want[c.Type])
				}
				if c.Reason != tt.wantReason {
					t.Errorf("%s reason = %q, want %q", c.Type, c.Reason, tt.wantReason)
				}
				if !strings.Contains(c.Message, tt.wantMsgContains) {
					t.Errorf("%s message = %q, want it to contain %q", c.Type, c.Message, tt.wantMsgContains)
				}
			}

			if (got.ProgressingSince == nil) != (tt.wantSince == nil) ||
				(got.ProgressingSince != nil && !got.ProgressingSince.Equal(tt.wantSince)) {
				t.Errorf("progressingSince = %v, want %v", got.ProgressingSince, tt.wantSince)
			}
			if got.RequeueAfter != tt.wantRequeueAfter {
				t.Errorf("requeueAfter = %s, want %s", got.RequeueAfter, tt.wantRequeueAfter)
			}
		})
	}
}

func TestDescribeStabilityCapsListedWorkloads(t *testing.T) {
	var workloads []v1alpha1.WatchedResource
	for _, n := range []string{"a", "b", "c", "d", "e", "f", "g"} {
		workloads = append(workloads, v1alpha1.WatchedResource{Kind: "Deployment", Namespace: "ns", Name: n})
	}

	msg, stable := describeStability(workloads)
	if stable {
		t.Fatal("expected not stable")
	}
	if !strings.HasSuffix(msg, "Deployment ns/e and 2 more") {
		t.Errorf("message = %q", msg)
	}
}
