package stringhelper

import (
	"strconv"
	"strings"
)

// StringInSlice Checks a slice for a given string.
func StringInSlice[T ~string](s []T, str T, contains bool) bool {
	for _, v := range s {
		if !contains {
			if strings.TrimSpace(string(v)) == string(str) {
				return true
			}
		} else {
			if strings.Contains(strings.TrimSpace(string(v)), string(str)) {
				return true
			}
		}
	}
	return false
}

// IsNumber Returns true if the string represent a number
func IsNumber(s string) bool {
	if _, err := strconv.Atoi(s); err != nil {
		return false
	}
	return true
}
