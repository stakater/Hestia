package resources

import (
	"context"
	"github.com/redhat-cop/operator-utils/pkg/util/apis"
	"github.com/stakater/hestia-operator/internal/constants"
	v1 "k8s.io/api/batch/v1"
	v12 "k8s.io/api/core/v1"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sort"
	"strconv"
)

var (
	JobStatusType       = "JobCompleted"
	SuccessfulRunReason = "Successful"
	FailedRunReason     = "Failed"
	PendingReason       = "Pending"
	JobNotFoundReason   = "JobNotFound"
)

type JobStatus struct {
	config *v12.ConfigMap
}

func NewJobStatus(cm *v12.ConfigMap) *JobStatus {
	return &JobStatus{
		config: cm,
	}
}

func (r *JobStatus) defaultJobLabels() map[string]string {
	return map[string]string{
		constants.RunnerLabel:         strconv.FormatBool(true),
		constants.OwnerLabel:          r.config.Labels[constants.OwnerLabel],
		constants.OwnerNamespaceLabel: r.config.Labels[constants.OwnerNamespaceLabel],
		constants.VersionLabel:        r.config.ResourceVersion,
	}
}

func (r *JobStatus) latestJob(ctx context.Context, kc client.Client) (*v1.Job, error) {
	labelSelector := labels.SelectorFromSet(r.defaultJobLabels())
	jobList := &v1.JobList{}
	listOpts := &client.ListOptions{
		Namespace:     r.config.Namespace,
		LabelSelector: labelSelector,
	}

	if err := kc.List(ctx, jobList, listOpts); err != nil {
		return nil, err
	}

	if len(jobList.Items) == 0 {
		return nil, nil
	}

	// Sort jobs by creation timestamp, newest first
	sort.Slice(jobList.Items, func(i, j int) bool {
		return jobList.Items[i].CreationTimestamp.After(jobList.Items[j].CreationTimestamp.Time)
	})

	// Return the newest job
	return &jobList.Items[0], nil
}

func getConditions(job *v1.Job) []v13.Condition {
	jobConditions := job.Status.Conditions

	var result []v13.Condition
	for _, jobCondition := range jobConditions {
		result = append(result, v13.Condition{
			Type:               string(jobCondition.Type),
			Status:             v13.ConditionStatus(jobCondition.Status),
			ObservedGeneration: job.Generation,
			LastTransitionTime: jobCondition.LastTransitionTime,
			Reason:             jobCondition.Reason,
			Message:            jobCondition.Message,
		})
	}

	return result
}

func (r *JobStatus) LastStatus(ctx context.Context, kc client.Client) v13.Condition {
	condition := v13.Condition{
		Type:   JobStatusType,
		Status: v13.ConditionFalse,
	}

	job, err := r.latestJob(ctx, kc)
	if job == nil {
		condition.Reason = JobNotFoundReason

		if err != nil {
			condition.Message = err.Error()
		} else {
			condition.Message = "No active job execution found!"
		}

		return condition
	}

	c, ok := apis.GetLastCondition(getConditions(job))
	if !ok {
		condition.Reason = PendingReason
		return condition
	}

	if c.Type == string(v1.JobComplete) || c.Type == string(v1.JobFailed) {
		condition.Status = v13.ConditionTrue
	}

	if c.Type == string(v1.JobComplete) && c.Status == v13.ConditionTrue {
		condition.Reason = SuccessfulRunReason
	} else if c.Type == string(v1.JobFailed) && c.Status == v13.ConditionTrue {
		condition.Reason = FailedRunReason
	} else {
		condition.Reason = c.Reason
	}

	condition.Message = c.Message
	condition.LastTransitionTime = v13.Now()
	return condition
}
