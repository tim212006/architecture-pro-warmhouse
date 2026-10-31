package clock

import "time"

// System returns the real system time.
type System struct{}

// NewSystem returns a system clock.
func NewSystem() *System {
	return &System{}
}

// Now returns the current time.
func (c *System) Now() time.Time {
	return time.Now()
}
