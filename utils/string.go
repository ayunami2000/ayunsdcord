package utils

import "strings"

// https://stackoverflow.com/a/59955447/6917520
func TruncateText(s string, max int) string {
	if max > len(s) {
		return s
	}
	return s[:strings.LastIndexAny(s[:max], " .,:;-")]
}

func StringOrNone(s string) string {
	if s == "" {
		return "None"
	}

	return s
}
