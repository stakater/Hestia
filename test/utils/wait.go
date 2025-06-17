package utils

import (
	"context"
	"fmt"
	"strings"

	"github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func WaitForResource(obj client.Object, validateFnc func() bool, timeArgs ...interface{}) {
	gomega.Eventually(func() bool {
		err := TestEnvironment.K8sClient.Get(context.Background(), client.ObjectKeyFromObject(obj), obj)
		if err != nil {
			return false
		}

		return validateFnc()
	}, timeArgs...).Should(gomega.BeTrue(), fmt.Sprintf("%s should become ready", strings.ToLower(GetKind(obj))))
}

func WaitForResources(obj client.ObjectList, options *client.ListOptions, validateFnc func() bool, timeArgs ...interface{}) {
	gomega.Eventually(func() bool {
		err := TestEnvironment.K8sClient.List(context.Background(), obj, options)
		if err != nil {
			return false
		}

		return validateFnc()
	}, timeArgs...).Should(gomega.BeTrue(), fmt.Sprintf("%s should become ready", strings.ToLower(GetKind(obj))))
}
