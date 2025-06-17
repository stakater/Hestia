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
	"os"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stakater/hestia-operator/test/utils"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
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
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	utils.TestEnvironment = utils.NewE2ETestEnv("hestia")
	err := utils.TestEnvironment.Setup()
	if err != nil {
		fmt.Printf("Test setup failed: %v\n", err)
		os.Exit(1)
	}
}

// Teardown function - called after all tests
func teardown(m *testing.M) {
	// Global test teardown
	snaps.Clean(m)
	err := utils.TestEnvironment.Teardown()
	if err != nil {
		// Log the error
		GinkgoWriter.Println(fmt.Sprintf("Test teardown failed: %v", err.Error()))
		os.Exit(1)
	}
}

// Run e2e tests using the Ginkgo runner.
func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	GinkgoWriter.Println("Starting hestia-operator suite")
	RunSpecs(t, "e2e suite")
}
