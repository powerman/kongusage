// Package usagetest runs the usage CLI in tests.
//
// The current version of usage is the one in PATH,
// other versions are run by `mise exec` (see mise.toml).
// Tests which need usage fail without it,
// unless the environment variable [SkipEnv] is not empty: then they are skipped.
package usagetest

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// SkipEnv names the environment variable which makes tests needing usage skip.
const SkipEnv = "KONGUSAGE_TEST_SKIP_USAGE"

// Output runs the usage CLI of the version ("" for the current one) with args and stdin
// and returns its stdout.
// It fails tb when usage exits with an error, reporting stderr.
func Output(tb testing.TB, version, stdin string, args ...string) string {
	tb.Helper()
	out, err := run(tb, version, stdin, args...)
	if err != nil {
		tb.Fatalf("usage %s %s: %v\n%s", version, strings.Join(args, " "), err, out)
	}
	return out
}

// Lint fails tb when `usage lint --warnings-as-errors` of the current usage
// or of any of the versions reports an error or a warning in spec.
func Lint(tb testing.TB, spec string, versions ...string) {
	tb.Helper()
	for _, version := range append([]string{""}, versions...) {
		out, err := run(tb, version, spec, "lint", "--warnings-as-errors", "-")
		if err != nil {
			tb.Errorf("usage %s lint: %v\n%s", version, err, out)
		}
	}
}

func run(tb testing.TB, version, stdin string, args ...string) (string, error) {
	tb.Helper()
	if os.Getenv(SkipEnv) != "" {
		tb.Skipf("usage is skipped by %s", SkipEnv)
	}
	name := "usage"
	if version != "" {
		name = "mise"
		args = append([]string{"exec", "usage@" + version, "--", "usage"}, args...)
	}
	path, err := exec.LookPath(name)
	if err != nil {
		tb.Fatalf("%s is required to run usage (set %s=1 to skip): %v", name, SkipEnv, err)
	}

	cmd := exec.CommandContext(tb.Context(), path, args...) //nolint:gosec // Runs usage.
	cmd.Stdin = strings.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		return stdout.String() + stderr.String(), err
	}
	return stdout.String(), nil
}
