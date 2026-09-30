package mwanachamacustody

import "time"

type Clock func() time.Time

func SystemClock() time.Time { return time.Now().UTC() }
