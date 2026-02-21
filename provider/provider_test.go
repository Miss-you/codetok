package provider

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDailyStatsJSON_SingleProviderGroup(t *testing.T) {
	ds := DailyStats{
		Date:         "2026-02-17",
		ProviderName: "claude",
		GroupBy:      "model",
		Group:        "claude-opus-4-6",
		Sessions:     2,
	}

	b, err := json.Marshal(ds)
	if err != nil {
		t.Fatalf("marshal DailyStats: %v", err)
	}
	got := string(b)

	if !strings.Contains(got, `"provider":"claude"`) {
		t.Fatalf("json missing provider field: %s", got)
	}
	if !strings.Contains(got, `"group_by":"model"`) {
		t.Fatalf("json missing group_by field: %s", got)
	}
	if !strings.Contains(got, `"group":"claude-opus-4-6"`) {
		t.Fatalf("json missing group field: %s", got)
	}
	if strings.Contains(got, `"providers"`) {
		t.Fatalf("json should omit providers for single-provider group: %s", got)
	}
}

func TestDailyStatsJSON_MultiProviderGroup(t *testing.T) {
	ds := DailyStats{
		Date:         "2026-02-17",
		ProviderName: "",
		GroupBy:      "model",
		Group:        "shared-model",
		Providers:    []string{"claude", "codex"},
		Sessions:     3,
	}

	b, err := json.Marshal(ds)
	if err != nil {
		t.Fatalf("marshal DailyStats: %v", err)
	}

	var decoded struct {
		Provider  string   `json:"provider"`
		GroupBy   string   `json:"group_by"`
		Group     string   `json:"group"`
		Providers []string `json:"providers"`
	}
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal DailyStats json: %v", err)
	}

	if decoded.Provider != "" {
		t.Fatalf("provider = %q, want empty for multi-provider group", decoded.Provider)
	}
	if decoded.GroupBy != "model" {
		t.Fatalf("group_by = %q, want %q", decoded.GroupBy, "model")
	}
	if decoded.Group != "shared-model" {
		t.Fatalf("group = %q, want %q", decoded.Group, "shared-model")
	}
	if len(decoded.Providers) != 2 || decoded.Providers[0] != "claude" || decoded.Providers[1] != "codex" {
		t.Fatalf("providers = %v, want [claude codex]", decoded.Providers)
	}
}
