package utils

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stakater/hestia-operator/api/v1alpha1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"strings"
	"time"
)

type E2ETestEnv struct {
	K8sClient   client.Client
	Environment *envtest.Environment

	restConfig        *rest.Config
	operatorNamespace string
	dockerUSer        string
	imageStream       string
	imageName         string
	imageTag          string
}

func NewE2ETestEnv(operatorName string) *E2ETestEnv {
	env := &E2ETestEnv{
		operatorNamespace: fmt.Sprintf("%s-operator-system", operatorName),
		imageStream:       "e2e",
		dockerUSer:        "e2e-docker",
		imageName:         operatorName,
		imageTag:          "e2e",
	}

	return env
}

func (env *E2ETestEnv) Setup() {
	env.Environment = &envtest.Environment{
		// Set to true to use an existing cluster
		UseExistingCluster: &[]bool{true}[0],
	}

	var err error
	env.restConfig, err = env.Environment.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(env.restConfig).NotTo(BeNil())

	err = v1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// Create a client for the test
	env.K8sClient, err = client.New(env.restConfig, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(env.K8sClient).NotTo(BeNil())

	// Create namespace for the operator
	By("creating manager namespace")
	_ = Run("sh", "-c", fmt.Sprintf("oc new-project %s || oc project %s", env.operatorNamespace, env.operatorNamespace))

	// Create user for pushing images
	By("creating docker user")
	Run("sh", "-c", fmt.Sprintf("oc create serviceaccount %s || echo 'Service account already exists'", env.dockerUSer))

	By("grant docker user permissions")
	Run("oc", "policy", "add-role-to-user", "system:image-builder", "-z", env.dockerUSer)

	By("patch image-stream default route")
	_ = Run("oc",
		"patch",
		"configs.imageregistry.operator.openshift.io/cluster",
		"--type=merge",
		"-p",
		"{\"spec\":{\"defaultRoute\":true}}")

	By("creating image-stream")
	_ = Run("sh", "-c", fmt.Sprintf("oc create is %s || echo 'Image-stream already exists'", env.imageStream))

	By("fetch the image-stream route path")
	output := Run("oc", "get", "route", "default-route", "-n", "openshift-image-registry", "--template={{ .spec.host }}")
	route := strings.TrimSpace(output)

	By("create sa token")
	output = Run("oc", "create", "token", env.dockerUSer, "-n", env.operatorNamespace)
	SetShellENV("SA_TOKEN", output)

	By("login to image-stream")
	output = RunShell("docker", "login", "-u", env.dockerUSer, "-p $SA_TOKEN", route)

	var e2eTestimage = fmt.Sprintf("%s/%s/%s:%s", route, env.operatorNamespace, env.imageName, env.imageTag)
	By("building the manager(Operator) image")
	_ = Run("make", "docker-build", fmt.Sprintf("IMG=%s", e2eTestimage))

	By("push the manager(Operator) image to image-stream")
	_ = RunShell("docker", "push", e2eTestimage)

	By("get the image url for direct referencing")
	output = RunShell("oc", "get", "istag", fmt.Sprintf("%s:%s", env.imageName, env.imageTag), "-o", "jsonpath='{.image.dockerImageReference}'")

	By("installing CRDs")
	_ = Run("make", "install")

	By("deploying the controller-manager")
	_ = Run("make", "deploy", fmt.Sprintf("IMG=%s", output))

	By("validating that the controller-manager pod is running as expected")
	verifyControllerUp := func() error {
		// Get pod name

		podOutput := Run("oc", "get",
			"pods", "-l", "control-plane=controller-manager",
			"-o", "go-template={{ range .items }}"+
				"{{ if not .metadata.deletionTimestamp }}"+
				"{{ .metadata.name }}"+
				"{{ \"\\n\" }}{{ end }}{{ end }}",
			"-n", env.operatorNamespace,
		)

		podNames := GetNonEmptyLines(string(podOutput))
		if len(podNames) != 1 {
			return fmt.Errorf("expect 1 controller pods running, but got %d", len(podNames))
		}
		controllerPodName := podNames[0]
		ExpectWithOffset(2, controllerPodName).Should(ContainSubstring("controller-manager"))

		// Validate pod status
		status := Run("oc", "get",
			"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
			"-n", env.operatorNamespace,
		)
		if string(status) != "Running" {
			return fmt.Errorf("controller pod in %s status", status)
		}
		return nil
	}
	EventuallyWithOffset(1, verifyControllerUp, time.Minute, time.Second).Should(Succeed())
}

func (env *E2ETestEnv) Teardown() {
	err := env.Environment.Stop()
	Expect(err).NotTo(HaveOccurred())
}

var TestEnvironment *E2ETestEnv
