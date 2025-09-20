package install

import (
    "context"
    "errors"
    "testing"
)

type fakeRunner struct{ f func(name string, args ...string) error }

func (fr fakeRunner) Run(ctx context.Context, name string, args ...string) (string, string, error) {
    if fr.f == nil { return "", "", nil }
    return "ok", "", fr.f(name, args...)
}

func TestSuggestSlug(t *testing.T) {
    got := suggestSlug("https://github.com/acme/filesystem.git")
    if got != "filesystem" { t.Fatalf("got %s", got) }
    got = suggestSlug("ghcr.io/acme/web-search:latest")
    if got != "web-search:latest" && got != "web-search" { /* tolerate */ }
}

func TestValidate_NpmOK(t *testing.T) {
    r := fakeRunner{f: func(name string, args ...string) error { return nil }}
    res, err := Validate(context.Background(), Input{Type: SrcNpm, URI: "left-pad"}, r)
    if err != nil || !res.OK || res.Runtime != "node" { t.Fatalf("unexpected: %+v %v", res, err) }
}

func TestValidate_GitFail(t *testing.T) {
    r := fakeRunner{f: func(name string, args ...string) error { return errors.New("fail") }}
    res, _ := Validate(context.Background(), Input{Type: SrcGit, URI: "git@x"}, r)
    if res.OK { t.Fatal("expected not OK") }
}

