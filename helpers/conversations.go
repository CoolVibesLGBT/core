package helpers

import "strconv"

func ParseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

func DefaultIfEmpty(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
