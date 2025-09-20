package health

import "testing"

func TestEvaluate(t *testing.T) {
    cases := []struct {
        in   ProbeInput
        want Status
    }{
        {ProbeInput{ProcessRunning: false}, Down},
        {ProbeInput{ProcessRunning: true, MissedPings: 0, LastPingMs: 100, RestartsLast10m: 0}, Ready},
        {ProbeInput{ProcessRunning: true, MissedPings: 1}, Degraded},
        {ProbeInput{ProcessRunning: true, LastPingMs: 1500}, Degraded},
        {ProbeInput{ProcessRunning: true, RestartsLast10m: 1}, Degraded},
    }
    for i, c := range cases {
        if got := Evaluate(c.in); got != c.want {
            t.Fatalf("case %d: got %s want %s", i, got, c.want)
        }
    }
}

