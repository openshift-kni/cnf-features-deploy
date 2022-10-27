package event

import (
	"os"
	"strconv"
)

// Enabled event if ptp event is required
func Enable() bool {
	eventMode, _ := strconv.ParseBool(os.Getenv("ENABLE_PTP_EVENT"))
	return eventMode
}
