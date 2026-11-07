package kongusage_test

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/powerman/check"

	"github.com/powerman/kongusage"
	"github.com/powerman/kongusage/internal/usagetest"
)

var update = flag.Bool("update", false, "update golden files in testdata/")

// golden compares got with testdata/<name>.kdl, or writes it there with -update,
// and lints it by the tested usage versions.
func golden(t *check.C, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".kdl")
	if *update {
		t.Nil(os.WriteFile(path, []byte(got), 0o644))
	}
	want, err := os.ReadFile(path)
	t.Nil(err)
	t.Equal(got, string(want))
	usagetest.Lint(t, got, kongusage.MinUsageVersion)
}

// spec returns the spec written by WriteSpec for cli built with options.
func spec(t *check.C, cli any, cfg kongusage.Config, options ...kong.Option) string {
	t.Helper()
	options = append([]kong.Option{kong.Name("app")}, options...)
	parser, err := kong.New(cli, options...)
	t.Nil(err)
	var buf bytes.Buffer
	t.Nil(kongusage.WriteSpec(&buf, parser, cfg))
	return buf.String()
}

type (
	testAppCLI struct {
		Verbose bool   `       help:"Be verbose."`
		File    string `arg:"" help:"File to read."`
	}
	testAppDetailCLI struct{}
)

func (testAppDetailCLI) Help() string { return "Long description\nof the app." }

type (
	testCommandsCLI struct {
		Serve  testServeCmd  `cmd:""           help:"Start a server."             aliases:"s,srv"`
		Remote testRemoteCmd `cmd:""           help:"Manage remotes."`
		Debug  testLeafCmd   `cmd:"" hidden:"" help:"Debug."`
		Tool   testToolCmd   `cmd:""           help:"Tool."`
		ValRun testValRunCmd `cmd:""           help:"Run on a value receiver."`
		Opt    testOptCmd    `cmd:""           help:"Run or choose a subcommand."`
		Def    testDefCmd    `cmd:""           help:"With a default subcommand."`
	}
	testServeCmd  struct{}
	testRemoteCmd struct {
		Add testLeafCmd `cmd:"" help:"Add a remote."`
		Rm  testLeafCmd `cmd:"" help:"Remove a remote."`
	}
	testLeafCmd struct{}
	// Only hidden subcommands are not required.
	testToolCmd struct {
		Secret testLeafCmd `cmd:"" hidden:"" help:"Secret."`
	}
	// Run makes subcommands optional.
	testOptCmd struct {
		Sub testLeafCmd `cmd:"" help:"Sub."`
	}
	testValRunCmd struct {
		Sub testLeafCmd `cmd:"" help:"Sub."`
	}
	testDefCmd struct {
		A testLeafCmd `cmd:"" default:"1" help:"A."`
		B testLeafCmd `cmd:""             help:"B."`
	}
)

func (testServeCmd) Help() string { return "Serves files\nover HTTP." }
func (*testOptCmd) Run() error    { return nil }
func (testValRunCmd) Run() error  { return nil }

type testValueMapper struct{}

func (testValueMapper) Decode(*kong.DecodeContext, reflect.Value) error { return nil }

type (
	testMapperValue string
	testBoolValue   bool
)

func (*testMapperValue) Decode(*kong.DecodeContext) error { return nil }
func (*testBoolValue) Decode(*kong.DecodeContext) error   { return nil }
func (*testBoolValue) IsBool() bool                       { return true }

type testFlagsCLI struct {
	Short    string `                      help:"Short and long."                      short:"s"`
	Required string `          required:"" help:"Required."` //nolint:revive // Kong's idiom.
	Hidden   bool   `hidden:""             help:"Hidden."`
	Place    string `                      help:"Placeholder."      placeholder:"FILE"`
	Help2    string `                      help:"Multi-line\nhelp."`
	NoHelp   string
}

type testFlagDefaultsCLI struct {
	String    string        `default:"x y"  help:"String."`
	Int       int           `default:"8080" help:"Int."`
	BoolTrue  bool          `default:"true" help:"Bool true."`
	BoolYes   bool          `default:"YES"  help:"Bool yes."`   //nolint:revive // Kong takes it.
	BoolFalse bool          `default:"0"    help:"Bool false."` //nolint:revive // Kong takes it.
	BoolVal   testBoolValue `default:"true" help:"MapperValue with IsBool."`
	Count     int           `default:"2"    help:"Counter."                 type:"counter"`
}

