package supervisor

import (
    "testing"
    "time"
)

func TestBackoff(t *testing.T) {
    base := 1 * time.Second
    max := 10 * time.Second
    cases := []struct{ restarts int; want time.Duration }{
        {0, 1 * time.Second},
        {1, 1 * time.Second},
        {2, 2 * time.Second},
        {3, 4 * time.Second},
        {5, 10 * time.Second},
        {6, 10 * time.Second},
    }
    for _, c := range cases {
        if got := Backoff(c.restarts, base, max); got != c.want {
            t.Fatalf("restarts=%d got %s want %s", c.restarts, got, c.want)
        }
    }
}

