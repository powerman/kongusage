package kongusage_test

import (
	"strings"
	"testing"

	"github.com/powerman/check"

	"github.com/powerman/kongusage"
	"github.com/powerman/kongusage/internal/usagetest"
)

type testRootUsageKDLCLI struct {
	Run   testLeafCmd `cmd:"" default:"1" help:"Run."`
	Other testLeafCmd `cmd:""             help:"Other."`
}

// UsageKDL gives every feature of the root, in an order other than the written one.
func (testRootUsageKDLCLI) UsageKDL() string {
	return `
default_subcommand_help #true
max_term_width 120
term_width 100
flatten_help #false
next_line_help #true
subcommand_value_name "COMMAND"
subcommand_help_heading "Commands"
available_if "linux" "macos"
surface "cli"
output "report" media_type="application/json" framing="json" help="The report." schema="{}" default=#true hide=#false
output "log"
exit_code 0 "Success."
exit_code 2 "Usage error."
example "app run" lang="sh" help="Runs it." header="Run"
example "app run --help"
after_long_help "After long."
after_help "After."
before_long_help "Before long."
before_help "Before."
logo "APP" style="bold"
source_code_link_template "https://github.com/example/app/blob/main/{{path}}"
repository "https://github.com/example/app"
license "MIT"
author "Someone <someone@example.com>"
about_md "The **app**."
`
}

type (
	testCmdUsageKDLCLI struct {
		Props    testCmdPropsCmd    `cmd:"" help:"Features as properties."`
		Children testCmdChildrenCmd `cmd:"" help:"Features as child nodes."`
	}
	testCmdPropsCmd struct {
		Sub testLeafCmd `cmd:"" help:"Sub."`
	}
	testCmdChildrenCmd struct {
		Sub testLeafCmd `cmd:"" help:"Sub."`
	}
)

func (testCmdPropsCmd) UsageKDL() string {
	return `cmd "" max_term_width=90 term_width=80 flatten_help=#true next_line_help=#true \
	subcommand_value_name="SUB" subcommand_help_heading="Subs" available_if="linux" surface="cli" \
	effect="destructive" after_help_md="AM." after_long_help="AL." after_help="A." \
	before_help_md="BM." before_long_help="BL." before_help="B." help_md="Help **md**."`
}

func (testCmdChildrenCmd) UsageKDL() string {
	return `cmd "" {
    output "log"
    exit_code 1 "Failed."
    example "app children sub"
    max_term_width 90
    term_width 80
    flatten_help #true
    next_line_help #true
    subcommand_value_name "SUB"
    subcommand_help_heading "Subs"
    available_if "linux" "macos"
    surface "cli"
    effect "write"
    after_help_md "AM."
    after_long_help "AL."
    after_help "A."
    before_help_md "BM."
    before_long_help "BL."
    before_help "B."
    help_md "Help **md**."
}`
}

//nolint:lll // Tags.
type testArgUsageKDLCLI struct {
	Props    string `                                     help:"Properties."  usage-kdl:"hide_long_help=#false hide_short_help=#true available_if=linux surface=cli effect=read help_md=\"**Props**.\" long_help=\"Long.\""`
	Children string `                                     help:"Child nodes." usage-kdl:"{ effect write; hide_short_help #true; available_if a b; warning \"W.\"; note \"N1.\"; note \"N2.\"; help_md \"**Children**.\"; long_help \"Long.\"; }"`
	Run      string `                                     help:"Run."         usage-kdl:"{ complete run=\"app hosts\" descriptions=#true; }"`
	Delegate string `                                     help:"Delegate."    usage-kdl:"{ complete delegate=\"app complete\"; }"`
	Type     int    `                                     help:"Type."        usage-kdl:"{ complete type=hostname; }"`
	Num      int    `                                     help:"Suggested."   usage-kdl:"{ choices \"1\" \"2\" strict=#false; }"`
	Size     string `                                     help:"Details."     usage-kdl:"{ choices s m strict=#false { choice l help=\"Large.\"; choice s help=\"Small.\" hide=#true; }; }"`
	Env      string `       default:"dev" enum:"prod,dev" help:"Enum."        usage-kdl:"{ choices { choice prod help=\"Production.\"; }; note \"Note.\"; }"`
	File     string `arg:""                               help:"File."        usage-kdl:"{ note \"Arg note.\"; complete type=file; }"`
}

