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
	"github.com/stakater/hestia-operator/test/utils"
	"os"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"testing"
)

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

// Run e2e tests using the Ginkgo runner.
func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	fmt.Fprintf(GinkgoWriter, "Starting hestia-operator suite\n")
	RunSpecs(t, "e2e suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	utils.TestEnvironment = utils.NewE2ETestEnv("hestia")
	utils.TestEnvironment.Setup()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	utils.TestEnvironment.Teardown()
})
