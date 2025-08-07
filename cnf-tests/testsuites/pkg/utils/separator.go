package utils

import "strings"

func GetSeparator(data string) string {
	separator := "\r\n"
	if !strings.Contains(data, separator) {
		separator = "\n"
	}
	return separator
}
