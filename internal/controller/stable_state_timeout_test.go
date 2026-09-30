package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	e2ev1alpha1 "github.com/stakater/hestia-operator/api/v1alpha1"
	"github.com/stakater/hestia-operator/internal/constants"
)

var _ = Describe("Runner stable-state timeout", func() {
	ctx := context.Background()
	runnerKey := types.NamespacedName{Name: "timeout-runner", Namespace: "default"}
	deploymentKey := types.NamespacedName{Name: "slow-app", Namespace: "default"}

	var reconciler *RunnerReconciler

	reconcileRunner := func() reconcile.Result {
		result, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: runnerKey})
		Expect(err).NotTo(HaveOccurred())
		return result
	}

	fetchRunner := func() *e2ev1alpha1.Runner {
		runner := &e2ev1alpha1.Runner{}
		Expect(k8sClient.Get(ctx, runnerKey, runner)).To(Succeed())
		return runner
	}

	condition := func(runner *e2ev1alpha1.Runner, conditionType string) *metav1.Condition {
		return apimeta.FindStatusCondition(runner.Status.Conditions.Conditions, conditionType)
	}

	BeforeEach(func() {
		reconciler = &RunnerReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}

		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: deploymentKey.Name, Namespace: deploymentKey.Namespace, Labels: map[string]string{"app": "slow-app"}},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "slow-app"}},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "slow-app"}},
					Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "busybox"}}},
				},
			},
		}
		Expect(k8sClient.Create(ctx, deployment)).To(Succeed())

		runner := &e2ev1alpha1.Runner{
			ObjectMeta: metav1.ObjectMeta{Name: runnerKey.Name, Namespace: runnerKey.Namespace},
			Spec: e2ev1alpha1.RunnerSpec{
				WorkloadSelector:   &metav1.LabelSelector{MatchLabels: map[string]string{"app": "slow-app"}},
				StableStateTimeout: &metav1.Duration{Duration: 2 * time.Second},
			},
		}
		Expect(k8sClient.Create(ctx, runner)).To(Succeed())
	})

	AfterEach(func() {
		Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, &e2ev1alpha1.Runner{ObjectMeta: metav1.ObjectMeta{Name: runnerKey.Name, Namespace: runnerKey.Namespace}}))).To(Succeed())
		Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: deploymentKey.Name, Namespace: deploymentKey.Namespace}}))).To(Succeed())
		Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: runnerKey.Name, Namespace: runnerKey.Namespace}}))).To(Succeed())
	})

	It("reports Progressing, then Stalled after the timeout, then recovers once workloads are ready", func() {
		By("starting the clock while the deployment is not ready")
		result := reconcileRunner()
		Expect(result.RequeueAfter).To(BeNumerically(">", 0))
		Expect(result.RequeueAfter).To(BeNumerically("<=", 2*time.Second))

		runner := fetchRunner()
		Expect(runner.Status.ProgressingSince).NotTo(BeNil())
		Expect(runner.Status.ObservedGeneration).To(Equal(runner.Generation))
		Expect(condition(runner, constants.ReconcilingType).Status).To(Equal(metav1.ConditionTrue))
		Expect(condition(runner, constants.ReconcilingType).Reason).To(Equal(constants.ProgressingReason))
		Expect(condition(runner, constants.StalledType).Status).To(Equal(metav1.ConditionFalse))

		By("stalling once the timeout has passed")
		Eventually(func() string {
			reconcileRunner()
			stalled := condition(fetchRunner(), constants.StalledType)
			if stalled.Status != metav1.ConditionTrue {
				return ""
			}
			return stalled.Reason
		}, 5*time.Second, 250*time.Millisecond).Should(Equal(constants.StableStateTimeoutReason))

		runner = fetchRunner()
		Expect(condition(runner, constants.ReadyType).Status).To(Equal(metav1.ConditionFalse))
		Expect(condition(runner, constants.StalledType).Message).To(ContainSubstring("Deployment default/slow-app"))

		By("recovering when the deployment becomes ready")
		deployment := &appsv1.Deployment{}
		Expect(k8sClient.Get(ctx, deploymentKey, deployment)).To(Succeed())
		deployment.Status = appsv1.DeploymentStatus{
			ObservedGeneration: deployment.Generation,
			Replicas:           1,
			UpdatedReplicas:    1,
			ReadyReplicas:      1,
			AvailableReplicas:  1,
		}
		Expect(k8sClient.Status().Update(ctx, deployment)).To(Succeed())

		result = reconcileRunner()
		Expect(result.RequeueAfter).To(BeZero())

		runner = fetchRunner()
		Expect(runner.Status.ProgressingSince).To(BeNil())
		Expect(condition(runner, constants.StalledType).Status).To(Equal(metav1.ConditionFalse))
		Expect(condition(runner, constants.ReconcilingType).Status).To(Equal(metav1.ConditionTrue))
		Expect(condition(runner, constants.ReconcilingType).Reason).To(Equal(constants.WaitingForJobReason))
	})

	It("defaults stableStateTimeout to 30m", func() {
		runner := &e2ev1alpha1.Runner{
			ObjectMeta: metav1.ObjectMeta{Name: "default-timeout-runner", Namespace: "default"},
		}
		Expect(k8sClient.Create(ctx, runner)).To(Succeed())
		DeferCleanup(func() { Expect(k8sClient.Delete(ctx, runner)).To(Succeed()) })

		Expect(runner.Spec.StableStateTimeout).NotTo(BeNil())
		Expect(runner.Spec.StableStateTimeout.Duration).To(Equal(constants.DefaultStableStateTimeout))
	})
})
