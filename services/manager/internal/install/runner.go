package install

import (
    "bytes"
    "context"
    "os/exec"
)

type Runner interface {
    Run(ctx context.Context, name string, args ...string) (stdout string, stderr string, err error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, name string, args ...string) (string, string, error) {
    cmd := exec.CommandContext(ctx, name, args...)
    var out, errb bytes.Buffer
    cmd.Stdout = &out
    cmd.Stderr = &errb
    err := cmd.Run()
    return out.String(), errb.String(), err
}

