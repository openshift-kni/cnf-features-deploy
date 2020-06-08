package discovery

import (
	"os"
	"strconv"
)

// DiscoveryModeEnabled indicates whether test discovery mode is enabled.
func Enabled() bool {
	discoveryMode, _ := strconv.ParseBool(os.Getenv("DISCOVERY_MODE"))
	return discoveryMode
}
