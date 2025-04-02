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

var _ = Describe("controller", Ordered, func() {
	const runnerNs = "hestia-deployment-config-instance"
	const dcf1Ns = "hestia-deployment-config-1"
	const dcf2Ns = "hestia-deployment-config-2"

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
		It("should watch deployment-config", func() {
			By("creating deployment-config 1 in namespace")
			replacements := map[string]string{
				"name":           "deployment-config-1",
				"readinessDelay": "1",
			}
			utils.ApplyFixtureTemplate("./test/e2e/fixtures/deployment_configs/busybox.yaml", dcf1Ns, replacements)

			By("creating deployment-config 2 in namespace")
			replacements = map[string]string{
				"name":           "deployment-config-2",
				"readinessDelay": "2",
			}
			utils.ApplyFixtureTemplate("./test/e2e/fixtures/deployment_configs/busybox.yaml", dcf2Ns, replacements)

			By("creating runner")
			replacements = map[string]string{
				"name":        "deployment-config-runner",
				"jobDuration": "1",
			}
			utils.ApplyFixtureTemplate("./test/e2e/fixtures/deployment_configs/runner.yaml", runnerNs, replacements)

			By("validate runner to be reconciled")
			runner := &v1alpha1.Runner{
				ObjectMeta: v12.ObjectMeta{
					Name:      replacements["name"],
					Namespace: runnerNs,
				},
			}
			utils.WaitForResource(runner, func() bool {
				return runner.Status.IsReady()
			}, "60s", "1s")
			utils.MatchYAMLResource(runner, "reconciled")

			By("validate job-config get created and track deployment-config readiness")
			jobConfig := &v13.ConfigMap{
				ObjectMeta: v12.ObjectMeta{
					Name:      replacements["name"],
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
