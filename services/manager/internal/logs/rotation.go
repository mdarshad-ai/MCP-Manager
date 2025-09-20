package logs

// PlanRotation determines per-file and global trimming according to hybrid policy.
// perFileCap and globalCap are bytes. sizes is the current size of each log file.
// It returns a slice of bytesToTrim, per file index.
func PlanRotation(sizes []int64, perFileCap, globalCap int64) []int64 {
    trim := make([]int64, len(sizes))
    // First pass: enforce per-file caps
    var total int64
    for i, s := range sizes {
        if s > perFileCap {
            trim[i] = s - perFileCap
            s = perFileCap
        }
        total += s
    }
    // Second pass: if total exceeds global cap, trim oldest across files (here: simple even trim)
    if total > globalCap {
        excess := total - globalCap
        // naive strategy: distribute trimming proportionally by size (could be replaced by age-aware)
        var sum int64
        for _, s := range sizes {
            if s > 0 {
                sum += s
            }
        }
        if sum > 0 {
            for i, s := range sizes {
                if s <= 0 {
                    continue
                }
                // proportional share
                share := excess * s / sum
                trim[i] += share
            }
        }
    }
    return trim
}

