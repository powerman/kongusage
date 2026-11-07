package kongusage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime/debug"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"golang.org/x/mod/semver"

	"github.com/powerman/kongusage/internal/usagecli"
	"github.com/powerman/kongusage/internal/usagekdl"
)

// KDLProvider may be implemented by the root of a Kong CLI
// and by the type of a command, a flag or a positional argument
// to supply usage features in KDL (see "Usage features" in the package doc).
type KDLProvider interface {
	UsageKDL() string
}

// CheckSpec fails the test t on mistakes in the usage spec of the CLI k:
//
//   - on mistakes in usage features which make [WriteSpec] and thus [Flag] panic,
//     e.g. a node, value or property which is not accepted,
//     a name of a command, a flag or a positional argument which is not in the spec,
//     or a node derived from Kong (see "Usage features" in the package doc);
//   - on errors and warnings of `usage lint`, e.g. an example which the spec cannot parse.
//
// Use it in a test of the CLI:
//
//	func TestUsageSpec(t *testing.T) {
//		kongusage.CheckSpec(t, kong.Must(&CLI{}))
//	}
//
// The spec is generated with the same [Config] as by [Flag].
// CheckSpec logs each part of the CLI which the spec fixes silently,
// e.g. a placeholder which usage cannot take as a value name,
// as it is likely a mistake which is worth correcting in the CLI.
// It requires the `usage` CLI in PATH, of a version not older than `min_usage_version`;
// otherwise the test fails with the version to install.
// When the environment variable KONGUSAGE_TEST_SKIP_USAGE is not empty,
// it does not run usage, e.g. on platforms without it, and checks only usage features.
func CheckSpec(t interface {
	Helper()
	Context() context.Context
	Logf(format string, args ...any)
	Fatalf(format string, args ...any)
}, k *kong.Kong,
) {
	t.Helper()
	skipUsage := usagecli.Skipped()

	if !skipUsage {
		out, err := runUsage(t, nil, "--version")
		if err != nil {
			t.Fatalf("usage %s or newer is required in PATH (set %s=1 to skip running it): %v",
				MinUsageVersion, usagecli.SkipEnv, err)
			return
		}
		version := strings.TrimPrefix(strings.TrimSpace(out), "usage ")
		if !semver.IsValid("v"+version) || semver.Compare("v"+version, "v"+MinUsageVersion) < 0 {
			t.Fatalf("usage %s is older than %s: install usage %s or newer", version, MinUsageVersion, MinUsageVersion)
			return
		}
	}

	spec, notes, err := checkedSpec(k)
	if err != nil {
		t.Fatalf("%v", err)
		return
	}
	for _, note := range notes {
		t.Logf("kongusage: %s", note)
	}

	if skipUsage {
		t.Logf("kongusage: usage lint is skipped by %s", usagecli.SkipEnv)
		return
	}
	_, err = runUsage(t, strings.NewReader(spec), "lint", "--warnings-as-errors", "-")
	if err != nil {
		t.Fatalf("usage lint: %v", err)
	}
}

// usageTimeout limits a run of usage by CheckSpec, so that a hung usage does not hang the test.
var usageTimeout = 3 * time.Second //nolint:gochecknoglobals // Changed by tests.

// runUsage runs the usage CLI in PATH with args and stdin (nil for none)
// and returns its stdout.
// It logs stderr to t; the error includes stdout and stderr.
func runUsage(t interface {
	Helper()
	Context() context.Context
	Logf(format string, args ...any)
}, stdin io.Reader, args ...string,
) (string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), usageTimeout)
	defer cancel()
	stdout, stderr, err := usagecli.Run(ctx, "", stdin, args...)
	if err != nil {
		return "", fmt.Errorf("%w\n%s%s", err, stdout, stderr)
	}
	if stderr != "" {
		t.Logf("usage %s: %s", strings.Join(args, " "), stderr)
	}
	return stdout, nil
}

var errPanic = errors.New("panic")

// checkedSpec returns the spec of k written as by [WriteSpec] with the Config of its root
// and the notes on silent fixes, or an error on a panic,
// so that a panic fails only the test which checks the spec.
func checkedSpec(k *kong.Kong) (spec string, notes []string, err error) {
	defer func() {
		switch p := recover().(type) {
		case nil:
		case *usagekdl.Error:
			err = p
		default:
			err = fmt.Errorf("%w: %v\n%s", errPanic, p, debug.Stack())
		}
	}()
	var buf strings.Builder
	notes, err = writeSpec(&buf, k, rootConfig(k))
	return buf.String(), notes, err
}
