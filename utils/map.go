package utils

func ToKeys[K comparable, V any](m map[K]V) []K {
	var slice []K

	for k := range m {
		slice = append(slice, k)
	}

	return slice
}

func ToValues[K comparable, V any](m map[K]V) []V {
	var slice []V

	for _, v := range m {
		slice = append(slice, v)
	}

	return slice
}
