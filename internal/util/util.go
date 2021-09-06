package util

// StrSliceContains runs a linear search over a string array
func StrSliceContains(array []string, element string) bool {
	for _, item := range array {
		if item == element {
			return true
		}
	}
	return false
}
