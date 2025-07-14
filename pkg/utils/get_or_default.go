package utils

func GetIntOrDefault(value int, defaultValue int) int {
	if value == 0 {
		return defaultValue
	}
	return value
}
