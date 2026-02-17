package provider

import "testing"

// testProvider is a minimal Provider implementation for testing.
type testProvider struct {
	name string
}

func (p *testProvider) Name() string { return p.name }
func (p *testProvider) CollectSessions(baseDir string) ([]SessionInfo, error) {
	return nil, nil
}

func TestRegistryAllProviders(t *testing.T) {
	// Save and restore original registry state.
	registryMu.Lock()
	orig := registry
	registry = nil
	registryMu.Unlock()
	defer func() {
		registryMu.Lock()
		registry = orig
		registryMu.Unlock()
	}()

	Register(&testProvider{name: "alpha"})
	Register(&testProvider{name: "beta"})

	providers := Registry()
	if len(providers) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(providers))
	}
	if providers[0].Name() != "alpha" {
		t.Errorf("providers[0].Name() = %q, want %q", providers[0].Name(), "alpha")
	}
	if providers[1].Name() != "beta" {
		t.Errorf("providers[1].Name() = %q, want %q", providers[1].Name(), "beta")
	}
}

func TestFilterByName(t *testing.T) {
	providers := []Provider{
		&testProvider{name: "kimi"},
		&testProvider{name: "claude"},
		&testProvider{name: "codex"},
	}

	// Filter for "kimi"
	filtered := FilterProviders(providers, "kimi")
	if len(filtered) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(filtered))
	}
	if filtered[0].Name() != "kimi" {
		t.Errorf("Name() = %q, want %q", filtered[0].Name(), "kimi")
	}

	// Empty filter returns all
	all := FilterProviders(providers, "")
	if len(all) != 3 {
		t.Fatalf("expected 3 providers with empty filter, got %d", len(all))
	}

	// Non-existent provider returns empty
	none := FilterProviders(providers, "nonexistent")
	if len(none) != 0 {
		t.Fatalf("expected 0 providers for nonexistent, got %d", len(none))
	}
}
