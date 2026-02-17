package provider

import "sync"

var (
	registryMu sync.Mutex
	registry   []Provider
)

// Register adds a provider to the registry.
// Providers typically call this from an init() function.
func Register(p Provider) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = append(registry, p)
}

// Registry returns all registered providers.
func Registry() []Provider {
	registryMu.Lock()
	defer registryMu.Unlock()
	result := make([]Provider, len(registry))
	copy(result, registry)
	return result
}

// FilterProviders returns providers matching the given name filter.
// If name is empty, all providers are returned.
func FilterProviders(providers []Provider, name string) []Provider {
	if name == "" {
		return providers
	}
	var filtered []Provider
	for _, p := range providers {
		if p.Name() == name {
			filtered = append(filtered, p)
		}
	}
	return filtered
}
