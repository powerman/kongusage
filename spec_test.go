package kongusage_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/alecthomas/kong"
	"github.com/powerman/check"

	"github.com/powerman/kongusage"
	"github.com/powerman/kongusage/internal/usagecli"
)

// fakeT records calls of CheckSpec; it does not stop on Fatalf.
type fakeT struct {
	ctx   context.Context //nolint:containedctx // Returned by Context, as testing.T does.
	logs  []string
	fatal []string
}

func (*fakeT) Helper() {}

func (f *fakeT) Context() context.Context { return f.ctx }

func (f *fakeT) Logf(format string, args ...any) {
	f.logs = append(f.logs, fmt.Sprintf(format, args...))
}

func (f *fakeT) Fatalf(format string, args ...any) {
	f.fatal = append(f.fatal, fmt.Sprintf(format, args...))
}

// checkSpec returns the calls of CheckSpec for cli.
func checkSpec(t *check.C, cli any) *fakeT {
	t.Helper()
	parser, err := kong.New(cli, kong.Name("app"))
	t.Must(t.Nil(err))
	f := fakeT{ctx: t.Context()}
	kongusage.CheckSpec(&f, parser)
	return &f
}

// checkFatal checks f has the only call of Fatalf matching want, or none when want is "".
func checkFatal(t *check.C, f *fakeT, want string) {
	t.Helper()
	if want == "" {
		t.Nil(f.fatal)
	} else if t.Len(f.fatal, 1) {
		t.Match(f.fatal[0], want)
	}
}

type testPanicCLI struct{}

func (testPanicCLI) UsageKDL() string { panic("boom") }

// testNotesCLI has a silent fix of the spec.
type testNotesCLI struct {
	Sub struct {
		Space string `placeholder:"a b"`
	} `cmd:""`
}

const testNote = `kongusage: app sub: flag --space: placeholder "a b" has whitespace, which usage refuses in a value name, so the spec uses "STRING"; change the placeholder tag of the flag`

func TestCheckSpec(tt *testing.T) {
	tt.Parallel()
	if usagecli.Skipped() {
		tt.Skipf("usage is skipped by %s", usagecli.SkipEnv)
	}

	tests := []struct {
		name      string
		cli       any
		wantLogs  []string
		wantFatal string
	}{
		{"valid", &testMergeCLI{}, nil, ""},
		{"config", &testConfigCLI{cfg: kongusage.Config{NoDisplayOrder: true}}, nil, ""},
		{"notes", &testNotesCLI{}, []string{testNote}, ""},
		{
			"feature error", &testKDLCLI{kdl: `help "x"`}, nil,
			`^UsageKDL: app: UsageKDL of kongusage_test.testKDLCLI: help is not accepted$`,
		},
		{
			"panic", &testPanicCLI{}, nil,
			`(?s)^panic: boom\n.*kongusage_test\.testPanicCLI\.UsageKDL`,
		},
		{
			"lint", &testKDLCLI{kdl: `example "app nope"`}, nil,
			`(?s)^usage lint: exit status 1\n.*example-does-not-parse`,
		},
	}
	for _, tc := range tests {
		tt.Run(tc.name, func(tt *testing.T) {
			tt.Parallel()
			t := check.T(tt)

			f := checkSpec(t, tc.cli)
			t.DeepEqual(f.logs, tc.wantLogs)
			checkFatal(t, f, tc.wantFatal)
		})
	}
}

// TestCheckSpec_version checks the version of usage by fake usage CLIs in PATH.
func TestCheckSpec_version(tt *testing.T) {
	if runtime.GOOS == "windows" {
		tt.Skip("fake usage CLIs are shell scripts")
	}
	tt.Setenv(usagecli.SkipEnv, "")

	tests := []struct {
		version   string // Empty for no usage in PATH.
		wantFatal string
	}{
		{"", `^usage 6\.12\.0 or newer is required in PATH \(set KONGUSAGE_TEST_SKIP_USAGE=1 to skip running it\): `},
		{"6.11.9", `^usage 6\.11\.9 is older than 6\.12\.0: install usage 6\.12\.0 or newer$`},
		{"6.12.x", `^usage 6\.12\.x is older than 6\.12\.0`},
		{"6.12.0", ""},
		{"10.0.0", ""},
	}
	for _, tc := range tests {
		tt.Run(tc.version, func(tt *testing.T) {
			t := check.T(tt)

			dir := t.TempDir()
			if tc.version != "" {
				script := fmt.Sprintf("#!/bin/sh\n[ \"$1\" = --version ] && echo 'usage %s' && echo warn >&2\nexit 0\n", tc.version)
				t.Nil(os.WriteFile(filepath.Join(dir, "usage"), []byte(script), 0o755))
			}
			tt.Setenv("PATH", dir)

			f := checkSpec(t, &testFlagsCLI{})
			checkFatal(t, f, tc.wantFatal)
			if tc.version != "" {
				t.DeepEqual(f.logs, []string{"usage --version: warn\n"})
			}
		})
	}
}

// TestCheckSpec_timeout checks a hung usage does not hang the test,
// even when its child holds the output.
func TestCheckSpec_timeout(tt *testing.T) {
	if runtime.GOOS == "windows" {
		tt.Skip("fake usage CLIs are shell scripts")
	}
	t := check.T(tt)
	tt.Setenv(usagecli.SkipEnv, "")
	sleep, err := exec.LookPath("sleep")
	t.Must(t.Nil(err))
	dir := t.TempDir()
	script := "#!/bin/sh\n" + sleep + " 10\n"
	t.Nil(os.WriteFile(filepath.Join(dir, "usage"), []byte(script), 0o755))
	tt.Setenv("PATH", dir)
	timeout := *kongusage.UsageTimeout
	*kongusage.UsageTimeout = 100 * time.Millisecond
	tt.Cleanup(func() { *kongusage.UsageTimeout = timeout })

	start := time.Now()
	f := checkSpec(t, &testFlagsCLI{})
	t.Less(time.Since(start), 5*time.Second)
	checkFatal(t, f, `^usage 6\.12\.0 or newer is required in PATH .*: signal: killed\n$`)
}

// TestCheckSpec_skipUsage checks usage features without usage in PATH.
func TestCheckSpec_skipUsage(tt *testing.T) {
	tt.Setenv(usagecli.SkipEnv, "1")
	tt.Setenv("PATH", tt.TempDir())

	const skipped = "kongusage: usage lint is skipped by KONGUSAGE_TEST_SKIP_USAGE"
	tests := []struct {
		name      string
		cli       any
		wantLogs  []string
		wantFatal string
	}{
		{"notes", &testNotesCLI{}, []string{testNote, skipped}, ""},
		{"feature error", &testKDLCLI{kdl: `help "x"`}, nil, `help is not accepted$`},
		{"lint", &testKDLCLI{kdl: `example "app nope"`}, []string{skipped}, ""},
	}
	for _, tc := range tests { //nolint:paralleltest // Setenv forbids parallel tests.
		tt.Run(tc.name, func(tt *testing.T) {
			t := check.T(tt)

			f := checkSpec(t, tc.cli)
			t.DeepEqual(f.logs, tc.wantLogs)
			checkFatal(t, f, tc.wantFatal)
		})
	}
}
