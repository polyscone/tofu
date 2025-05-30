package size

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	Byte = 1

	Kilobyte = 1000 * Byte
	Megabyte = 1000 * Kilobyte
	Gigabyte = 1000 * Megabyte
	Terabyte = 1000 * Gigabyte
	Petabyte = 1000 * Terabyte

	Kibibyte = 1024 * Byte
	Mebibyte = 1024 * Kibibyte
	Gibibyte = 1024 * Mebibyte
	Tebibyte = 1024 * Gibibyte
	Pebibyte = 1024 * Tebibyte
)

// ParseBytes converts the string to an int representing the number of bytes
func ParseBytes(s string) (int, error) {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, ",", "")

	parts := strings.Fields(s)
	if len(parts) == 0 {
		return 0, nil
	}

	size, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, err
	}

	if len(parts) > 1 {
		switch suffix := parts[1]; suffix {
		case "b", "byte", "bytes":
			// Do nothing

		case "kb", "kilobyte", "kilobytes":
			size *= Kilobyte

		case "mb", "megabyte", "megabytes":
			size *= Megabyte

		case "gb", "gigabyte", "gigabytes":
			size *= Gigabyte

		case "tb", "terabyte", "terabytes":
			size *= Terabyte

		case "pb", "petabyte", "petabytes":
			size *= Petabyte

		case "kib", "kibibyte", "kibibytes":
			size *= Kibibyte

		case "mib", "mebibyte", "mebibytes":
			size *= Mebibyte

		case "gib", "gibibyte", "gibibytes":
			size *= Gibibyte

		case "tib", "tebibyte", "tebibytes":
			size *= Tebibyte

		case "pib", "pebibyte", "pebibytes":
			size *= Pebibyte

		default:
			return 0, fmt.Errorf("unknown suffix %q", suffix)
		}
	}

	return int(size), nil
}
