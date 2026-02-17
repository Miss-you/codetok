package cmd

import (
	"fmt"
	"runtime/debug"
	"strings"
)

const (
	defaultVersion    = "dev"
	defaultCommitHash = "none"
	defaultBuildDate  = "unknown"
	develVersion      = "(devel)"
)

type buildInfo struct {
	mainVersion string
	vcsRevision string
	vcsTime     string
}

func readBuildInfo() buildInfo {
	info, ok := debug.ReadBuildInfo()
	if !ok || info == nil {
		return buildInfo{}
	}

	data := buildInfo{
		mainVersion: info.Main.Version,
	}

	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			data.vcsRevision = shortRevision(setting.Value)
		case "vcs.time":
			data.vcsTime = setting.Value
		}
	}

	return data
}

func resolveVersionInfo(v, c, d string, bi buildInfo) (string, string, string) {
	if isPlaceholder(v, defaultVersion) && bi.mainVersion != "" && bi.mainVersion != develVersion {
		v = bi.mainVersion
	}
	if isPlaceholder(c, defaultCommitHash) && bi.vcsRevision != "" {
		c = bi.vcsRevision
	}
	if isPlaceholder(d, defaultBuildDate) && bi.vcsTime != "" {
		d = bi.vcsTime
	}

	if v == "" {
		v = defaultVersion
	}
	if c == "" {
		c = defaultCommitHash
	}
	if d == "" {
		d = defaultBuildDate
	}

	return v, c, d
}

func isPlaceholder(value, placeholder string) bool {
	return value == "" || value == placeholder
}

func shortRevision(revision string) string {
	const shortLen = 12
	if len(revision) > shortLen {
		return revision[:shortLen]
	}
	return revision
}

func formatVersionLine(v, c, d string) string {
	metadata := make([]string, 0, 2)
	if c != defaultCommitHash {
		metadata = append(metadata, "commit: "+c)
	}
	if d != defaultBuildDate {
		metadata = append(metadata, "built: "+d)
	}
	if len(metadata) == 0 {
		return fmt.Sprintf("codetok %s", v)
	}
	return fmt.Sprintf("codetok %s (%s)", v, strings.Join(metadata, ", "))
}
