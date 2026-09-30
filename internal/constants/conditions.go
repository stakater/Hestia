package constants

import "time"

var (
	JobStatusType       = "JobCompleted"
	SuccessfulRunReason = "Successful"
	FailedRunReason     = "Failed"
	PendingReason       = "Pending"
	JobNotFoundReason   = "JobNotFound"
)

// kstatus condition types, recognised by Argo CD (via the shipped health check), Flux and kubectl wait
var (
	ReadyType       = "Ready"
	ReconcilingType = "Reconciling"
	StalledType     = "Stalled"

	ProgressingReason        = "Progressing"
	StableStateTimeoutReason = "StableStateTimeout"
	WaitingForJobReason      = "WaitingForJob"
	JobFailedReason          = "JobFailed"
	JobSucceededReason       = "JobSucceeded"
)

// DefaultStableStateTimeout mirrors the CRD default for spec.stableStateTimeout
const DefaultStableStateTimeout = 30 * time.Minute
