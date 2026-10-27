package upgrade

import (
	_ "embed"
	"fmt"
	"runtime/debug"
	"strings"
)

//go:embed version.txt
var versionFile string

const RequiredPocketBaseVersion = "v0.41.0"

func AppVersion() string { return strings.TrimSpace(versionFile) }

func PocketBaseVersion() (string, error) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", fmt.Errorf("build metadata is unavailable")
	}
	for _, dependency := range info.Deps {
		if dependency.Path == "ganigeorgiev.com/pocketbase/pocketbase" {
			return dependency.Version, nil
		}
	}
	return "", fmt.Errorf("PocketBase dependency version is absent from build metadata")
}
