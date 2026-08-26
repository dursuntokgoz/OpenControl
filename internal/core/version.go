// Package core contains shared domain types for ServerPanel.
package core

import "fmt"

// BuildInfo describes the build identity of ServerPanel binaries.
type BuildInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"buildDate"`
}

// String renders the build identity in a compact human-readable form.
func (b BuildInfo) String() string {
	return fmt.Sprintf("serverpanel %s (commit=%s built=%s)", b.Version, b.Commit, b.BuildDate)
}

// Build information injected via -ldflags at link time.
var (
	buildVersion   = "dev"
	buildCommit    = "unknown"
	buildBuildDate = "unknown"
)

// CurrentBuild returns the build identity of the running binary.
func CurrentBuild() BuildInfo {
	return BuildInfo{
		Version:   buildVersion,
		Commit:    buildCommit,
		BuildDate: buildBuildDate,
	}
}
