package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/miss-you/codetok/provider"
)

func collectUsageEvents(cmd *cobra.Command) ([]provider.UsageEvent, error) {
	return collectUsageEventsFromProviders(cmd, provider.Registry())
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

func collectUsageEventsFromProviders(cmd *cobra.Command, providers []provider.Provider) ([]provider.UsageEvent, error) {
	providerFilter, _ := cmd.Flags().GetString("provider")
	baseDir, _ := cmd.Flags().GetString("base-dir")

	filtered := provider.FilterProviders(providers, providerFilter)

	var allEvents []provider.UsageEvent
	for _, p := range filtered {
		dir := baseDir
		if providerDir, _ := cmd.Flags().GetString(providerDirFlag(p.Name())); providerDir != "" {
			dir = providerDir
		}

		if eventProvider, ok := p.(provider.UsageEventProvider); ok {
			events, err := eventProvider.CollectUsageEvents(dir)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, fmt.Errorf("collecting usage events from %s: %w", p.Name(), err)
			}
			allEvents = append(allEvents, events...)
			continue
		}

		sessions, err := p.CollectSessions(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("collecting sessions for usage events from %s: %w", p.Name(), err)
		}
		for _, session := range sessions {
			allEvents = append(allEvents, provider.UsageEvent{
				ProviderName: session.ProviderName,
				ModelName:    session.ModelName,
				SessionID:    session.SessionID,
				Title:        session.Title,
				WorkDirHash:  session.WorkDirHash,
				Timestamp:    session.StartTime,
				TokenUsage:   session.TokenUsage,
			})
		}
	}

	return allEvents, nil
}
