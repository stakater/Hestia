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

package utils

import (
	"fmt"
	"github.com/onsi/gomega"
	"os"
	"os/exec"
	"reflect"
	"strings"

	. "github.com/onsi/ginkgo/v2" //nolint:golint,revive
)

func warnError(err error) {
	_, _ = fmt.Fprintf(GinkgoWriter, "warning: %v\n", err)
}

// Run executes the provided command within this context
func Run(name string, args ...string) (string, error) {
	dir, _ := GetProjectDir()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir

	if err := os.Chdir(cmd.Dir); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "chdir dir: %s\n", err)
		return "", fmt.Errorf("chdir dir: %s\n", err.Error())
	}

	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	command := strings.Join(cmd.Args, " ")
	_, _ = fmt.Fprintf(GinkgoWriter, "running: %s\n", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s failed with error: (%v) %s", command, err.Error(), string(output))
	}

	return strings.TrimSpace(string(output)), nil
}

func RunShell(args ...string) (string, error) {
	return Run("sh", "-c", strings.Join(args, " "))
}

func SetShellENV(env string, value string) error {
	return os.Setenv(env, value)
}

// GetNonEmptyLines converts given command output string into individual objects
// according to line breakers, and ignores the empty elements in it.
func GetNonEmptyLines(output string) []string {
	var res []string
	elements := strings.Split(output, "\n")
	for _, element := range elements {
		if element != "" {
			res = append(res, element)
		}
	}

	return res
}

// GetProjectDir will return the directory where the project is
func GetProjectDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return wd, err
	}
	wd = strings.Replace(wd, "/test/e2e", "", -1)
	return wd, nil
}

func AssertError(err error, msg ...string) {
	if err != nil {
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), fmt.Sprintf("[%s] %s", err.Error(), strings.Join(msg, " ")))
	}
}

func GetKind(obj interface{}) string {
	t := reflect.TypeOf(obj)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}
