package main

import "fmt"

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func displayVersion() string {
	if version == "" {
		return "dev"
	}
	return version
}

func buildInfo() string {
	return fmt.Sprintf("Version: %s  Commit: %s  Built: %s", displayVersion(), commit, buildDate)
}
