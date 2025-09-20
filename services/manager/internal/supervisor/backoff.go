package supervisor

import "time"

// Backoff returns next delay with exponential backoff capped at max.
func Backoff(restarts int, base, max time.Duration) time.Duration {
    if restarts <= 0 {
        return base
    }
    d := base << (restarts - 1)
    if d > max {
        return max
    }
    return d
}

