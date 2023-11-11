package human

import (
	"fmt"
	"math"
	"strings"
	"time"
)

func Duration(d time.Duration) string {
	var parts []string

	hours := int(d.Hours())
	if hours == 1 {
		parts = append(parts, fmt.Sprintf("%v hour", hours))
	} else if hours > 1 {
		parts = append(parts, fmt.Sprintf("%v hours", hours))
	}

	minutes := int(math.Mod(d.Minutes(), 60))
	if minutes == 1 {
		parts = append(parts, fmt.Sprintf("%v minute", minutes))
	} else if minutes > 1 {
		parts = append(parts, fmt.Sprintf("%v minutes", minutes))
	}

	seconds := int(math.Mod(d.Seconds(), 60))
	if seconds == 1 {
		parts = append(parts, fmt.Sprintf("%v second", seconds))
	} else if seconds > 1 {
		parts = append(parts, fmt.Sprintf("%v seconds", seconds))
	}

	if len(parts) == 0 {
		return "0 seconds"
	}

	return AndList(parts)
}

func DurationPrecise(d time.Duration) string {
	var parts []string

	hours := int(d.Hours())
	if hours >= 1 {
		parts = append(parts, fmt.Sprintf("%vh", hours))
	}

	minutes := int(math.Mod(d.Minutes(), 60))
	if minutes >= 1 {
		parts = append(parts, fmt.Sprintf("%vm", minutes))
	}

	seconds := int(math.Mod(d.Seconds(), 60))
	if seconds >= 1 {
		parts = append(parts, fmt.Sprintf("%vs", seconds))
	}

	milliseconds := d.Milliseconds() % 1000
	if milliseconds >= 1 {
		parts = append(parts, fmt.Sprintf("%vms", milliseconds))
	}

	microseconds := d.Microseconds() % 1000
	if microseconds >= 1 {
		parts = append(parts, fmt.Sprintf("%vµs", microseconds))
	}

	if len(parts) == 0 {
		return "0s"
	}

	return strings.Join(parts, " ")
}
