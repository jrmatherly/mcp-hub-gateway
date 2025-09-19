package version

import (
	"fmt"
	"runtime"
)

var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

func UserAgent() string {
	return "docker/mcp_gateway/v/" + Version
}

func FullVersion() string {
	return fmt.Sprintf(
		"Docker MCP Gateway\nVersion: %s\nCommit: %s\nBuilt: %s\nGo: %s\nOS/Arch: %s/%s",
		Version,
		GitCommit,
		BuildDate,
		runtime.Version(),
		runtime.GOOS,
		runtime.GOARCH,
	)
}
