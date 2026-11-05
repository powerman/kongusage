package kongusage_test

import (
	"bytes"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/powerman/check"

	"github.com/powerman/kongusage"
)

type testCLI struct {
	Name string `help:"Name."`

	UsageSpec kongusage.Flag `env:"-"`
}

type testHelpCLI struct {
	UsageSpec kongusage.Flag `help:"Custom."`
}

type testPtrCLI struct {
	UsageSpec *kongusage.Flag
}

type testRequiredCLI struct {
	Name string `arg:""`

	UsageSpec kongusage.Flag
}

type testEnvCLI struct {
	Name string `help:"Name."`

	UsageSpec kongusage.Flag
}

type testConfigCLI struct {
	testCLI

	cfg kongusage.Config
}

func (c testConfigCLI) UsageConfig() kongusage.Config { return c.cfg }

// newParser returns a parser of cli
// and a function returning its output and exit code (-1 if no exit) since the last call.
func newParser(t *check.C, cli any) (*kong.Kong, func() (string, int)) {
	t.Helper()
	var out bytes.Buffer
	exit := -1
	parser, err := kong.New(cli,
		kong.Name("test"),
		kong.DefaultEnvars("TEST"),
		kong.Writers(&out, &out),
		kong.Exit(func(code int) { exit = code }),
	)
	t.Nil(err)
	result := func() (string, int) {
		defer func() { out.Reset(); exit = -1 }()
		return out.String(), exit
	}
	return parser, result
}

// parse parses args by cli and returns its output, exit code (-1 if no exit) and error.
func parse(t *check.C, cli any, args ...string) (string, int, error) {
	t.Helper()
	parser, result := newParser(t, cli)
	_, err := parser.Parse(args)
	out, exit := result()
	return out, exit, err
}

func TestFlag_Signature(tt *testing.T) {
	tt.Parallel()

	tests := []struct {
		cli     any
		wantOut string
	}{
		{&testCLI{}, `(?m)^\s+--usage-spec\s+Print usage spec \(KDL\) and exit\.$`},
		{&testHelpCLI{}, `(?m)^\s+--usage-spec\s+Custom\b`},
	}
	for _, tc := range tests {
		tt.Run("", func(tt *testing.T) {
			tt.Parallel()
			t := check.T(tt).MustAll()

			out, exit, err := parse(t, tc.cli, "--help")
			t.Nil(err)
			t.Equal(exit, 0)
			t.Match(out, tc.wantOut)
		})
	}
}

func TestFlag_BeforeReset(tt *testing.T) {
	tt.Parallel()

	tests := []struct {
		cli      any
		args     []string
		wantExit int
		wantOut  string
		wantErr  string
	}{
		{&testCLI{}, []string{"--usage-spec"}, 0, `(?m)^flag --name `, ""},
		{&testCLI{}, []string{"--usage-spec=true"}, 0, `(?m)^flag --name `, ""},
		{&testCLI{}, []string{"--usage-spec=false"}, -1, `^$`, ""},
		{&testCLI{}, []string{"--usage-spec", "--name=x"}, 0, `(?m)^flag --name `, ""},
		{&testCLI{}, []string{"--usage-spec", "x"}, -1, `^$`, `^unexpected argument x$`},
		{&testCLI{}, []string{"--usage-spec=x"}, -1, `^$`, `^--usage-spec: bool value must be`},
		{&testPtrCLI{}, []string{"--usage-spec"}, 0, `(?m)^bin `, ""},
		{&testPtrCLI{}, []string{"--usage-spec=false"}, -1, `^$`, ""},
		// The spec is printed before the check of required arguments,
		// which fails only because the test's exit does not stop parsing.
		{&testRequiredCLI{}, []string{"--usage-spec"}, 0, `(?m)^arg <name>`, `^expected "<name>"$`},
		{&testRequiredCLI{}, nil, -1, `^$`, `^expected "<name>"$`},
		{
			&testConfigCLI{cfg: kongusage.Config{NoDisplayOrder: true}},
			[]string{"--usage-spec"},
			0, `(?m)^flag --name `, "",
		},
	}
	for _, tc := range tests {
		tt.Run("", func(tt *testing.T) {
			tt.Parallel()
			t := check.T(tt).MustAll()

			out, exit, err := parse(t, tc.cli, tc.args...)
			t.Equal(exit, tc.wantExit)
			t.Match(out, tc.wantOut)
			if tc.wantErr == "" {
				t.Nil(err)
			} else {
				t.Match(err, tc.wantErr)
			}
		})
	}
}

func TestFlag_BeforeReset_env(tt *testing.T) {
	tests := []struct {
		value   string
		wantErr string
	}{
		{"true", ""},
		{"1", ""},
		{"false", ""},
		// Kong fails to apply the variable to the field.
		{"x", `^--usage-spec: bool value must be`},
	}
	for _, tc := range tests {
		tt.Run(tc.value, func(tt *testing.T) {
			t := check.T(tt).MustAll()
			tt.Setenv("TEST_USAGE_SPEC", tc.value)

			// The second parse checks the flag ignores a value left in the field by the first one.
			parser, result := newParser(t, &testEnvCLI{})
			for range 2 {
				_, err := parser.Parse(nil)
				out, exit := result()
				t.Equal(exit, -1)
				t.Equal(out, "")
				if tc.wantErr == "" {
					t.Nil(err)
				} else {
					t.Match(err, tc.wantErr)
				}
			}
		})
	}
}