type (
	testFlagKindsCLI struct {
		Neg       bool              `help:"Negatable."               negatable:""`
		NegX      bool              `help:"Negatable custom."        negatable:"without-x"`
		Verbose   int               `help:"Counter."                                       short:"v" type:"counter"`
		Slice     []string          `help:"Slice."`
		Named     testNamedSlice    `help:"Named slice."`
		Map       map[string]string `help:"Map."`
		Mapped    string            `help:"Custom mapper."                                           type:"testvalue"`
		MapperVal testMapperValue   `help:"MapperValue."`
		BoolVal   testBoolValue     `help:"MapperValue with IsBool."`
	}
	testNamedSlice []string
)

type testFlagEnvCLI struct {
	Env     string           `env:"APP_A,APP_B,APP_C" help:"Env."`
	NoEnv   string           `env:"-"                 help:"No env."`
	Version kong.VersionFlag `env:"APP_VERSION"       help:"Show version."`
}

type testFlagHelpVarsCLI struct {
	Color string `default:"auto"             enum:"auto,always,never" help:"Enum."`
	Vars  string `default:"a"    env:"APP_H" enum:"a,b"               help:"Default ${default}, one of ${enum}, from ${env}."`
}

type testHyphenPrefixedCLI struct {
	String  string `help:"Takes a value."`
	Bool    bool   `help:"Takes no value."`
	Verbose int    `help:"Counter."        type:"counter"`
}

// Two is required by its last group.
type testGroupsCLI struct {
	A   string `            help:"A."          and:"ab"`
	B   string `            help:"B."          and:"ab"`
	C   string `required:"" help:"C."          and:"cde"` //nolint:revive // Kong's idiom.
	D   string `            help:"D."          and:"cde"`
	E   string `            help:"E."          and:"cde"`
	F   string `            help:"F."                       lastwins:"fg"`
	G   string `            help:"G."                       lastwins:"fg"`
	Two string `            help:"Two groups." and:"ab,cde"`
}

type (
	testPositionalsCLI struct {
		Required string   `arg:""                                                help:"Required."`
		Enum     string   `arg:""                                     enum:"a,b" help:"Enum."`
		Optional string   `arg:"" optional:""                                    help:"Optional."`
		Default  string   `arg:""             default:"x" env:"APP_D"            help:"Default ${default}."`
		Rest     []string `arg:"" optional:""                                    help:"Rest."`
	}
	testRequiredSliceCLI struct {
		Map map[string]string `arg:"" help:"Map."`
	}
)

func TestWriteSpec(tt *testing.T) {
	tt.Parallel()

	tests := []struct {
		name    string
		cli     any
		cfg     kongusage.Config
		options []kong.Option
	}{
		{"application", &testAppCLI{}, kongusage.Config{}, []kong.Option{kong.Description("The app.")}},
		{"application_detail", &testAppDetailCLI{}, kongusage.Config{}, nil},
		{"commands", &testCommandsCLI{}, kongusage.Config{}, nil},
		{"commands_no_display_order", &testCommandsCLI{}, kongusage.Config{NoDisplayOrder: true}, nil},
		{"flags", &testFlagsCLI{}, kongusage.Config{}, nil},
		{"flag_defaults", &testFlagDefaultsCLI{}, kongusage.Config{}, nil},
		{"flag_kinds", &testFlagKindsCLI{}, kongusage.Config{}, []kong.Option{
			kong.NamedMapper("testvalue", testValueMapper{}),
		}},
		{"flag_env", &testFlagEnvCLI{}, kongusage.Config{}, nil},
		{"flag_help_vars", &testFlagHelpVarsCLI{}, kongusage.Config{}, nil},
		{"flag_hyphen_prefixed", &testHyphenPrefixedCLI{}, kongusage.Config{}, []kong.Option{
			kong.WithHyphenPrefixedParameters(true),
		}},
		{"flag_groups", &testGroupsCLI{}, kongusage.Config{}, nil},
		{"positionals", &testPositionalsCLI{}, kongusage.Config{}, nil},
		{"positionals_map", &testRequiredSliceCLI{}, kongusage.Config{}, nil},
	}
	for _, tc := range tests {
		tt.Run(tc.name, func(tt *testing.T) {
			tt.Parallel()
			t := check.T(tt).MustAll()

			golden(t, tc.name, spec(t, tc.cli, tc.cfg, tc.options...))
		})
	}
}

func TestWriteSpec_writeError(tt *testing.T) {
	tt.Parallel()
	t := check.T(tt)

	parser, err := kong.New(&testAppCLI{}, kong.Name("app"))
	t.Nil(err)
	t.Err(kongusage.WriteSpec(errWriter{}, parser, kongusage.Config{}), errWrite)
}

var errWrite = errors.New("write failed")

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) { return 0, errWrite }
