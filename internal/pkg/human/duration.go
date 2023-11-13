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

func DurationStat(d time.Duration) string {
	var parts []string

	hours := int(d.Hours())
	if hours >= 1 {
		parts = append(parts, fmt.Sprintf("%v h", hours))
	}

	minutes := int(math.Mod(d.Minutes(), 60))
	if minutes >= 1 {
		parts = append(parts, fmt.Sprintf("%v m", minutes))
	}

	seconds := int(math.Mod(d.Seconds(), 60))
	if seconds >= 1 {
		parts = append(parts, fmt.Sprintf("%v s", seconds))
	}

	if len(parts) <= 1 {
		milliseconds := d.Milliseconds() % 1000
		if milliseconds >= 1 {
			parts = append(parts, fmt.Sprintf("%v ms", milliseconds))
		}
	}

	if len(parts) <= 1 {
		microseconds := d.Microseconds() % 1000
		if microseconds >= 1 {
			parts = append(parts, fmt.Sprintf("%v µs", microseconds))
		}
	}

	if len(parts) <= 1 {
		nanoseconds := d.Nanoseconds() % 1000
		if nanoseconds >= 1 {
			parts = append(parts, fmt.Sprintf("%v ns", nanoseconds))
		}
	}

	if len(parts) == 0 {
		return "0 s"
	}

	return strings.Join(parts, " ")
}
