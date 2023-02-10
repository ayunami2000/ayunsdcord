package utils

import (
	"fmt"
)

func Contains[V comparable](arr []V, e V) bool {
	for _, v := range arr {
		if v == e {
			return true
		}
	}

	return false
}

func ToStringSlice[V comparable](arr []V) (out []string) {
	for _, v := range arr {
		out = append(out, fmt.Sprint(v))
	}

	return
}
