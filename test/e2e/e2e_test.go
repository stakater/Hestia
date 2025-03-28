/*
Copyright 2025.

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
	. "github.com/onsi/ginkgo/v2"
	"github.com/stakater/hestia-operator/test/utils"
)

const namespace = "hestia-instance"

var _ = Describe("controller", Ordered, func() {
	BeforeAll(func() {
		By("creating manager namespace")
		_ = utils.Run("oc", "new-project", namespace)
	})

	AfterAll(func() {
		By("removing manager namespace")
		_ = utils.Run("oc", "delete", "project", namespace)
	})

	Context("Operator", func() {
		It("should run successfully", func() {
			//var controllerPodName string
			By("validating that the controller-manager pod is running as expected")
		})
	})
})
