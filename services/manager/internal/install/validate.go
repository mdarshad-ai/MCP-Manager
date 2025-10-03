package install

import (
    "context"
    "errors"
    "fmt"
    "net/url"
    "path"
    "regexp"
    "strings"
)

type SourceType string

const (
    SrcGit    SourceType = "git"
    SrcNpm    SourceType = "npm"
    SrcPip    SourceType = "pip"
    SrcDocker SourceType = "docker-image"
    SrcCompose SourceType = "docker-compose"
)

type Input struct {
    Type SourceType `json:"type"`
    URI  string     `json:"uri"`
}

type Result struct {
    OK       bool     `json:"ok"`
    Problems []string `json:"problems"`
    Slug     string   `json:"slug"`
    Runtime  string   `json:"runtime"` // node|python|docker|binary
    Manager  string   `json:"manager"` // npm|pnpm|pip|uv|pipx
}

var slugRE = regexp.MustCompile(`[^a-z0-9-]+`)

func slugify(s string) string {
    s = strings.ToLower(s)
    s = strings.TrimSuffix(s, ".git")
    s = strings.TrimSuffix(s, ".tar.gz")
    s = strings.Trim(s, "/ ")
    s = strings.ReplaceAll(s, "_", "-")
    s = slugRE.ReplaceAllString(s, "-")
    s = strings.Trim(s, "-")
    if s == "" { s = "server" }
    return s
}

func suggestSlug(uri string) string {
    if strings.Contains(uri, "://") {
        if u, err := url.Parse(uri); err == nil {
            base := path.Base(u.Path)
            return slugify(base)
        }
    }
    parts := strings.Split(uri, "/")
    return slugify(parts[len(parts)-1])
}

func Validate(ctx context.Context, in Input, r Runner) (Result, error) {
    if r == nil { r = ExecRunner{} }
    res := Result{OK: true, Slug: suggestSlug(in.URI)}
    switch in.Type {
    case SrcGit:
        if _, _, err := r.Run(ctx, "git", "ls-remote", in.URI); err != nil {
            res.OK = false; res.Problems = append(res.Problems, fmt.Sprintf("git unreachable: %v", err))
        }
    case SrcNpm:
        if _, _, err := r.Run(ctx, "npm", "view", in.URI, "version"); err != nil {
            res.OK = false; res.Problems = append(res.Problems, fmt.Sprintf("npm not found or package missing: %v", err))
        } else { res.Runtime = "node"; res.Manager = "npm" }
    case SrcPip:
        if _, _, err := r.Run(ctx, "pip", "index", "versions", in.URI); err != nil {
            res.OK = false; res.Problems = append(res.Problems, fmt.Sprintf("pip package not found: %v", err))
        } else { res.Runtime = "python"; res.Manager = "pip" }
    case SrcDocker:
        if _, _, err := r.Run(ctx, "docker", "image", "inspect", in.URI); err != nil {
            res.OK = false; res.Problems = append(res.Problems, fmt.Sprintf("docker image missing: %v", err))
        } else { res.Runtime = "docker" }
    case SrcCompose:
        if _, _, err := r.Run(ctx, "docker", "compose", "config", "-q"); err != nil {
            res.OK = false; res.Problems = append(res.Problems, fmt.Sprintf("docker compose not available: %v", err))
        } else { res.Runtime = "docker" }
    default:
        return res, errors.New("unsupported source type")
    }
    // Disk space sanity: require at least 500MB free
    if ok, err := hasDiskSpace(500 * 1024 * 1024); err == nil && !ok {
        res.OK = false; res.Problems = append(res.Problems, "insufficient disk space (<500MB)")
    }
    return res, nil
}

func hasDiskSpace(minBytes uint64) (bool, error) {
    // For Windows, we'll skip disk space checking for now
    // This could be implemented using windows.GetDiskFreeSpaceEx
    return true, nil
}

