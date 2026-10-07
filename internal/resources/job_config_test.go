package resources

import "testing"

func TestNextRunVersion(t *testing.T) {
	base := map[string]string{
		"apps_v1-deployment-ns-gw":            "true",
		"apps_v1-deployment-ns-gw.generation": "3",
		"generation":                          "1",
		RunVersionKey:                         "7",
	}
	with := func(changes map[string]string, drop ...string) map[string]string {
		m := map[string]string{}
		for k, v := range base {
			m[k] = v
		}
		for k, v := range changes {
			m[k] = v
		}
		for _, k := range drop {
			delete(m, k)
		}
		return m
	}

	tests := []struct {
		name            string
		previous        map[string]string
		current         map[string]string
		resourceVersion string
		want            string
	}{
		{"new config", nil, with(nil), "", "1"},
		{"nothing changed", base, with(nil), "100", "7"},
		{"workload rolled out", base, with(map[string]string{"apps_v1-deployment-ns-gw.generation": "4"}), "100", "8"},
		{"readiness changed", base, with(map[string]string{"apps_v1-deployment-ns-gw": "false"}), "100", "8"},
		{"runner spec changed", base, with(map[string]string{"generation": "2"}), "100", "8"},
		{"workload joined", base, with(map[string]string{"apps_v1-deployment-ns-api": "true"}), "100", "8"},
		{"workload left", base, with(nil, "apps_v1-deployment-ns-gw", "apps_v1-deployment-ns-gw.generation"), "100", "8"},
		{
			"upgrade only adds generation entries: adopt the resourceVersion the existing job carries",
			map[string]string{"apps_v1-deployment-ns-gw": "true", "generation": "1"},
			with(nil, RunVersionKey),
			"100", "100",
		},
		{
			"upgrade together with a real change: bump from the adopted resourceVersion",
			map[string]string{"apps_v1-deployment-ns-gw": "false", "generation": "1"},
			with(nil, RunVersionKey),
			"100", "101",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nextRunVersion(tt.previous, tt.current, tt.resourceVersion); got != tt.want {
				t.Errorf("nextRunVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}
