// Package exec runs shell commands for dot and provides a scripted fake for tests.
package exec

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	osexec "os/exec"
	"strings"
)

// Result is the outcome of one shell command. A non-zero ExitCode is not an error.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Output returns stdout followed by stderr.
func (r Result) Output() string { return r.Stdout + r.Stderr }

// Runner executes a shell command string. When out is non-nil, combined
// output is streamed to it as well as captured in the Result.
type Runner interface {
	Run(ctx context.Context, command string, out io.Writer) (Result, error)
}

// Shell runs commands through /bin/zsh -c with a fixed environment.
type Shell struct {
	Env []string
}

// NewShell returns a Shell whose HOME is home and whose PATH starts with
// /opt/homebrew/bin. All other variables are inherited from the process.
func NewShell(home string) *Shell {
	env := []string{"HOME=" + home, "PATH=/opt/homebrew/bin:" + os.Getenv("PATH")}
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "HOME=") || strings.HasPrefix(kv, "PATH=") {
			continue
		}
		env = append(env, kv)
	}
	return &Shell{Env: env}
}

func (s *Shell) Run(ctx context.Context, command string, out io.Writer) (Result, error) {
	cmd := osexec.CommandContext(ctx, "/bin/zsh", "-c", command)
	cmd.Env = s.Env
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if out != nil {
		cmd.Stdout = io.MultiWriter(&stdout, out)
		cmd.Stderr = io.MultiWriter(&stderr, out)
	}
	err := cmd.Run()
	res := Result{Stdout: stdout.String(), Stderr: stderr.String()}
	var exitErr *osexec.ExitError
	if errors.As(err, &exitErr) {
		res.ExitCode = exitErr.ExitCode()
		return res, nil
	}
	return res, err
}
