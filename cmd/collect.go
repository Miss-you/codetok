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
	return collectUsageEventsFromProvidersInRange(cmd, providers, provider.UsageEventCollectOptions{})
}

func collectUsageEventsFromProvidersInRange(cmd *cobra.Command, providers []provider.Provider, opts provider.UsageEventCollectOptions) ([]provider.UsageEvent, error) {
	var allEvents []provider.UsageEvent
	err := forEachUsageEventBatchFromProvidersInRange(cmd, providers, opts, func(events []provider.UsageEvent) error {
		allEvents = append(allEvents, events...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return allEvents, nil
}

func forEachUsageEventFromProvidersInRange(cmd *cobra.Command, providers []provider.Provider, opts provider.UsageEventCollectOptions, consume func(provider.UsageEvent) error) error {
	return forEachUsageEventBatchFromProvidersInRange(cmd, providers, opts, func(events []provider.UsageEvent) error {
		for _, event := range events {
			if err := consume(event); err != nil {
				return err
			}
		}
		return nil
	})
}

func forEachUsageEventBatchFromProvidersInRange(cmd *cobra.Command, providers []provider.Provider, opts provider.UsageEventCollectOptions, consume func([]provider.UsageEvent) error) error {
	providerFilter, _ := cmd.Flags().GetString("provider")
	baseDir, _ := cmd.Flags().GetString("base-dir")

	filtered := provider.FilterProviders(providers, providerFilter)

	for _, p := range filtered {
		dir := baseDir
		if providerDir, _ := cmd.Flags().GetString(providerDirFlag(p.Name())); providerDir != "" {
			dir = providerDir
		}

		if eventProvider, ok := p.(provider.UsageEventProvider); ok {
			events, err := collectProviderUsageEvents(eventProvider, dir, opts)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return fmt.Errorf("collecting usage events from %s: %w", p.Name(), err)
			}
			if err := consume(events); err != nil {
				return fmt.Errorf("processing usage events from %s: %w", p.Name(), err)
			}
			continue
		}

		sessions, err := p.CollectSessions(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("collecting sessions for usage events from %s: %w", p.Name(), err)
		}
		events := make([]provider.UsageEvent, 0, len(sessions))
		for _, session := range sessions {
			events = append(events, provider.UsageEvent{
				ProviderName: session.ProviderName,
				ModelName:    session.ModelName,
				SessionID:    session.SessionID,
				Title:        session.Title,
				WorkDirHash:  session.WorkDirHash,
				Timestamp:    session.StartTime,
				TokenUsage:   session.TokenUsage,
			})
		}
		if err := consume(events); err != nil {
			return fmt.Errorf("processing usage events from %s: %w", p.Name(), err)
		}
	}

	return nil
}

func collectProviderUsageEvents(eventProvider provider.UsageEventProvider, dir string, opts provider.UsageEventCollectOptions) ([]provider.UsageEvent, error) {
	if opts.HasRange() {
		if rangeProvider, ok := eventProvider.(provider.RangeAwareUsageEventProvider); ok {
			return rangeProvider.CollectUsageEventsInRange(dir, opts)
		}
	}
	return eventProvider.CollectUsageEvents(dir)
}