//nolint:lll // Tags.
type (
	testMergeCLI struct {
		Deploy testDeployCmd `cmd:"" help:"Deploy." usage-kdl:"{ example \"app deploy --env=dev\"; flag \"--env <ENV>\" { note #null; }; flag \"--region <STRING>\" { note \"Tag.\"; }; }"`
		Build  testBuildCmd  `cmd:"" help:"Build."`
		Rm     testRmCmd     `cmd:"" help:"Remove."`
	}
	testDeployCmd struct {
		Env    string      `required:""             enum:"prod,dev" help:"Env."    placeholder:"ENV" usage-kdl:"{ choices { choice dev help=#null hide=#true; }; long_help \"Tag.\"; }"` //nolint:revive // Kong's idiom.
		Force  bool        `                                        help:"Force."                    usage-kdl:"effect=#null hide_short_help=#true"`
		Region string      `                                        help:"Region."`
		Size   testSizeID  `                                        help:"Size."                     usage-kdl:"{ choices l xl { choice xl help=\"Extra large.\"; choice s hide=#true; }; }"`
		Level  testLevelID `            default:"a" enum:"a,b"      help:"Level."                    usage-kdl:"{ choices #null; }"`
	}
	testBuildCmd struct{}
	testRmCmd    struct {
		ID testID `arg:"" help:"ID." usage-kdl:"{ warning \"Tag.\"; }"`
	}
	testID      string
	testLevelID string
	testSizeID  string
)

func (testMergeCLI) UsageKDL() string {
	return `
cmd "build" effect=write
cmd "rm" {
    arg "<id>" { note "Root."; }
}
`
}

func (testDeployCmd) UsageKDL() string {
	return `cmd "" effect=destructive {
    example "app deploy --env=prod"
    example "app deploy --env=dev --force"
    flag "--env <ENV>" long_help="Long." {
        note "Note."
        choices { choice prod help="Prod."; choice dev help="Dev."; }
    }
    flag "--force" effect=destructive long_help="Overwrites."
}`
}

func (testBuildCmd) UsageKDL() string { return `cmd "" effect=read` }
func (testSizeID) UsageKDL() string {
	return `flag "" { choices s m strict=#false { choice s help="Small."; }; }`
}
func (testID) UsageKDL() string { return `arg "" { complete run="app ids"; note "Type."; }` }

// UsageKDL has a choice outside the enum of a field, which the tag of the field removes.
func (testLevelID) UsageKDL() string { return `flag "" { choices { choice c help="C."; }; }` }

type (
	testTypesCLI struct {
		Embed testEmbedCmd `cmd:"" help:"Promoted UsageKDL."`
		Value testValueCmd `cmd:"" help:"Value receiver."`
		ID    testID       `       help:"Value type of a flag."`
		PtrID *testID      `       help:"Pointer to a value type."`
	}
	testReusable struct{}
	testEmbedCmd struct{ testReusable }
	testValueCmd struct {
		Pos testPtrIDName `arg:"" help:"Pointer receiver."`
	}
	testPtrIDName string
)

func (*testReusable) UsageKDL() string  { return `cmd "" effect=read` }
func (testValueCmd) UsageKDL() string   { return `cmd "" effect=write` }
func (*testPtrIDName) UsageKDL() string { return `flag "" help_md="Interchangeable with arg."` }

type (
	testMountCLI struct {
		Verbose bool         `       help:"Be verbose."`
		Tool    testToolRoot `cmd:"" help:"Run the tool."                 usage-kdl:"effect=write { example \"app tool --dry-run sub\"; }"`
		Other   testToolRoot `cmd:"" help:"Run the tool without changes."`
	}
	// testToolRoot is the root of another CLI.
	testToolRoot struct {
		DryRun bool        `       help:"Do nothing."`
		Sub    testLeafCmd `cmd:"" help:"Sub."`
	}
)

func (testToolRoot) UsageKDL() string {
	return `
about_md "The **tool**."
license "MIT"
author "Someone"
example "tool --dry-run sub" header="Dry run"
surface "cli"
flag "--dry-run" long_help="Does nothing at all."
cmd "sub" effect=read
`
}

func TestWriteSpec_usageKDL(tt *testing.T) {
	tt.Parallel()

	tests := []struct {
		name string
		cli  any
	}{
		{"usagekdl_root", &testRootUsageKDLCLI{}},
		{"usagekdl_cmd", &testCmdUsageKDLCLI{}},
		{"usagekdl_arg", &testArgUsageKDLCLI{}},
		{"usagekdl_merge", &testMergeCLI{}},
		{"usagekdl_types", &testTypesCLI{}},
		{"usagekdl_mount", &testMountCLI{}},
	}
	for _, tc := range tests {
		tt.Run(tc.name, func(tt *testing.T) {
			tt.Parallel()
			t := check.T(tt).MustAll()

			golden(t, tc.name, spec(t, tc.cli, kongusage.Config{}))
		})
	}
}

