package cmd

import "testing"

func TestResolveVersionInfoPrefersLdflags(t *testing.T) {
	version, commit, built := resolveVersionInfo(
		"v1.2.3",
		"abc123",
		"2026-02-18T00:00:00Z",
		buildInfo{
			mainVersion: "v9.9.9",
			vcsRevision: "def456",
			vcsTime:     "2026-02-17T00:00:00Z",
		},
	)

	if version != "v1.2.3" {
		t.Fatalf("expected version from ldflags, got %q", version)
	}
	if commit != "abc123" {
		t.Fatalf("expected commit from ldflags, got %q", commit)
	}
	if built != "2026-02-18T00:00:00Z" {
		t.Fatalf("expected build date from ldflags, got %q", built)
	}
}

func TestResolveVersionInfoFallbackToBuildInfo(t *testing.T) {
	version, commit, built := resolveVersionInfo(
		defaultVersion,
		defaultCommitHash,
		defaultBuildDate,
		buildInfo{
			mainVersion: "v0.1.0",
			vcsRevision: "abcdef123456",
			vcsTime:     "2026-02-18T01:02:03Z",
		},
	)

	if version != "v0.1.0" {
		t.Fatalf("expected version from build info, got %q", version)
	}
	if commit != "abcdef123456" {
		t.Fatalf("expected commit from build info, got %q", commit)
	}
	if built != "2026-02-18T01:02:03Z" {
		t.Fatalf("expected build date from build info, got %q", built)
	}
}

func TestResolveVersionInfoDevelFallback(t *testing.T) {
	version, commit, built := resolveVersionInfo(
		defaultVersion,
		defaultCommitHash,
		defaultBuildDate,
		buildInfo{
			mainVersion: develVersion,
		},
	)

	if version != defaultVersion {
		t.Fatalf("expected default version for devel builds, got %q", version)
	}
	if commit != defaultCommitHash {
		t.Fatalf("expected default commit for devel builds, got %q", commit)
	}
	if built != defaultBuildDate {
		t.Fatalf("expected default build date for devel builds, got %q", built)
	}
}

func TestShortRevision(t *testing.T) {
	if got := shortRevision("12345678901234567890"); got != "123456789012" {
		t.Fatalf("expected shortened revision, got %q", got)
	}
	if got := shortRevision("abc123"); got != "abc123" {
		t.Fatalf("expected unchanged short revision, got %q", got)
	}
}

func TestFormatVersionLine(t *testing.T) {
	tests := []struct {
		name string
		v    string
		c    string
		d    string
		want string
	}{
		{
			name: "version only",
			v:    "v0.1.0",
			c:    defaultCommitHash,
			d:    defaultBuildDate,
			want: "codetok v0.1.0",
		},
		{
			name: "version and commit",
			v:    "v0.1.0",
			c:    "abc123",
			d:    defaultBuildDate,
			want: "codetok v0.1.0 (commit: abc123)",
		},
		{
			name: "version and metadata",
			v:    "v0.1.0",
			c:    "abc123",
			d:    "2026-02-18T00:00:00Z",
			want: "codetok v0.1.0 (commit: abc123, built: 2026-02-18T00:00:00Z)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatVersionLine(tc.v, tc.c, tc.d); got != tc.want {
				t.Fatalf("unexpected version line: got %q want %q", got, tc.want)
			}
		})
	}
}
