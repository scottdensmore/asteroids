package game

import "testing"

func TestVersionFallsBackToDev(t *testing.T) {
	original := version
	t.Cleanup(func() {
		version = original
	})

	version = ""

	if got := Version(); got != "dev" {
		t.Fatalf("Version() = %q, want %q", got, "dev")
	}
}

func TestBuildInfoIncludesVersionMetadata(t *testing.T) {
	originalVersion := version
	originalCommit := commit
	originalBuildDate := buildDate
	t.Cleanup(func() {
		version = originalVersion
		commit = originalCommit
		buildDate = originalBuildDate
	})

	version = "1.2.3"
	commit = "abc123"
	buildDate = "2026-05-23T00:00:00Z"

	got := buildInfo()
	want := "Version: 1.2.3  Commit: abc123  Built: 2026-05-23T00:00:00Z"
	if got != want {
		t.Fatalf("buildInfo() = %q, want %q", got, want)
	}
}
