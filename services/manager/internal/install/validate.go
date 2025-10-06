package install

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"
)

type SourceType string

const (
	SrcGit     SourceType = "git"
	SrcNpm     SourceType = "npm"
	SrcPip     SourceType = "pip"
	SrcDocker  SourceType = "docker-image"
	SrcCompose SourceType = "docker-compose"
)

var slugRE = regexp.MustCompile(`[^a-z0-9-]+`)

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.TrimSuffix(s, ".git")
	s = strings.TrimSuffix(s, ".tar.gz")
	s = strings.Trim(s, "/ ")
	s = strings.ReplaceAll(s, "_", "-")
	s = slugRE.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "server"
	}
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

func hasDiskSpace(minBytes uint64) (bool, error) {
	// For Windows, we'll skip disk space checking for now
	// This could be implemented using windows.GetDiskFreeSpaceEx
	return true, nil
}

func logf(dst Logger, format string, a ...any) {
	dst.Log(time.Now().Format("15:04:05 ") + fmt.Sprintf(format, a...))
}
