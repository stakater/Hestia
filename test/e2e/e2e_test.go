package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/redhat-cop/operator-utils/pkg/util/apis"
	"github.com/stakater/hestia-operator/api/v1alpha1"
	"github.com/stakater/hestia-operator/internal/constants"
	"github.com/stakater/hestia-operator/test/utils"
	v1 "k8s.io/api/batch/v1"
	v13 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const runner_ns = "hestia-instance"
const deployment1_ns = "hestia-deployment-1"
const deployment2_ns = "hestia-deployment-2"

var _ = Describe("controller", Ordered, func() {
	BeforeAll(func() {
		By("creating manager namespace")
		for _, ns := range []string{runner_ns, deployment1_ns, deployment2_ns} {
			_ = utils.RunShell("oc", "new-project", ns, "||", "oc", "project", ns)
		}
	})

	AfterAll(func() {
		By("removing manager namespace")
		By("creating manager namespace")
		for _, ns := range []string{runner_ns, deployment1_ns, deployment2_ns} {
			_ = utils.Run("oc", "delete", "project", ns)
		}
	})

	Context("operator", func() {
		It("should watch deployments", func() {
			By("creating deployment 1 in namespace")
			replacements := map[string]string{
				"name":           "deployment-1",
				"readinessDelay": "1",
			}
			utils.ApplyFixtureTemplate("./test/e2e/fixtures/deployments/busybox.yaml", deployment1_ns, replacements)

			By("creating deployment 2 in namespace")
			replacements = map[string]string{
				"name":           "deployment-2",
				"readinessDelay": "2",
			}
			utils.ApplyFixtureTemplate("./test/e2e/fixtures/deployments/busybox.yaml", deployment2_ns, replacements)

			By("creating runner")
			replacements = map[string]string{
				"name":        "runner",
				"jobDuration": "1",
			}
			utils.ApplyFixtureTemplate("./test/e2e/fixtures/deployments/runner.yaml", runner_ns, replacements)

			By("validate runner to be reconciled")
			runner := &v1alpha1.Runner{
				ObjectMeta: v12.ObjectMeta{
					Name:      replacements["name"],
					Namespace: runner_ns,
				},
			}
			utils.WaitForResource(runner, func() bool {
				return runner.Status.IsReady()
			}, "60s", "1s")
			utils.MatchYAMLResource(runner, "reconciled")

			By("validate job-config get created and track deployment readiness")
			jobConfig := &v13.ConfigMap{
				ObjectMeta: v12.ObjectMeta{
					Name:      replacements["name"],
					Namespace: runner_ns,
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
			utils.MatchYAMLResource(jobConfig)

			By("validate job get run once ready")
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
			utils.MatchYAMLResource(&jobs.Items[0], jobConfig.Name, "job", "execution")

			By("validate runner to report job run condition")
			utils.WaitForResource(runner, func() bool {
				_, ok := apis.GetCondition(constants.JobStatusType, runner.Status.Conditions.Conditions)
				return ok
			}, "60s", "1s")
			utils.MatchYAMLResource(runner, "reported")
		})
	})
})
