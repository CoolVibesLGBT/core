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

func StrToInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

func StrToInt(s string) (int, error) {
	return strconv.Atoi(s)
}
