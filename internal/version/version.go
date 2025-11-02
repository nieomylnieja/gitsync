package version

import (
	"runtime/debug"
	"strings"
)

// Set during build time.
var (
	BuildGitRevision string
	BuildGitBranch   string
	BuildVersion     string
)

func GetVersion() string {
	version := BuildVersion
	if version == "" {
		version = getRuntimeVersion()
	}
	return strings.TrimSpace(version)
}

func getRuntimeVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "(devel)" {
		return "0.0.0"
	}
	return strings.TrimPrefix(info.Main.Version, "v")
}
