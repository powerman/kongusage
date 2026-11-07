// Package usagecli runs the usage CLI for tests of kongusage and for its CheckSpec.
package usagecli

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"time"
)

// SkipEnv names the environment variable which makes tests skip running usage,
// e.g. on platforms without it.
const SkipEnv = "KONGUSAGE_TEST_SKIP_USAGE"

// Skipped reports whether running usage is skipped by [SkipEnv].
func Skipped() bool {
	return os.Getenv(SkipEnv) != ""
}

// waitDelay limits waiting for output of usage after it is killed.
const waitDelay = 100 * time.Millisecond

// Run runs the usage CLI of the version with args and stdin (nil for none)
// and returns its stdout and stderr.
// The version "" is the one in PATH, other versions are run by `mise exec`.
func Run(ctx context.Context, version string, stdin io.Reader, args ...string) (stdout, stderr string, err error) {
	name := "usage"
	if version != "" {
		name = "mise"
		args = append([]string{"exec", "usage@" + version, "--", "usage"}, args...)
	}
	cmd := exec.CommandContext(ctx, name, args...) //nolint:gosec // Runs usage.
	cmd.Stdin = stdin
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	// After ctx is done, do not wait for output of children which outlive a killed usage,
	// e.g. of a wrapper script.
	cmd.WaitDelay = waitDelay
	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}
