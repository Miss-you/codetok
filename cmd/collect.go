package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/miss-you/codetok/provider"
)

func collectSessions(cmd *cobra.Command) ([]provider.SessionInfo, error) {
	return collectSessionsFromProviders(cmd, provider.Registry())
}

func collectSessionsFromProviders(cmd *cobra.Command, providers []provider.Provider) ([]provider.SessionInfo, error) {
	providerFilter, _ := cmd.Flags().GetString("provider")
	baseDir, _ := cmd.Flags().GetString("base-dir")

	filtered := provider.FilterProviders(providers, providerFilter)

	var allSessions []provider.SessionInfo
	for _, p := range filtered {
		dir := baseDir
		if providerDir, _ := cmd.Flags().GetString(providerDirFlag(p.Name())); providerDir != "" {
			dir = providerDir
		}

		sessions, err := p.CollectSessions(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("collecting sessions from %s: %w", p.Name(), err)
		}
		allSessions = append(allSessions, sessions...)
	}

	return allSessions, nil
}
