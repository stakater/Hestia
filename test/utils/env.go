package utils

import (
	"fmt"
	"github.com/onsi/ginkgo/v2"
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

func (env *E2ETestEnv) Setup() error {
	env.Environment = &envtest.Environment{
		// Set to true to use an existing cluster
		UseExistingCluster: &[]bool{true}[0],
	}

	var err error
	env.restConfig, err = env.Environment.Start()
	if err != nil || env.restConfig == nil {
		return fmt.Errorf("failed to setup rest-config")
	}

	err = v1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		return fmt.Errorf("failed to setup scheme: %s", err.Error())
	}

	// Create a client for the test
	env.K8sClient, err = client.New(env.restConfig, client.Options{Scheme: scheme.Scheme})
	if err != nil || env.K8sClient == nil {
		return fmt.Errorf("failed to setup K8sClient: %s", err.Error())
	}

	// Create namespace for the operator
	ginkgo.GinkgoWriter.Println("creating operator namespace")
	_, err = Run("sh", "-c", fmt.Sprintf("oc new-project %s || oc project %s", env.operatorNamespace, env.operatorNamespace))
	if err != nil {
		return fmt.Errorf("failed to create operator namespace: %s", err.Error())
	}

	// Create user for pushing images
	ginkgo.GinkgoWriter.Println("creating docker user")
	_, err = Run("sh", "-c", fmt.Sprintf("oc create serviceaccount %s || echo 'Service account already exists'", env.dockerUSer))
	if err != nil {
		return fmt.Errorf("failed to create docker user: %s", err.Error())
	}

	ginkgo.GinkgoWriter.Println("grant docker user permissions")
	_, err = Run("oc", "policy", "add-role-to-user", "system:image-builder", "-z", env.dockerUSer)
	if err != nil {
		return fmt.Errorf("failed to grant docker user permissions: %s", err.Error())
	}

	ginkgo.GinkgoWriter.Println("patch image-stream default route")
	_, err = Run("oc",
		"patch",
		"configs.imageregistry.operator.openshift.io/cluster",
		"--type=merge",
		"-p",
		"{\"spec\":{\"defaultRoute\":true}}")
	if err != nil {
		return fmt.Errorf("failed to patch image-stream default route: %s", err.Error())
	}

	ginkgo.GinkgoWriter.Println("creating image-stream")
	_, err = Run("sh", "-c", fmt.Sprintf("oc create is %s || echo 'Image-stream already exists'", env.imageStream))
	if err != nil {
		return fmt.Errorf("failed to create image-stream: %s", err.Error())
	}

	ginkgo.GinkgoWriter.Println("fetch the image-stream route path")
	output, err := Run("oc", "get", "route", "default-route", "-n", "openshift-image-registry", "--template={{ .spec.host }}")
	route := strings.TrimSpace(output)
	if err != nil {
		return fmt.Errorf("failed to fetch image-stream route: %s", err.Error())
	}

	ginkgo.GinkgoWriter.Println("create sa token")
	output, err = Run("oc", "create", "token", env.dockerUSer, "-n", env.operatorNamespace)
	if err != nil {
		return fmt.Errorf("failed to create sa token: %s", err.Error())
	}
	err = SetShellENV("SA_TOKEN", output)
	if err != nil {
		return fmt.Errorf("failed to set sa token ENV: %s", err.Error())
	}

	ginkgo.GinkgoWriter.Println("login to image-stream")
	output, err = RunShell("docker", "login", "-u", env.dockerUSer, "-p $SA_TOKEN", route)
	if err != nil {
		return fmt.Errorf("failed to login to image-stream: %s", err.Error())
	}

	var e2eTestimage = fmt.Sprintf("%s/%s/%s:%s", route, env.operatorNamespace, env.imageName, env.imageTag)
	ginkgo.GinkgoWriter.Println("building the manager(Operator) image")
	_, err = Run("make", "docker-build", fmt.Sprintf("IMG=%s", e2eTestimage))
	if err != nil {
		return fmt.Errorf("failed to build operator image: %s", err.Error())
	}

	ginkgo.GinkgoWriter.Println("push the manager(Operator) image to image-stream")
	_, err = RunShell("docker", "push", e2eTestimage)
	if err != nil {
		return fmt.Errorf("failed to push operator image: %s", err.Error())
	}

	ginkgo.GinkgoWriter.Println("get the image url for direct referencing")
	output, err = RunShell("oc", "get", "istag", fmt.Sprintf("%s:%s", env.imageName, env.imageTag), "-o", "jsonpath='{.image.dockerImageReference}'")
	if err != nil {
		return fmt.Errorf("failed to fetch pushed image url: %s", err.Error())
	}

	ginkgo.GinkgoWriter.Println("installing CRDs")
	_, err = Run("make", "install")
	if err != nil {
		return fmt.Errorf("failed to install operator CRDs: %s", err.Error())
	}

	ginkgo.GinkgoWriter.Println("deploying the operator")
	_, err = Run("make", "deploy", fmt.Sprintf("IMG=%s", output))
	if err != nil {
		return fmt.Errorf("failed to deploy operator: %s", err.Error())
	}

	ginkgo.GinkgoWriter.Println("validating that the controller-manager pod is running as expected")
	verifyControllerUp := func() error {
		// Get pod name

		podOutput, err := Run("oc", "get",
			"pods", "-l", "control-plane=controller-manager",
			"-o", "go-template={{ range .items }}"+
				"{{ if not .metadata.deletionTimestamp }}"+
				"{{ .metadata.name }}"+
				"{{ \"\\n\" }}{{ end }}{{ end }}",
			"-n", env.operatorNamespace,
		)

		if err != nil {
			return err
		}

		podNames := GetNonEmptyLines(podOutput)
		if len(podNames) != 1 {
			return fmt.Errorf("expect 1 controller pods running, but got %d", len(podNames))
		}
		controllerPodName := podNames[0]
		if !strings.Contains(controllerPodName, "controller-manager") {
			return fmt.Errorf("pod should contain 'controller-manager', but got %s", controllerPodName)
		}

		// Validate pod status
		status, err := Run("oc", "get",
			"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
			"-n", env.operatorNamespace,
		)
		if err != nil || string(status) != "Running" {
			return fmt.Errorf("controller pod in %s status", status)
		}
		return nil
	}

	var lastErr error
	defer func() {
		deadline := time.Now().Add(time.Minute)

		// Track the last error to return if we time out
		var lastErr error

		// Retry loop
		for time.Now().Before(deadline) {
			// Try running the function
			err := verifyControllerUp()
			if err == nil {
				return
			}

			// Store the error and try again after the interval
			lastErr = err

			// Sleep for the interval
			time.Sleep(time.Second)
		}

		// If we get here, we timed out
		lastErr = fmt.Errorf("timed out after %s: %w", time.Minute, lastErr)
	}()

	return lastErr
}

func (env *E2ETestEnv) Teardown() error {
	return env.Environment.Stop()
}

var TestEnvironment *E2ETestEnv
