package version

import "fmt"

// These are set at build time via ldflags.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

// String returns a formatted version string.
func String() string {
	return fmt.Sprintf("neoncast %s (commit: %s, built: %s)", Version, Commit, BuildTime)
}

// Short returns just the version number.
func Short() string {
	return Version
}
