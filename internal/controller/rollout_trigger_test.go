package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	e2ev1alpha1 "github.com/stakater/hestia-operator/api/v1alpha1"
)

var _ = Describe("Runner rollout trigger", func() {
	ctx := context.Background()
	runnerKey := types.NamespacedName{Name: "rollout-runner", Namespace: "default"}
	deploymentKey := types.NamespacedName{Name: "fast-app", Namespace: "default"}

	var reconciler *RunnerReconciler

	// markReady sets the status a finished rollout leaves, without ever passing through not-ready
	markReady := func() {
		d := &appsv1.Deployment{}
		Expect(k8sClient.Get(ctx, deploymentKey, d)).To(Succeed())
		n := *d.Spec.Replicas
		d.Status = appsv1.DeploymentStatus{ObservedGeneration: d.Generation, Replicas: n, UpdatedReplicas: n, ReadyReplicas: n, AvailableReplicas: n}
		Expect(k8sClient.Status().Update(ctx, d)).To(Succeed())
	}

	jobConfig := func() map[string]string {
		_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: runnerKey})
		Expect(err).NotTo(HaveOccurred())
		cm := &corev1.ConfigMap{}
		Expect(k8sClient.Get(ctx, runnerKey, cm)).To(Succeed())
		return cm.Data
	}

	BeforeEach(func() {
		reconciler = &RunnerReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
		Expect(k8sClient.Create(ctx, &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: deploymentKey.Name, Namespace: deploymentKey.Namespace, Labels: map[string]string{"e2e": "fast"}},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "fast-app"}},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "fast-app"}},
					Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "busybox"}}},
				},
			},
		})).To(Succeed())
		markReady()
		Expect(k8sClient.Create(ctx, &e2ev1alpha1.Runner{
			ObjectMeta: metav1.ObjectMeta{Name: runnerKey.Name, Namespace: runnerKey.Namespace},
			Spec:       e2ev1alpha1.RunnerSpec{WorkloadSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"e2e": "fast"}}},
		})).To(Succeed())
	})

	AfterEach(func() {
		Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, &e2ev1alpha1.Runner{ObjectMeta: metav1.ObjectMeta{Name: runnerKey.Name, Namespace: runnerKey.Namespace}}))).To(Succeed())
		Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: runnerKey.Name, Namespace: runnerKey.Namespace}}))).To(Succeed())
		Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: deploymentKey.Name, Namespace: deploymentKey.Namespace}}))).To(Succeed())
	})

	It("changes the job config after a rollout that was never seen not ready", func() {
		before := jobConfig()

		By("restarting the deployment and finishing the rollout before the next reconcile")
		d := &appsv1.Deployment{}
		Expect(k8sClient.Get(ctx, deploymentKey, d)).To(Succeed())
		d.Spec.Template.Annotations = map[string]string{"kubectl.kubernetes.io/restartedAt": "2026-10-07T12:00:00Z"}
		Expect(k8sClient.Update(ctx, d)).To(Succeed())
		markReady()

		after := jobConfig()
		Expect(after).NotTo(Equal(before))
		Expect(after).To(HaveKeyWithValue("apps_v1-deployment-default-fast-app", "true"))
		Expect(after["runVersion"]).NotTo(Equal(before["runVersion"]))

		By("reconciling again with nothing changed")
		Expect(jobConfig()["runVersion"]).To(Equal(after["runVersion"]))
	})
})
