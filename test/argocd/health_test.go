package argocd

import (
	"os"
	"path/filepath"
	"testing"

	lua "github.com/yuin/gopher-lua"
	"sigs.k8s.io/yaml"
)

// healthCheckDir follows the argo-cd resource_customizations layout so the check can be upstreamed as-is
var healthCheckDir = filepath.Join("..", "..", "config", "argocd", "e2e.stakater.com", "Runner")

type healthTestCases struct {
	Tests []struct {
		HealthStatus struct {
			Status  string `json:"status"`
			Message string `json:"message"`
		} `json:"healthStatus"`
		InputPath string `json:"inputPath"`
	} `json:"tests"`
}

func TestRunnerHealthCheck(t *testing.T) {
	script, err := os.ReadFile(filepath.Join(healthCheckDir, "health.lua"))
	if err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(healthCheckDir, "health_test.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var cases healthTestCases
	if err := yaml.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}

	for _, tc := range cases.Tests {
		t.Run(tc.InputPath, func(t *testing.T) {
			input, err := os.ReadFile(filepath.Join(healthCheckDir, tc.InputPath))
			if err != nil {
				t.Fatal(err)
			}
			var obj map[string]interface{}
			if err := yaml.Unmarshal(input, &obj); err != nil {
				t.Fatal(err)
			}

			status, message := runHealthCheck(t, string(script), obj)
			if status != tc.HealthStatus.Status {
				t.Errorf("status = %q, want %q", status, tc.HealthStatus.Status)
			}
			if message != tc.HealthStatus.Message {
				t.Errorf("message = %q, want %q", message, tc.HealthStatus.Message)
			}
		})
	}
}

// runHealthCheck evaluates the script the way Argo CD does: obj as a global, the returned table as the result
func runHealthCheck(t *testing.T, script string, obj map[string]interface{}) (string, string) {
	t.Helper()

	l := lua.NewState()
	defer l.Close()

	l.SetGlobal("obj", toLua(l, obj))
	if err := l.DoString(script); err != nil {
		t.Fatal(err)
	}

	result, ok := l.Get(-1).(*lua.LTable)
	if !ok {
		t.Fatalf("health check returned %s, want a table", l.Get(-1).Type())
	}

	return lua.LVAsString(result.RawGetString("status")), lua.LVAsString(result.RawGetString("message"))
}

func toLua(l *lua.LState, value interface{}) lua.LValue {
	switch v := value.(type) {
	case map[string]interface{}:
		table := l.NewTable()
		for key, item := range v {
			table.RawSetString(key, toLua(l, item))
		}
		return table
	case []interface{}:
		table := l.NewTable()
		for i, item := range v {
			table.RawSetInt(i+1, toLua(l, item))
		}
		return table
	case string:
		return lua.LString(v)
	case float64:
		return lua.LNumber(v)
	case bool:
		return lua.LBool(v)
	default:
		return lua.LNil
	}
}
