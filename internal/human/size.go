package human

import (
	"fmt"
	"math"
	"strings"
)

var unitsSI = []string{"B", "kB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}

func SizeSI(bytes int64) string {
	var prefix string
	if bytes < 0 {
		prefix = "-"
		bytes = -bytes
	}

	if bytes < 1000 {
		return fmt.Sprintf("%v%v B", prefix, bytes)
	}

	const epsilon = 0.00000000000001

	size := float64(bytes)
	index := min(int(math.Log10(size)+epsilon)/3, len(unitsSI)-1)
	unit := unitsSI[index]

	size /= math.Pow(1000, float64(index))

	str := fmt.Sprintf("%.2f", size)
	if strings.Contains(str, ".") {
		str = strings.TrimRight(str, "0")
		str = strings.TrimSuffix(str, ".")
	}

	return fmt.Sprintf("%v%v %v", prefix, str, unit)
}

var unitsIEC = []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB", "ZiB", "YiB"}

func SizeIEC(bytes int64) string {
	var prefix string
	if bytes < 0 {
		prefix = "-"
		bytes = -bytes
	}

	if bytes < 1024 {
		return fmt.Sprintf("%v%v B", prefix, bytes)
	}

	size := float64(bytes)
	index := min(int(math.Log10(size))/3, len(unitsIEC)-1)
	unit := unitsIEC[index]

	size /= math.Pow(1024, float64(index))

	str := fmt.Sprintf("%.2f", size)
	if strings.Contains(str, ".") {
		str = strings.TrimRight(str, "0")
		str = strings.TrimSuffix(str, ".")
	}

	return fmt.Sprintf("%v%v %v", prefix, str, unit)
}
