package global

import (
	"time"
)

// SleepBackoffSeconds power of 2 delay
func SleepBackoffSeconds(i uint) time.Duration {
	return time.Second * (1 << i)
}
