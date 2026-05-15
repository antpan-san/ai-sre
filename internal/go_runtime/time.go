package go_runtime

import "time"

var now = func() time.Time { return time.Now() }
