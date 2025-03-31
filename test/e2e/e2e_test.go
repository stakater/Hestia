package e2e

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stakater/hestia-operator/test/utils"
	v13 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	Context("Operator", func() {
		It("should watch deployments", func() {
			By("creating deployment 1 in namespace")
			replacements := map[string]string{
				"name":           "deployment-1",
				"readinessDelay": "5",
			}
			utils.ApplyFixtureTemplate("./test/e2e/fixtures/deployments/busybox.yaml", deployment1_ns, replacements)

			By("creating deployment 2 in namespace")
			replacements = map[string]string{
				"name":           "deployment-2",
				"readinessDelay": "10",
			}
			utils.ApplyFixtureTemplate("./test/e2e/fixtures/deployments/busybox.yaml", deployment2_ns, replacements)

			By("creating runner")
			By("creating deployment 2 in namespace")
			replacements = map[string]string{
				"name":        "runner",
				"jobDuration": "10",
			}
			utils.ApplyFixtureTemplate("./test/e2e/fixtures/deployments/runner.yaml", runner_ns, replacements)

			// Validate job configmap exists
			Eventually(func() bool {
				cfg := &v13.ConfigMap{
					ObjectMeta: v12.ObjectMeta{
						Name:      replacements["name"],
						Namespace: runner_ns,
					},
				}
				err := k8sClient.Get(context.Background(), client.ObjectKeyFromObject(cfg), cfg)
				if err != nil {
					return false
				}

				return true
			}, "60s", "5s").Should(BeTrue(), "resource should become ready")
		})
	})
})
