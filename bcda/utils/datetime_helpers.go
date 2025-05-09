package utils

import "time"

// Get current Performance Year
func GetPY() int {
	return time.Now().Year() % 100
}
