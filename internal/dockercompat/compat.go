package dockercompat

import (
	"fmt"
	"os"
	"regexp"

	sdkClient "github.com/moby/moby/client"
	"github.com/moby/moby/client/pkg/versions"
)

const DefaultAPIVersion = "1.42"

var apiVersionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+$`)

func ConfiguredAPIVersion() string {
	if version := os.Getenv("DOCKER_API_VERSION"); version != "" {
		return version
	}
	return DefaultAPIVersion
}

func Supports(requiredVersion string) bool {
	return !versions.LessThan(ConfiguredAPIVersion(), requiredVersion)
}

func Validate(version string) error {
	if !apiVersionPattern.MatchString(version) {
		return fmt.Errorf("invalid Docker API version %q: expected MAJOR.MINOR", version)
	}
	if versions.LessThan(version, DefaultAPIVersion) {
		return fmt.Errorf("docker API version %s is unsupported: watchtower requires %s or later", version, DefaultAPIVersion)
	}
	if versions.GreaterThan(version, sdkClient.MaxAPIVersion) {
		return fmt.Errorf("docker API version %s is unsupported by the pinned Docker SDK: maximum is %s", version, sdkClient.MaxAPIVersion)
	}
	return nil
}
