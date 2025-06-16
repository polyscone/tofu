package human

import (
	"fmt"
	"strings"
)

var unitsSI = []string{"B", "kB", "MB", "GB", "TB", "PB"}

func SizeSI(bytes int64) string {
	var prefix string
	if bytes < 0 {
		prefix = "-"
		bytes = -bytes
	}

	if bytes < 1000 {
		return fmt.Sprintf("%v%v B", prefix, bytes)
	}

	var i int
	size := float64(bytes)
	for size >= 1000 && i < len(unitsSI)-1 {
		size /= 1000
		i++
	}
	unit := unitsSI[i]

	// We use 10 decimal places here because we want to lose
	// as little data as possible whilst also being human readable
	// and avoiding floating point artifacts that might show up
	// with a higher precision like 15 decimal places
	str := fmt.Sprintf("%.10f", size)
	if strings.Contains(str, ".") {
		str = strings.TrimRight(str, "0")
		str = strings.TrimSuffix(str, ".")
	}

	return fmt.Sprintf("%v%v %v", prefix, str, unit)
}

var unitsIEC = []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB"}

func SizeIEC(bytes int64) string {
	var prefix string
	if bytes < 0 {
		prefix = "-"
		bytes = -bytes
	}

	if bytes < 1024 {
		return fmt.Sprintf("%v%v B", prefix, bytes)
	}

	var i int
	size := float64(bytes)
	for size >= 1024 && i < len(unitsIEC)-1 {
		size /= 1024
		i++
	}
	unit := unitsIEC[i]

	// We use 10 decimal places here because we want to lose
	// as little data as possible whilst also being human readable
	// and avoiding floating point artifacts that might show up
	// with a higher precision like 15 decimal places
	str := fmt.Sprintf("%.10f", size)
	if strings.Contains(str, ".") {
		str = strings.TrimRight(str, "0")
		str = strings.TrimSuffix(str, ".")
	}

	return fmt.Sprintf("%v%v %v", prefix, str, unit)
}
