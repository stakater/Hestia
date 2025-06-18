package utils

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"text/template"
)

func ApplyFixtureTemplate(templatePath string, namespace string, values map[string]interface{}) string {
	// Read the template file
	templateData, err := os.ReadFile(templatePath)
	AssertError(err)

	// Parse the template
	tmpl, err := template.New("resource").Parse(string(templateData))
	AssertError(err)

	// Execute the template with the provided values
	var buffer bytes.Buffer
	err = tmpl.Execute(&buffer, values)
	AssertError(err)

	// Apply using kubectl with stdin
	args := []string{"apply", "-f-"}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}

	cmd := exec.Command("kubectl", args...)
	cmd.Stdin = &buffer
	output, err := cmd.CombinedOutput()
	AssertError(err, string(output))

	return strings.TrimSpace(string(output))
}
