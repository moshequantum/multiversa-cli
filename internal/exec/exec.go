// Package exec wraps os/exec with a small surface tailored to the wizard:
// run a command, collect combined output, return a Result. Prereq checks
// against PATH.
package exec

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"
)

// Result captures the outcome of a single command invocation.
type Result struct {
	Cmd      string   // pretty-printed command line
	Output   []string // combined stdout+stderr, one entry per line
	Err      error    // non-nil if the command failed (non-zero exit or run error)
	Duration time.Duration
}

// Run executes name with args, captures combined output (stdout+stderr), and
// returns a Result. The command runs with a sane default timeout (5 minutes).
// For long-running installs that exceed this, call RunWithContext.
func Run(name string, args ...string) Result {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	return RunWithContext(ctx, name, args...)
}

// RunWithContext is Run with a caller-supplied context (cancellation,
// custom timeout).
func RunWithContext(ctx context.Context, name string, args ...string) Result {
	c := exec.CommandContext(ctx, name, args...)
	var buf bytes.Buffer
	c.Stdout = &buf
	c.Stderr = &buf

	start := time.Now()
	err := c.Run()
	dur := time.Since(start)

	return Result{
		Cmd:      pretty(name, args),
		Output:   splitLines(buf.String()),
		Err:      err,
		Duration: dur,
	}
}

// Check returns true if `name` exists on PATH.
func Check(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// LastLine returns the final non-empty output line from the Result, useful for
// summary rendering. Returns "" if there's no output.
func (r Result) LastLine() string {
	for i := len(r.Output) - 1; i >= 0; i-- {
		if strings.TrimSpace(r.Output[i]) != "" {
			return r.Output[i]
		}
	}
	return ""
}

func pretty(name string, args []string) string {
	if len(args) == 0 {
		return name
	}
	return name + " " + strings.Join(args, " ")
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	raw := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		out = append(out, line)
	}
	// Drop a trailing empty line caused by ending newline.
	if len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return out
}
