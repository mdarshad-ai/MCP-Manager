package supervisor

import (
    "testing"
    "time"
)

func TestRestartsInLast(t *testing.T) {
    // Create a minimal supervisor for testing
    sup := &Supervisor{}
    ps := &ProcState{}
    now := time.Now()
    ps.RestartsAt = []time.Time{now.Add(-11*time.Minute), now.Add(-9*time.Minute), now.Add(-1*time.Minute)}
    n := sup.restartsInLast(ps, 10*time.Minute)
    if n != 2 { t.Fatalf("want 2 got %d", n) }
    // Old entries should be trimmed
    if len(ps.RestartsAt) != 2 { t.Fatalf("kept %d", len(ps.RestartsAt)) }
}

func TestDeriveHTTPURL(t *testing.T) {
    u := deriveHTTPURL([]string{"--port=8080"}, nil)
    if u != "http://127.0.0.1:8080" { t.Fatalf("got %s", u) }
    u = deriveHTTPURL([]string{"-p", "3000"}, nil)
    if u != "http://127.0.0.1:3000" { t.Fatalf("got %s", u) }
    u = deriveHTTPURL(nil, map[string]string{"HEALTH_HTTP_URL": "http://127.0.0.1:9090/health"})
    if u != "http://127.0.0.1:9090/health" { t.Fatalf("got %s", u) }
}