// TestWriteSpec_choicesCompletion checks suggested values of a number are completed:
// usage 6.12.0–7.3.0 complete nothing for a value with `complete type="none"` instead of choices.
func TestWriteSpec_choicesCompletion(tt *testing.T) {
	tt.Parallel()
	t := check.T(tt)

	spec := spec(t, &testArgUsageKDLCLI{}, kongusage.Config{})
	for _, version := range []string{"", kongusage.MinUsageVersion} {
		got := usagetest.Output(t, version, strings.NewReader(spec), "complete-word", "-f", "-", "--", "app", "--num", "")
		t.Equal(got, "1\n2\n", version)
	}
}

type (
	// testKDLCLI gives UsageKDL of the root by its field.
	testKDLCLI struct {
		kdl   string
		Force bool       `                                     help:"Force."`
		Env   string     `       default:"dev" enum:"prod,dev" help:"Env."   placeholder:"ENV"`
		Run   testKDLCmd `cmd:""                               help:"Run."`
	}
	// testKDLCmd gives UsageKDL of a command by its field.
	testKDLCmd struct {
		kdl  string
		File string `arg:"" help:"File."`
	}
	testCmdValue string
)

func (c testKDLCLI) UsageKDL() string { return c.kdl }
func (c testKDLCmd) UsageKDL() string { return c.kdl }
func (testCmdValue) UsageKDL() string { return `cmd ""` }

// TestWriteSpec_usageKDLErrors checks errors which depend on the conversion:
// the source named in messages, the place of a tag, entities missing from the spec
// and checks of a value by its Kong flag or positional.
// Errors of parsing alone are checked by internal/usagekdl.
func TestWriteSpec_usageKDLErrors(tt *testing.T) {
	tt.Parallel()

	root := func(kdl string) any { return &testKDLCLI{kdl: kdl} }
	cmd := func(kdl string) any { return &testKDLCLI{Run: testKDLCmd{kdl: kdl}} }
	tests := []struct {
		name string
		cli  any
		want string
	}{
		{"root", root(`help "x"`), `^UsageKDL: app: UsageKDL of kongusage_test.testKDLCLI: help is not accepted$`},
		{"cmd", cmd(`cmd "" { note "x"; }`), `^UsageKDL: app run: UsageKDL of kongusage_test.testKDLCmd: cmd "": note is not accepted on commands$`},
		{"value type", &struct {
			X testCmdValue
		}{}, `^UsageKDL: app: flag --x: UsageKDL of kongusage_test.testCmdValue: it must return one node flag "" or arg ""$`},
		{"cmd tag", &struct {
			Run testLeafCmd `cmd:"" usage-kdl:"{ license MIT; }"`
		}{}, `^UsageKDL: app run: usage-kdl tag: license is not accepted on commands$`},
		{"flag tag", &struct {
			X string `usage-kdl:"{ example x; }"`
		}{}, `^UsageKDL: app: flag --x: usage-kdl tag: example is not accepted on flags and positionals$`},
		{"entity not in spec", root(`flag "-f --force"`), `^UsageKDL: app: flag "-f --force" is not in the spec$`},
		{"nested not in spec", root(`cmd "run" { arg "[file]"; }`), `^UsageKDL: app run: arg "\[file\]" is not in the spec$`},
		{"mounted root empty name", cmd(`flag ""`), `^UsageKDL: app run: flag "" is not in the spec$`},
		{"choices without value", root(`flag "--force" { choices "a" strict=#false; }`), `^UsageKDL: app: flag --force: choices is not accepted on a flag without a value$`},
		{"choices strict", &struct {
			X string `usage-kdl:"{ choices a b; }"`
		}{}, `^UsageKDL: app: flag --x: choices: strict=#false is required without an enum, as Kong accepts any value$`},
		{"enum choice", root(`flag "--env <ENV>" { choices { choice test; }; }`), `^UsageKDL: app: flag --env: choices: choice "test" is not in the enum$`},
	}
	for _, tc := range tests {
		tt.Run(tc.name, func(tt *testing.T) {
			tt.Parallel()
			t := check.T(tt)

			t.PanicMatch(func() { spec(t, tc.cli, kongusage.Config{}) }, tc.want)
		})
	}
}
