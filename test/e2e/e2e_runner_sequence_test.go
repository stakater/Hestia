package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/redhat-cop/operator-utils/pkg/util/apis"
	"github.com/stakater/hestia-operator/api/v1alpha1"
	"github.com/stakater/hestia-operator/internal/constants"
	"github.com/stakater/hestia-operator/test/utils"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("controller", Ordered, func() {
	const runner1Ns = "hestia-runner1-instance"
	const runner2Ns = "hestia-runner2-instance"
	const app1Ns = "hestia-app-1"
	const app2Ns = "hestia-app-2"

	BeforeAll(func() {
		By("creating namespaces")
		for _, ns := range []string{runner1Ns, runner2Ns, app1Ns, app2Ns} {
			_, err := utils.RunShell("oc", "new-project", ns, "||", "oc", "project", ns)
			Expect(err).NotTo(HaveOccurred())
		}
	})

	AfterAll(func() {
		By("removing namespaces")
		for _, ns := range []string{runner1Ns, runner2Ns, app1Ns, app2Ns} {
			_, err := utils.Run("oc", "delete", "project", ns)
			Expect(err).NotTo(HaveOccurred())
		}
	})

	Context("operator", func() {
		It("should schedule job sequence by watching apps", func() {
			By("creating app 1")
			replacements := map[string]interface{}{
				"name":           "app-1",
				"readinessDelay": "1",
				"appLabel":       "runner-1-app",
			}
			utils.ApplyFixtureTemplate("./test/e2e/fixtures/sequence/busybox.yaml", app1Ns, replacements)

			By("creating runner 1")
			replacements = map[string]interface{}{
				"name":          "runner-1",
				"jobDuration":   "10",
				"appLabel":      "runner-1-app",
				"runnerLabel":   "runner-1",
				"sequenceLabel": "runner-sequence",
			}
			utils.ApplyFixtureTemplate("./test/e2e/fixtures/sequence/runner-1.yaml", runner1Ns, replacements)

			By("creating runner 2")
			replacements = map[string]interface{}{
				"name":             "runner-2",
				"jobDuration":      "10",
				"watchRunnerLabel": "runner-1",
				"sequenceLabel":    "runner-sequence",
			}
			utils.ApplyFixtureTemplate("./test/e2e/fixtures/sequence/runner-2.yaml", runner2Ns, replacements)

			By("validate runner 1 finished")
			runner1 := &v1alpha1.Runner{
				ObjectMeta: v1.ObjectMeta{
					Name:      "runner-1",
					Namespace: runner1Ns,
				},
			}
			utils.WaitForResource(runner1, func() bool {
				condition, ok := apis.GetCondition(constants.JobStatusType, runner1.Status.Conditions.Conditions)
				return ok && condition.Status == v1.ConditionTrue
			}, "60s", "1s")
			utils.MatchYAMLResource(runner1, "runner-1")

			By("validate runner 2 finished")
			runner2 := &v1alpha1.Runner{
				ObjectMeta: v1.ObjectMeta{
					Name:      "runner-2",
					Namespace: runner2Ns,
				},
			}
			utils.WaitForResource(runner2, func() bool {
				condition, ok := apis.GetCondition(constants.JobStatusType, runner2.Status.Conditions.Conditions)
				return ok && condition.Status == v1.ConditionTrue
			}, "60s", "1s")
			utils.MatchYAMLResource(runner2, "runner-2")

			Expect(runner1.Status.LastSuccessfulRun.Before(&runner1.Status.LastSuccessfulRun))
		})
	})
})
