package mgr

import (
	"fmt"
	"time"

	"github.com/davidwalter0/backoff"
)

// HMSN hours, minutes, seconds, nanoseconds
func HMSN(d time.Duration) (h, m, s, n int64) {
	return int64(d.Hours()), int64(d.Minutes()), int64(d.Seconds()), int64(d.Nanoseconds())
}

// DurationString string of hours, minutes, seconds, nanoseconds
func DurationString(d time.Duration) string {
	h, m, s, n := HMSN(d)
	return fmt.Sprintf("%02d.%02d.%03.3d.%03.3d", h, m, (s*1000)/1000, ((n*1000)/1000)%1000)
}

// ConfigureBackoff using some normal steps
func ConfigureBackoff(step, maxStep, maxElapsedTime time.Duration, cancel chan struct{}) *backoff.ExponentialBackOff {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = step          //  1 * time.second
	b.MaxInterval = maxStep           //  5 * time.Minute
	b.MaxElapsedTime = maxElapsedTime // 15 * time.Minute
	// Store cancel channel in the context of the caller
	return b
}
