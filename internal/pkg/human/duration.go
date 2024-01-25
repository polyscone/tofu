package human

import (
	"fmt"
	"math"
	"strings"
	"time"
)

func Duration(d time.Duration) string {
	var parts []string

	// We assume a 365 day year
	// Even though it may not necessarily be correct due to leap years etc.
	// it's a close enough approximation for a user-friendly display string
	years := int(d.Hours() / 24 / 365)
	if years == 1 {
		parts = append(parts, fmt.Sprintf("%v year", years))
	} else if years > 1 {
		parts = append(parts, fmt.Sprintf("%v years", years))
	}

	// We assume a 24 hour day
	// Even though it may not necessarily be correct due to daylight savings etc.
	// it's a close enough approximation for a user-friendly display string
	days := int(math.Mod(d.Hours()/24, 365))
	if days == 1 {
		parts = append(parts, fmt.Sprintf("%v day", days))
	} else if days > 1 {
		parts = append(parts, fmt.Sprintf("%v days", days))
	}

	// If we have a duration in years then we don't need a higher resolution
	// description than years and days
	if years > 0 {
		return AndList(parts)
	}

	// Assuming a good enough approximation of a 24 hour day
	hours := int(math.Mod(d.Hours(), 24))
	if hours == 1 {
		parts = append(parts, fmt.Sprintf("%v hour", hours))
	} else if hours > 1 {
		parts = append(parts, fmt.Sprintf("%v hours", hours))
	}

	// If we have a duration in days then we don't need a higher resolution
	// description than days and hours
	if days > 0 {
		return AndList(parts)
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
