package e2e

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/redhat-cop/operator-utils/pkg/util/apis"
	v1 "k8s.io/api/batch/v1"
	v13 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/stakater/hestia-operator/api/v1alpha1"
	"github.com/stakater/hestia-operator/internal/constants"
	"github.com/stakater/hestia-operator/test/utils"
)

var _ = Describe("controller", Ordered, func() {
	const runnerNs = "hestia-daemonset-instance"
	const daemonset1Ns = "hestia-daemonset-1"
	const daemonset2Ns = "hestia-daemonset-2"

	BeforeAll(func() {
		By("creating namespaces")
		for _, ns := range []string{runnerNs, daemonset1Ns, daemonset2Ns} {
			_, err := utils.RunShell("oc", "new-project", ns, "||", "oc", "project", ns)
			Expect(err).NotTo(HaveOccurred())
		}
	})

	AfterAll(func() {
		By("removing namespaces")
		for _, ns := range []string{runnerNs, daemonset1Ns, daemonset2Ns} {
			_, err := utils.Run("oc", "delete", "project", ns)
			Expect(err).NotTo(HaveOccurred())
		}
	})

	Context("operator", func() {
		It("should schedule job by watching daemonsets", func() {
			By("creating daemonset 1")
			replacements := map[string]interface{}{
				"name":           "daemonset-1",
				"readinessDelay": "1",
				"appLabel":       "e2e-daemonset",
			}
			utils.ApplyFixtureTemplate("./test/e2e/fixtures/daemonsets/busybox.yaml", daemonset1Ns, replacements)

			By("creating daemonset 2")
			replacements = map[string]interface{}{
				"name":           "daemonset-2",
				"readinessDelay": "2",
				"appLabel":       "e2e-daemonset",
			}
			utils.ApplyFixtureTemplate("./test/e2e/fixtures/daemonsets/busybox.yaml", daemonset2Ns, replacements)

			By("creating runner")
			replacements = map[string]interface{}{
				"name":        "daemonset-runner",
				"jobDuration": "1",
				"appLabel":    "e2e-daemonset",
			}
			utils.ApplyFixtureTemplate("./test/e2e/fixtures/daemonsets/runner.yaml", runnerNs, replacements)

			By("validate runner reconcile")
			runner := &v1alpha1.Runner{
				ObjectMeta: v12.ObjectMeta{
					Name:      fmt.Sprintf("%s", replacements["name"]),
					Namespace: runnerNs,
				},
			}
			utils.WaitForResource(runner, func() bool {
				return runner.Status.IsReady()
			}, "60s", "1s")
			utils.MatchYAMLResource(runner, "reconciled")

			By("validate job-config and track readiness")
			jobConfig := &v13.ConfigMap{
				ObjectMeta: v12.ObjectMeta{
					Name:      fmt.Sprintf("%s", replacements["name"]),
					Namespace: runnerNs,
				},
			}
			utils.WaitForResource(jobConfig, func() bool {
				if jobConfig.Data == nil {
					return false
				} else {
					for _, v := range jobConfig.Data {
						if v == "false" {
							return false
						}
					}

					return true
				}
			}, "60s", "1s")
			utils.MatchYAMLResource(jobConfig, "job", "config")

			By("validate job execution")
			jobs := &v1.JobList{}
			utils.WaitForResources(jobs, &client.ListOptions{
				LabelSelector: labels.SelectorFromSet(map[string]string{
					constants.OwnerLabel:          jobConfig.Labels[constants.OwnerLabel],
					constants.OwnerNamespaceLabel: jobConfig.Labels[constants.OwnerNamespaceLabel],
					constants.VersionLabel:        jobConfig.ResourceVersion,
				}),
			}, func() bool {
				if len(jobs.Items) == 0 {
					return false
				}

				for _, job := range jobs.Items {
					if job.Status.Active != 0 || job.Status.Failed != 0 {
						return false
					}
				}

				return true
			}, "60s", "1s")
			Expect(jobs.Items).To(HaveLen(1))
			utils.MatchYAMLResource(&jobs.Items[0], "execution")

			By("validate runner job status")
			utils.WaitForResource(runner, func() bool {
				condition, ok := apis.GetCondition(constants.JobStatusType, runner.Status.Conditions.Conditions)
				return ok && condition.Status == v12.ConditionTrue
			}, "60s", "1s")
			utils.MatchYAMLResource(runner, "reported")
		})
	})
})
