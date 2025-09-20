package health

import (
    "time"
)

// Loop updates status fields on a Proc-like object.
type ProbeTarget interface {
    SetStatus(Status)
    GetProbeInput() ProbeInput
}

// Run evaluates ProbeInput at interval and sets status accordingly.
func Run(t Target, interval time.Duration, stop <-chan struct{}) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()
    for {
        select {
        case <-stop:
            return
        case <-ticker.C:
            in := t.GetProbeInput()
            st := Evaluate(in)
            t.SetStatus(st)
        }
    }
}

// Keep API stable even if we rename later.
type Target = ProbeTarget

