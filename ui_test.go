package main

import "testing"

func TestScaledDebugTextSize(t *testing.T) {
	width, height := scaledDebugTextSize("AB\nC", 2)

	if width != 24 {
		t.Errorf("width = %d, want 24", width)
	}
	if height != 64 {
		t.Errorf("height = %d, want 64", height)
	}
}

func TestBuildInfoMultilineSeparatesMetadata(t *testing.T) {
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

	got := buildInfoMultiline()
	want := "VERSION: 1.2.3\nCOMMIT: abc123\nBUILT: 2026-05-23T00:00:00Z"
	if got != want {
		t.Fatalf("buildInfoMultiline() = %q, want %q", got, want)
	}
}
