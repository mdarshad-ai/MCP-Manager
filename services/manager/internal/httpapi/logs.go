package httpapi

import (
    "bufio"
    "bytes"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "strconv"
    "strings"

    "mcp/manager/internal/paths"
)

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
    // GET /v1/logs/{slug}?tail=200
    if r.Method != http.MethodGet { w.WriteHeader(http.StatusMethodNotAllowed); return }
    parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
    if len(parts) < 3 { w.WriteHeader(http.StatusBadRequest); return }
    slug := parts[2]
    tailN := 200
    if v := r.URL.Query().Get("tail"); v != "" {
        if n, err := strconv.Atoi(v); err == nil { tailN = n }
    }
    dir, err := paths.LogsDir(); if err != nil { w.WriteHeader(http.StatusInternalServerError); return }
    file := filepath.Join(dir, slug+".log")
    lines, _ := tailLines(file, tailN)
    w.Header().Set("Content-Type", "text/plain; charset=utf-8")
    for _, l := range lines { _, _ = w.Write([]byte(l+"\n")) }
}

func tailLines(path string, n int) ([]string, error) {
    f, err := os.Open(path); if err != nil { return nil, err }
    defer f.Close()
    const chunk = 32 * 1024
    fi, err := f.Stat(); if err != nil { return nil, err }
    size := fi.Size()
    var lines []string
    var remainder []byte
    off := size
    for off > 0 && len(lines) <= n {
        toRead := chunk
        if int64(toRead) > off { toRead = int(off) }
        off -= int64(toRead)
        buf := make([]byte, toRead)
        if _, err := f.ReadAt(buf, off); err != nil && err != io.EOF { break }
        // split and prepend remainder
        seg := append(buf, remainder...)
        parts := splitLines(seg)
        // last part may be partial start; save as remainder for next chunk
        if len(parts) > 0 {
            remainder = []byte(parts[0])
            // prepend other parts (from end backwards excluding first partial)
            for i := len(parts)-1; i >= 1; i-- {
                lines = append(lines, parts[i])
                if len(lines) >= n { break }
            }
        }
    }
    // reverse collected lines (currently newest-first)
    for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
        lines[i], lines[j] = lines[j], lines[i]
    }
    if len(lines) > n { lines = lines[len(lines)-n:] }
    return lines, nil
}

func splitLines(b []byte) []string {
    s := bufio.NewScanner(bytes.NewReader(b))
    s.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
    var out []string
    for s.Scan() { out = append(out, s.Text()) }
    return out
}
