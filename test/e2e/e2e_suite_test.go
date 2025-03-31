/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"fmt"
	"github.com/gkampitakis/go-snaps/snaps"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stakater/hestia-operator/api/v1alpha1"
	"github.com/stakater/hestia-operator/test/utils"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"strings"
	"testing"
	"time"
)

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

func TestMain(m *testing.M) {
	// Setup before any tests run
	setup()

	// Run all tests and get exit code
	code := m.Run()

	// Teardown after all tests finish
	teardown(m)

	// Exit with the same code
	os.Exit(code)
}

// Setup function - called before all tests
func setup() {
	// Global test setup
}

// Teardown function - called after all tests
func teardown(m *testing.M) {
	// Global test teardown
	snaps.Clean(m)
}

const operatorNamespace = "hestia-operator-system"
const imageStream = "e2e"
const dockerUSer = "hestia-docker"
const ImageName = "hestia-operator"
const ImageTag = "e2e"

// Run e2e tests using the Ginkgo runner.
func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	fmt.Fprintf(GinkgoWriter, "Starting hestia-operator suite\n")
	RunSpecs(t, "e2e suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		// Set to true to use an existing cluster
		UseExistingCluster: &[]bool{true}[0],
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = v1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// Create a client for the test
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// Create namespace for the operator
	By("creating manager namespace")
	_ = utils.Run("sh", "-c", fmt.Sprintf("oc new-project %s || oc project %s", operatorNamespace, operatorNamespace))

	// Create user for pushing images
	By("creating docker user")
	utils.Run("sh", "-c", fmt.Sprintf("oc create serviceaccount %s || echo 'Service account already exists'", dockerUSer))

	By("grant docker user permissions")
	utils.Run("oc", "policy", "add-role-to-user", "system:image-builder", "-z", dockerUSer)

	By("patch image-stream default route")
	_ = utils.Run("oc",
		"patch",
		"configs.imageregistry.operator.openshift.io/cluster",
		"--type=merge",
		"-p",
		"{\"spec\":{\"defaultRoute\":true}}")
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("creating image-stream")
	_ = utils.Run("sh", "-c", fmt.Sprintf("oc create is %s || echo 'Image-stream already exists'", imageStream))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("fetch the image-stream route path")
	output := utils.Run("oc", "get", "route", "default-route", "-n", "openshift-image-registry", "--template={{ .spec.host }}")
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	route := strings.TrimSpace(output)

	By("create sa token")
	output = utils.Run("oc", "create", "token", dockerUSer, "-n", operatorNamespace)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	utils.SetShellENV("SA_TOKEN", output)

	By("login to image-stream")
	output = utils.RunShell("docker", "login", "-u", dockerUSer, "-p $SA_TOKEN", route)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	var e2eTestimage = fmt.Sprintf("%s/%s/%s:%s", route, operatorNamespace, ImageName, ImageTag)
	By("building the manager(Operator) image")
	_ = utils.Run("make", "docker-build", fmt.Sprintf("IMG=%s", e2eTestimage))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("push the manager(Operator) image to image-stream")
	_ = utils.RunShell("docker", "push", e2eTestimage)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("get the image url for direct referencing")
	output = utils.RunShell("oc", "get", "istag", fmt.Sprintf("%s:%s", ImageName, ImageTag), "-o", "jsonpath='{.image.dockerImageReference}'")
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("installing CRDs")
	_ = utils.Run("make", "install")
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("deploying the controller-manager")
	_ = utils.Run("make", "deploy", fmt.Sprintf("IMG=%s", output))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("validating that the controller-manager pod is running as expected")
	verifyControllerUp := func() error {
		// Get pod name

		podOutput := utils.Run("oc", "get",
			"pods", "-l", "control-plane=controller-manager",
			"-o", "go-template={{ range .items }}"+
				"{{ if not .metadata.deletionTimestamp }}"+
				"{{ .metadata.name }}"+
				"{{ \"\\n\" }}{{ end }}{{ end }}",
			"-n", operatorNamespace,
		)

		ExpectWithOffset(2, err).NotTo(HaveOccurred())
		podNames := utils.GetNonEmptyLines(string(podOutput))
		if len(podNames) != 1 {
			return fmt.Errorf("expect 1 controller pods running, but got %d", len(podNames))
		}
		controllerPodName := podNames[0]
		ExpectWithOffset(2, controllerPodName).Should(ContainSubstring("controller-manager"))

		// Validate pod status
		status := utils.Run("oc", "get",
			"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
			"-n", operatorNamespace,
		)
		ExpectWithOffset(2, err).NotTo(HaveOccurred())
		if string(status) != "Running" {
			return fmt.Errorf("controller pod in %s status", status)
		}
		return nil
	}
	EventuallyWithOffset(1, verifyControllerUp, time.Minute, time.Second).Should(Succeed())
})

var _ = AfterSuite(func() {
	By("removing manager namespace")
	_ = utils.Run("oc", "delete", "project", operatorNamespace)

	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
