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

type testFlagSlicesCLI struct {
	Comma    []string          `                         help:"Default sep."`
	NoSep    []string          `                         help:"No sep."                       sep:"none"`
	Space    []string          `                         help:"Space sep."                    sep:" "`
	Enum     []string          `default:"a,b" enum:"a,b" help:"Enum."`
	Map      map[string]string `                         help:"Default mapsep."`
	MapSep   map[string]string `                         help:"Custom mapsep."  mapsep:","`
	MapNoSep map[string]int    `                         help:"No mapsep."      mapsep:"none"`
}

type (
	testInheritedCLI struct {
		Root  string           `       help:"Root flag."`
		Group testInheritedGr  `cmd:"" help:"Has subcommands."`
		Leaf  testInheritedLf  `cmd:"" help:"Leaf."`
		Tool  testToolFlagsCmd `cmd:"" help:"Only a hidden subcommand."`
	}
	testInheritedGr struct {
		Mid  string          `       help:"Mid flag."`
		Leaf testInheritedLf `cmd:"" help:"Leaf."`
	}
	testInheritedLf struct {
		Own string `help:"Leaf flag."`
	}
	testToolFlagsCmd struct {
		Flag   bool        `                 help:"Flag."`
		Secret testLeafCmd `cmd:"" hidden:"" help:"Secret."`
	}
)

type testVersionCLI struct {
	Version kong.VersionFlag `       help:"Show version." short:"V"`
	Sub     testLeafCmd      `cmd:"" help:"Sub."`
}

type testVersionVarCLI struct {
	Name string `help:"Name."`
}

type testFlagAliasesCLI struct {
	Flag  string `help:"Aliases."         aliases:"f,flg"`
	Short bool   `help:"Short and alias." aliases:"sh"                 short:"s"`
	Neg   bool   `help:"Negatable."       aliases:"n"     negatable:""`
}

// testPlaceHolderMapper provides a placeholder.
type testPlaceHolderMapper struct{ placeHolder string }

func (testPlaceHolderMapper) Decode(*kong.DecodeContext, reflect.Value) error { return nil }
func (m testPlaceHolderMapper) PlaceHolder(*kong.Flag) string                 { return m.placeHolder }

type testValueNamesCLI struct {
	Int      int               `default:"8080" help:"Default int."`
	String   string            `default:"x"    help:"Default string."`
	Map      map[string]string `default:"a=1"  help:"Default map."`
	Angles   string            `               help:"Angles."         placeholder:"<FILE>"`
	Twice    string            `               help:"Angles twice."   placeholder:"<<FILE>>"`
	Tags     testNamedSlice    `default:"a"    help:"Slice."          placeholder:"<TAG>"`
	Space    string            `               help:"Whitespace."     placeholder:"a b"`
	NBSP     string            `               help:"No-break space." placeholder:"a\u00a0b"`
	Empty    string            `               help:"Empty."          placeholder:"<>"`
	Dots     testNamedSlice    `               help:"Dots."           placeholder:"TAG..."`
	Ellipsis map[string]string `               help:"Ellipsis."       placeholder:"K=V…"`
	Mapper   string            `               help:"Mapper."                                type:"ph"`
	BadMap   string            `               help:"Bad mapper."                            type:"badph"`
}

type testPositionalNamesCLI struct {
	Angles string   `arg:""             name:"<file>"   help:"Angles."`
	Dots   string   `arg:""             name:"files..." help:"Dots."`
	Space  string   `arg:""             name:"a b"      help:"Whitespace."`
	Empty  []string `arg:"" optional:"" name:"<>"       help:"Empty."`
}

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
		{"flag_slices", &testFlagSlicesCLI{}, kongusage.Config{}, nil},
		{"flags_inherited", &testInheritedCLI{}, kongusage.Config{}, nil},
		{"flag_no_help", &testInheritedCLI{}, kongusage.Config{}, []kong.Option{kong.NoDefaultHelp()}},
		{"flag_version", &testVersionCLI{}, kongusage.Config{}, []kong.Option{
			kong.Vars{"version": "1.2.3"},
		}},
		{"flag_version_no_var", &testVersionCLI{}, kongusage.Config{}, nil},
		{"version_no_flag", &testVersionVarCLI{}, kongusage.Config{}, []kong.Option{
			kong.Vars{"version": "1.2.3"},
		}},
		{"flag_aliases", &testFlagAliasesCLI{}, kongusage.Config{}, nil},
		{"value_names", &testValueNamesCLI{}, kongusage.Config{}, testValueNamesOptions},
		{"positional_names", &testPositionalNamesCLI{}, kongusage.Config{}, nil},
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

var testValueNamesOptions = []kong.Option{
	kong.NamedMapper("ph", testPlaceHolderMapper{"<HOST>"}),
	kong.NamedMapper("badph", testPlaceHolderMapper{"HOST..."}),
}

func TestNotes(tt *testing.T) {
	tt.Parallel()

	tests := []struct {
		name    string
		cli     any
		options []kong.Option
		want    []string
	}{
		{"none", &testFlagsCLI{}, nil, nil},
		{"value_names", &testValueNamesCLI{}, testValueNamesOptions, []string{
			`app: flag --space: placeholder "a b" has whitespace, which usage refuses in a value name, so the spec uses "STRING"; change the placeholder tag of the flag`,
			`app: flag --nbsp: placeholder "a\u00a0b" has whitespace, which usage refuses in a value name, so the spec uses "STRING"; change the placeholder tag of the flag`,
			`app: flag --empty: placeholder "<>" is empty, so the spec uses "STRING"; change the placeholder tag of the flag`,
			`app: flag --dots: placeholder "TAG..." has an ellipsis, which usage refuses in a value name, so the spec uses "TEST-NAMED-SLICE"; change the placeholder tag of the flag`,
			`app: flag --ellipsis: placeholder "K=V…" has an ellipsis, which usage refuses in a value name, so the spec uses "KEY=VALUE"; change the placeholder tag of the flag`,
			`app: flag --bad-map: placeholder "HOST..." has an ellipsis, which usage refuses in a value name, so the spec uses "STRING"; change PlaceHolder of its mapper`,
		}},
		{"positional_names", &testPositionalNamesCLI{}, nil, []string{
			`app: positional files...: the name "files..." has an ellipsis, which usage takes as a mark of a variadic positional, so the spec uses "string"; change the name tag`,
			`app: positional <>: the name "<>" is empty, so the spec uses "arg"; change the name tag`,
		}},
		{"in_command", &struct {
			Sub struct {
				Space string `placeholder:"a b"`
			} `cmd:""`
		}{}, nil, []string{
			`app sub: flag --space: placeholder "a b" has whitespace, which usage refuses in a value name, so the spec uses "STRING"; change the placeholder tag of the flag`,
		}},
	}
	for _, tc := range tests {
		tt.Run(tc.name, func(tt *testing.T) {
			tt.Parallel()
			t := check.T(tt).MustAll()

			parser, err := kong.New(tc.cli, append([]kong.Option{kong.Name("app")}, tc.options...)...)
			t.Nil(err)
			t.DeepEqual(kongusage.Notes(parser), tc.want)
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
