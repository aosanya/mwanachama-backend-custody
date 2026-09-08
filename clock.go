package mwanachamacustody

import "time"

// Clock is a testable time source, used by every store constructor to stamp
// a row's OccurredAt/ReadAt/RequestedAt/etc. in Go rather than relying on a
// database DEFAULT — mirrors mwanachama-backend-comm's Clock.
type Clock func() time.Time

// SystemClock is the default clock.
func SystemClock() time.Time { return time.Now().UTC() }
