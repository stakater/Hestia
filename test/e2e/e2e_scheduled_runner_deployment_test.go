package e2e

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/redhat-cop/operator-utils/pkg/util/apis"
	"github.com/stakater/hestia-operator/api/v1alpha1"
	"github.com/stakater/hestia-operator/internal/constants"
	"github.com/stakater/hestia-operator/test/utils"
	v1 "k8s.io/api/batch/v1"
	v13 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("controller", Ordered, func() {
	const runnerNs = "hestia-cron-deploy"
	const dcf1Ns = "hestia-cron-deploy-1"
	const dcf2Ns = "hestia-cron-deploy-2"

	BeforeAll(func() {
		By("creating namespaces")
		for _, ns := range []string{runnerNs, dcf1Ns, dcf2Ns} {
			_, err := utils.RunShell("oc", "new-project", ns, "||", "oc", "project", ns)
			Expect(err).NotTo(HaveOccurred())
		}
	})

	AfterAll(func() {
		By("removing namespaces")
		for _, ns := range []string{runnerNs, dcf1Ns, dcf2Ns} {
			_, err := utils.Run("oc", "delete", "project", ns)
			Expect(err).NotTo(HaveOccurred())
		}
	})

	Context("operator", func() {
		It("should watch and schedule job for deployment", func() {
			By("creating deployment 1")
			replacements := map[string]interface{}{
				"name":           "cron-deploy-1",
				"readinessDelay": "1",
				"appLabel":       "cron-deploy-job",
			}
			utils.ApplyFixtureTemplate("./test/e2e/fixtures/deployments/busybox.yaml", dcf1Ns, replacements)

			By("creating deployment 2")
			replacements = map[string]interface{}{
				"name":           "cron-deploy-2",
				"readinessDelay": "2",
				"appLabel":       "cron-deploy-job",
			}
			utils.ApplyFixtureTemplate("./test/e2e/fixtures/deployments/busybox.yaml", dcf2Ns, replacements)

			By("creating scheduled runner")
			replacements = map[string]interface{}{
				"name":            "cron-deploy-runner",
				"jobDuration":     "1",
				"schedule":        "* * * * *", // every minute
				"deadlineSeconds": 120,
				"appLabel":        "cron-deploy-job",
			}
			utils.ApplyFixtureTemplate("./test/e2e/fixtures/deployments/scheduled_runner.yaml", runnerNs, replacements)

			By("validate runner reconcile")
			runner := &v1alpha1.Runner{
				ObjectMeta: v12.ObjectMeta{
					Name:      fmt.Sprintf("%s", replacements["name"]),
					Namespace: runnerNs,
				},
			}
			utils.WaitForResource(runner, func() bool {
				return runner.Status.IsReady()
			}, "1m", "1s")
			utils.MatchYAMLResource(runner, "schedule", "reconciled")

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
			}, "1m", "1s")
			utils.MatchYAMLResource(jobConfig, "job", "config")

			By("validate cron-job creation")
			cron := &v1.CronJob{
				ObjectMeta: v12.ObjectMeta{
					Name:      fmt.Sprintf("%s", replacements["name"]),
					Namespace: runnerNs,
				},
			}
			utils.WaitForResource(cron, func() bool {
				return true
			}, "1m", "1s")
			utils.MatchYAMLResource(cron, "scheduled", "execution")

			By("validate runner job status")
			utils.WaitForResource(runner, func() bool {
				condition, ok := apis.GetCondition(constants.JobStatusType, runner.Status.Conditions.Conditions)
				return ok && condition.Status == v12.ConditionTrue
			}, "2m", "5s")
			utils.MatchYAMLResource(runner, "scheduled", "reported")
		})
	})
})
