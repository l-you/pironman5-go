package status

import "fmt"

func FormatBytes(value uint64, fixedUnit ...string) (float64, string) {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	unitIndex := 0
	if len(fixedUnit) > 0 {
		for i, unit := range units {
			if unit == fixedUnit[0] {
				unitIndex = i
				break
			}
		}
		divisor := float64(uint64(1) << (10 * unitIndex))
		return float64(value) / divisor, units[unitIndex]
	}
	v := float64(value)
	for unitIndex < len(units)-1 && v >= 1024 {
		v /= 1024
		unitIndex++
	}
	return v, units[unitIndex]
}

func FormatBytesString(value uint64, fixedUnit ...string) string {
	v, unit := FormatBytes(value, fixedUnit...)
	if unit == "B" {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%.1f", v)
}
