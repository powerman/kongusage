package kongusage_test

import (
	"bytes"
	"errors"
	"flag"
	urlpkg "net/url"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

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

type (
	testPassthroughCLI struct {
		Exec    testPassthroughCmd `cmd:"" passthrough:"" help:"Passthrough command."`
		All     testPassthroughAll `cmd:""                help:"Passthrough positional."`
		Partial testPassthroughPrt `cmd:""                help:"Partial passthrough positional."`
	}
	testPassthroughCmd struct {
		Args []string `arg:"" optional:"" help:"Arguments."`
	}
	testPassthroughAll struct {
		Flag bool     `                                  help:"Flag."`
		Args []string `arg:"" passthrough:"" optional:"" help:"Arguments."`
	}
	testPassthroughPrt struct {
		Flag bool     `                                         help:"Flag."`
		Args []string `arg:"" passthrough:"partial" optional:"" help:"Arguments."`
	}
)

type (
	testDefaultCLI struct {
		Serve testLeafCmd    `cmd:"" default:"1" help:"Serve."`
		Other testDefaultSub `cmd:""             help:"Below the root."`
	}
	testDefaultWithArgsCLI struct {
		Serve testDefaultArgs `cmd:"" default:"withargs" help:"Serve."`
		Other testLeafCmd     `cmd:""                    help:"Other."`
	}
	testDefaultSub struct {
		A testLeafCmd `cmd:"" default:"1" help:"A."`
		B testLeafCmd `cmd:""             help:"B."`
	}
	testDefaultArgs struct {
		Port int    `                   help:"Port."`
		Dir  string `arg:"" optional:"" help:"Directory."`
	}
)

type (
	testCompletionCLI struct {
		Int       int                       `                              help:"Int."`
		Int8      int8                      `                              help:"Int8."`
		Int16     int16                     `                              help:"Int16."`
		Int32     int32                     `                              help:"Int32."`
		Int64     int64                     `                              help:"Int64."`
		Uint      uint                      `                              help:"Uint."`
		Uint16    uint16                    `                              help:"Uint16."`
		Uint32    uint32                    `                              help:"Uint32."`
		Uint64    uint64                    `                              help:"Uint64."`
		Float     float32                   `                              help:"Float."`
		Float64   float64                   `                              help:"Float64."`
		Named     testNamedInt              `                              help:"Named int."`
		Duration  time.Duration             `                              help:"Duration."`
		Time      time.Time                 `                              help:"Time."`
		URL       *urlpkg.URL               `                              help:"URL."`
		IntPtr    *int                      `                              help:"Pointer to int."`
		Ints      []int                     `                              help:"Slice of ints."`
		Map       map[string]int            `                              help:"Map."`
		Dir       string                    `                              help:"Existing dir."                type:"existingdir"`
		ChDir     kong.ChangeDirFlag        `                              help:"Change dir."`
		String    string                    `                              help:"String."`
		Path      string                    `                              help:"Path."                        type:"path"`
		File      string                    `                              help:"Existing file."               type:"existingfile"`
		Content   []byte                    `                              help:"File content."                type:"filecontent"`
		OSFile    *os.File                  `                              help:"File."`
		FileFlag  kong.FileContentFlag      `                              help:"File content flag."`
		NamedFile kong.NamedFileContentFlag `                              help:"Named file content flag."`
		Config    kong.ConfigFlag           `                              help:"Config."`
		Level     testLevel                 `                              help:"MapperValue of int kind."`
		Text      testTextLevel             `                              help:"TextUnmarshaler of int kind."`
		Mapped    int                       `                              help:"Custom named mapper."         type:"testvalue"`
		Enum      int                       `       default:"1" enum:"1,2" help:"Enum."`
		Verbose   int                       `                              help:"Counter."                     type:"counter"`
		Bytes     []byte                    `                              help:"Bytes."`
		Pos       int                       `arg:""                        help:"Positional."`
	}
	testNamedInt  int
	testLevel     int
	testTextLevel int
)

func (*testLevel) Decode(*kong.DecodeContext) error { return nil }
func (*testTextLevel) UnmarshalText([]byte) error   { return nil }

// testDurationMapper replaces Kong's mapper of time.Duration.
type testDurationMapper struct{}

func (testDurationMapper) Decode(*kong.DecodeContext, reflect.Value) error { return nil }

type testCustomTypeCLI struct {
	Duration time.Duration `help:"Duration with a custom mapper."`
	Int      int           `help:"Int with a custom kind mapper."`
}

type (
	testChoiceGroupsCLI struct {
		A   string              `                   help:"A."                 xor:"ab"`
		B   string              `                   help:"B."                 xor:"ab,bc"`
		C   string              `                   help:"C."                 xor:"bc"`
		D   string              `       required:"" help:"D."                 xor:"de"` //nolint:revive // Kong's idiom.
		E   string              `       required:"" help:"E."                 xor:"de"` //nolint:revive // Kong's idiom.
		F   string              `       required:"" help:"F."                 xor:"fg"` //nolint:revive // Kong's idiom.
		G   string              `                   help:"G."                 xor:"fg"`
		H   string              `       required:"" help:"H."                 xor:"h"` //nolint:revive // Kong's idiom.
		I   string              `       required:"" help:"I."   lastwins:"ij"`         //nolint:revive // Kong's idiom.
		J   string              `       required:"" help:"J."   lastwins:"ij"`         //nolint:revive // Kong's idiom.
		K   string              `                   help:"K."   lastwins:"kl"`
		L   string              `                   help:"L."   lastwins:"kl"`
		Sub testChoiceGroupsCmd `cmd:""             help:"Sub."`
	}
	testChoiceGroupsCmd struct {
		X bool `help:"X." xor:"xy"`
		Y bool `help:"Y." xor:"xy"`
	}
)

type testHiddenPositionalCLI struct {
	File string `arg:"" hidden:"" help:"Hidden positional."`
}

type (
	testHeadingsCLI struct {
		Plain  string         `       help:"Without a group."`
		Out    string         `       help:"Explicit group."       group:"out"`
		Format string         `       help:"Explicit group."       group:"out"`
		Ad     string         `       help:"Ad hoc group."         group:"Misc"`
		Remote testHeadingsGr `cmd:"" help:"Grouped command."      group:"net"`
		Local  testLeafCmd    `cmd:"" help:"Ungrouped command."`
		Fetch  testLeafCmd    `cmd:"" help:"Grouped command."      group:"net"`
		Debug  testLeafCmd    `cmd:"" help:"Ad hoc command group." group:"Debugging"`
	}
	testHeadingsGr struct {
		Add testLeafCmd `cmd:"" help:"Inherits the group."`
		Rm  testLeafCmd `cmd:"" help:"Own group."          group:"out"`
	}
)

var testHeadingsOptions = []kong.Option{kong.ExplicitGroups([]kong.Group{
	{Key: "out", Title: "Output", Description: "Output options."},
	{Key: "net", Title: "Network", Description: "Network commands."},
})}

type (
	testBranchingCLI struct {
		Rename testRenameCmd `cmd:"" help:"Rename."`
	}
	testRenameCmd struct {
		From testRenameFrom `arg:"" help:"Old name."`
	}
	testRenameFrom struct {
		From string       `       arg:"" help:"Old name."`
		To   testRenameTo `cmd:""        help:"To."`
	}
	testRenameTo struct {
		Name string `arg:"" help:"New name."`
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
		{"positional_hidden", &testHiddenPositionalCLI{}, kongusage.Config{}, nil},
		{"positional_branching", &testBranchingCLI{}, kongusage.Config{}, nil},
		{"passthrough", &testPassthroughCLI{}, kongusage.Config{}, nil},
		{"default_commands", &testDefaultCLI{}, kongusage.Config{}, nil},
		{"default_commands_withargs", &testDefaultWithArgsCLI{}, kongusage.Config{}, nil},
		{"completion", &testCompletionCLI{}, kongusage.Config{}, []kong.Option{
			kong.NamedMapper("testvalue", testValueMapper{}),
		}},
		{"completion_custom_type", &testCustomTypeCLI{}, kongusage.Config{}, []kong.Option{
			kong.TypeMapper(reflect.TypeFor[time.Duration](), testDurationMapper{}),
			kong.KindMapper(reflect.Int, testValueMapper{}),
		}},
		{"flag_choice_groups", &testChoiceGroupsCLI{}, kongusage.Config{}, nil},
		{"group_headings", &testHeadingsCLI{}, kongusage.Config{}, testHeadingsOptions},
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
