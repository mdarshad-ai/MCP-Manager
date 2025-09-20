package health

type Status string

const (
    Ready    Status = "ready"
    Degraded Status = "degraded"
    Down     Status = "down"
)

type ProbeInput struct {
    ProcessRunning   bool
    MissedPings      int
    LastPingMs       int
    RestartsLast10m  int
}

// Evaluate returns Ready, Degraded, or Down based on inputs and v0 policy.
func Evaluate(in ProbeInput) Status {
    if !in.ProcessRunning {
        return Down
    }
    degraded := false
    if in.MissedPings >= 1 && in.MissedPings <= 2 {
        degraded = true
    }
    if in.LastPingMs > 1000 {
        degraded = true
    }
    if in.RestartsLast10m >= 1 {
        degraded = true
    }
    if degraded {
        return Degraded
    }
    return Ready
}

