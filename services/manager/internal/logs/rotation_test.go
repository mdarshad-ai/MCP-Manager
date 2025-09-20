package logs

import "testing"

func TestPlanRotation_PerFile(t *testing.T) {
    sizes := []int64{200, 50}
    trim := PlanRotation(sizes, 100, 1000)
    if trim[0] != 100 || trim[1] != 0 {
        t.Fatalf("unexpected trim: %+v", trim)
    }
}

func TestPlanRotation_Global(t *testing.T) {
    sizes := []int64{100, 100, 100}
    trim := PlanRotation(sizes, 200, 250)
    var total int64
    for _, v := range trim {
        total += v
    }
    if total != 50 {
        t.Fatalf("expected trim 50 got %d", total)
    }
}

