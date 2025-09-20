package logs

import (
    "io"
    "os"
    "path/filepath"
)

// ApplyRotation trims log files by rewriting from the tail to respect trim bytes.
// For each file in paths, if trim[i] > 0, it keeps the last size-trim[i] bytes.
func ApplyRotation(paths []string, trim []int64) error {
    for i, p := range paths {
        if trim[i] <= 0 { continue }
        fi, err := os.Stat(p)
        if err != nil { continue }
        keep := fi.Size() - trim[i]
        if keep <= 0 { _ = os.Truncate(p, 0); continue }
        f, err := os.Open(p)
        if err != nil { continue }
        defer f.Close()
        if _, err := f.Seek(fi.Size()-keep, io.SeekStart); err != nil { continue }
        tmp := p + ".tmp"
        tf, err := os.Create(tmp)
        if err != nil { continue }
        if _, err := io.Copy(tf, f); err != nil { tf.Close(); _ = os.Remove(tmp); continue }
        tf.Close()
        _ = os.Rename(tmp, p)
    }
    return nil
}

// ListLogFiles returns absolute paths and sizes for files in dir with .log extension.
func ListLogFiles(dir string) ([]string, []int64, error) {
    entries, err := os.ReadDir(dir)
    if err != nil { return nil, nil, err }
    var pathsList []string
    var sizes []int64
    for _, e := range entries {
        if e.IsDir() { continue }
        if filepath.Ext(e.Name()) != ".log" { continue }
        p := filepath.Join(dir, e.Name())
        fi, err := os.Stat(p); if err != nil { continue }
        pathsList = append(pathsList, p)
        sizes = append(sizes, fi.Size())
    }
    return pathsList, sizes, nil
}

