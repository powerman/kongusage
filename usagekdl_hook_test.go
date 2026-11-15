package kongusage_test

import (
	"fmt"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/powerman/check"

	"github.com/powerman/kongusage"
)

type (
	// testHookCLI has a command for every source of usage features
	// which a hook of a node is merged with.
	testHookCLI struct {
		Deploy testHookCmd `cmd:"" help:"UsageKDL of the type."`
		Show   testLeafCmd `cmd:"" help:"No usage features."`
		Pin    testHookCmd `cmd:"" help:"The tag replaces."      usage-kdl:"{ exit_code 7 \"Tag.\"; }"`
		Rm     testHookCmd `cmd:"" help:"The tag removes."       usage-kdl:"{ exit_code #null; }"`
		Ls     testHookCmd `cmd:"" help:"Described by the root."`
		Grp    testHookGrp `cmd:"" help:"Has a subcommand."`
	}
	testHookCmd struct{}
	testHookGrp struct {
		Leaf testHookCmd `cmd:"" help:"Leaf."`
	}
)

func (testHookCLI) UsageKDL() string {
	return `
license MIT
exit_code 9 "Own."
cmd "ls" effect=read
`
}

func (testHookCmd) UsageKDL() string {
	return `cmd "" effect=destructive { example "app show"; exit_code 9 "Own."; }`
}

// testHookCodes gives exit statuses to the root and to every command but show,
// as a library does. Its text for commands has several lines and starts with an empty one.
func testHookCodes(n *kong.Node) string {
	switch {
	case n.Type == kong.ApplicationNode:
		return `{ exit_code 3 "Root."; }`
	case n.Name == "show":
		return ""
	default:
		return "\n{\n    exit_code 3 \"Root.\"\n    exit_code 4 \"Cmd.\"\n}\n"
	}
}

// testHookPatch changes what testHookCodes and UsageKDL gave to some commands,
// as the author of a CLI does after a hook of a library.
func testHookPatch(n *kong.Node) string {
	switch n.FullPath() {
	case "app deploy":
		return `{ exit_code 5 "Patched."; }`
	case "app show":
		return `effect=read { example "app show"; }`
	case "app ls":
		return `effect=#null { exit_code #null; }`
	case "app grp":
		return `{ cmd "leaf" effect=write; }`
	default:
		return ""
	}
}

// TestWriteSpec_nodeKDL checks Config.NodeKDL is merged as a tag:
// in order, after UsageKDL of the type or of an ancestor and before the tag of the field.
func TestWriteSpec_nodeKDL(tt *testing.T) {
	tt.Parallel()

	tests := []struct {
		name  string
		hooks []kongusage.NodeKDLFunc
	}{
		{"usagekdl_nodekdl", []kongusage.NodeKDLFunc{testHookCodes}},
		{"usagekdl_nodekdl_order", []kongusage.NodeKDLFunc{testHookCodes, testHookPatch}},
	}
	for _, tc := range tests {
		tt.Run(tc.name, func(tt *testing.T) {
			tt.Parallel()
			t := check.T(tt).MustAll()

			golden(t, tc.name, spec(t, &testHookCLI{}, kongusage.Config{NodeKDL: tc.hooks}, kong.Description("App.")))
		})
	}
}

type (
	// testValueHookCLI has a flag or a positional for every source of usage features
	// which a hook of a value is merged with.
	testValueHookCLI struct {
		Level testHookLevel    `       help:"UsageKDL of the type."`
		Host  string           `       help:"Only hooks."           predictor:"hosts"`
		User  string           `       help:"The tag replaces."     predictor:"users" usage-kdl:"{ complete type=username; }"`
		Force bool             `       help:"No usage features."`
		Run   testValueHookRun `cmd:"" help:"Run."                                    usage-kdl:"{ flag \"--job <STRING>\" long_help=\"Described.\"; }"`
	}
	testValueHookRun struct {
		Job string        `       help:"Described by the command." predictor:"jobs"`
		ID  testHookLevel `arg:"" help:"Positional."               predictor:"ids"`
	}
	testHookLevel string
)

func (testHookLevel) UsageKDL() string {
	return `flag "" { note "Type."; complete run="app levels"; }`
}

// testHookPredictor completes a value with the tag `predictor` by a command of the CLI,
// as the author of a CLI with tags of its own does.
func testHookPredictor(_ *kong.Node, v *kong.Value) string {
	name := v.Tag.Get("predictor")
	if name == "" {
		return ""
	}
	return fmt.Sprintf(`{ complete run="app complete %s"; }`, name)
}

// testHookNote gives a note to the flag --level and to every positional.
func testHookNote(cmd *kong.Node, v *kong.Value) string {
	switch {
	case v.Flag == nil:
		return fmt.Sprintf(`{ note "Positional of %s."; }`, cmd.Name)
	case v.Name == "level":
		return `{ note "Hook."; }`
	default:
		return ""
	}
}

// TestWriteSpec_valueKDL checks Config.ValueKDL is merged as a tag:
// in order, after UsageKDL of the type or of an ancestor and before the tag of the field.
func TestWriteSpec_valueKDL(tt *testing.T) {
	tt.Parallel()
	t := check.T(tt).MustAll()

	cfg := kongusage.Config{ValueKDL: []kongusage.ValueKDLFunc{testHookPredictor, testHookNote}}
	golden(t, "usagekdl_valuekdl", spec(t, &testValueHookCLI{}, cfg, kong.Description("App.")))
}

// testHookCallsCLI has every kind of a node and of a value.
type testHookCallsCLI struct {
	Verbose bool `help:"Verbose."`
	Run     struct {
		File string `arg:"" help:"File."`
	} `cmd:""           help:"Run."`
	Debug  testLeafCmd `cmd:"" hidden:"" help:"Debug."`
	Rename struct {
		Name struct {
			Name string      `       arg:"" help:"Old name."`
			To   testLeafCmd `cmd:""        help:"To."`
		} `arg:""`
	} `cmd:"" help:"Rename."`
}

// TestWriteSpec_hookCalls checks hooks are called once for every entity of the spec:
// Config.NodeKDL for the root and commands, Config.ValueKDL for flags and positionals.
func TestWriteSpec_hookCalls(tt *testing.T) {
	tt.Parallel()
	t := check.T(tt)

	var nodes, values []string
	cfg := kongusage.Config{
		NodeKDL: []kongusage.NodeKDLFunc{func(n *kong.Node) string {
			nodes = append(nodes, n.FullPath())
			return ""
		}},
		ValueKDL: []kongusage.ValueKDLFunc{func(cmd *kong.Node, v *kong.Value) string {
			kind := "positional"
			if v.Flag != nil {
				kind = "flag"
			}
			values = append(values, fmt.Sprintf("%s: %s %s", cmd.FullPath(), kind, v.Name))
			return ""
		}},
	}
	spec(t, &testHookCallsCLI{}, cfg)
	t.DeepEqual(nodes, []string{"app", "app run", "app debug", "app rename"})
	t.DeepEqual(values, []string{
		"app: flag help",
		"app: flag verbose",
		"app run: positional file",
		// Commands below a positional are not in the spec.
		"app rename <name>: positional name",
	})
}

// TestWriteSpec_hookErrors checks errors in usage features given by hooks:
// the hook named in messages, the format of the tag and the checks made after merging.
func TestWriteSpec_hookErrors(tt *testing.T) {
	tt.Parallel()

	node := func(path, kdl string) kongusage.NodeKDLFunc {
		return func(n *kong.Node) string {
			if n.FullPath() == path {
				return kdl
			}
			return ""
		}
	}
	nodes := func(hooks ...kongusage.NodeKDLFunc) kongusage.Config {
		return kongusage.Config{NodeKDL: hooks}
	}
	value := func(name, kdl string) kongusage.Config {
		return kongusage.Config{ValueKDL: []kongusage.ValueKDLFunc{func(_ *kong.Node, v *kong.Value) string {
			if v.Name == name {
				return kdl
			}
			return ""
		}}}
	}
	tests := []struct {
		name string
		cfg  kongusage.Config
		want string
	}{
		{"root", nodes(node("app", `{ effect read; }`)), `^UsageKDL: app: Config.NodeKDL\[0\]: effect is not accepted on the root$`},
		{
			"second hook", nodes(node("app run", `{ exit_code 3 "x"; }`), node("app run", `{ license MIT; }`)),
			`^UsageKDL: app run: Config.NodeKDL\[1\]: license is not accepted on commands$`,
		},
		{"syntax", nodes(node("app", `{`)), `^UsageKDL: app: Config.NodeKDL\[0\]: .`},
		{
			"form of UsageKDL of a command", nodes(node("app run", `cmd "" { exit_code 3 "x"; }`)),
			`^UsageKDL: app run: Config.NodeKDL\[0\]: it must hold properties and child nodes of one node$`,
		},
		{
			"form of UsageKDL of the root", nodes(node("app", `exit_code 3 "x"`)),
			`^UsageKDL: app: Config.NodeKDL\[0\]: it must hold properties and child nodes of one node$`,
		},
		{"entity not in spec", nodes(node("app", `{ cmd "nosuch"; }`)), `^UsageKDL: app: cmd "nosuch" is not in the spec$`},
		{"flag", value("force", `{ example x; }`), `^UsageKDL: app: flag --force: Config.ValueKDL\[0\]: example is not accepted on flags and positionals$`},
		{"positional", value("file", `{ example x; }`), `^UsageKDL: app run: positional file: Config.ValueKDL\[0\]: example is not accepted on flags and positionals$`},
		{"entity of a value", value("force", `{ flag "--env <ENV>"; }`), `^UsageKDL: app: flag --force: Config.ValueKDL\[0\]: flag is not accepted on flags and positionals$`},
		{"complete without value", value("force", `{ complete type=file; }`), `^UsageKDL: app: flag --force: complete is not accepted on a flag without a value$`},
		{"enum choice", value("env", `{ choices { choice test; }; }`), `^UsageKDL: app: flag --env: choices: choice "test" is not in the enum$`},
	}
	for _, tc := range tests {
		tt.Run(tc.name, func(tt *testing.T) {
			tt.Parallel()
			t := check.T(tt)

			t.PanicMatch(func() { spec(t, &testKDLCLI{}, tc.cfg) }, tc.want)
		})
	}
}

// TestCheckSpec_hooks checks a mistake in a hook fails the test:
// an error in its usage features and its panic.
func TestCheckSpec_hooks(tt *testing.T) {
	tt.Parallel()

	tests := []struct {
		name string
		cfg  kongusage.Config
		want string
	}{
		{
			"node error",
			kongusage.Config{NodeKDL: []kongusage.NodeKDLFunc{func(*kong.Node) string { return `{ help "x"; }` }}},
			`^UsageKDL: app: Config.NodeKDL\[0\]: help is not accepted$`,
		},
		{
			"node panic",
			kongusage.Config{NodeKDL: []kongusage.NodeKDLFunc{func(*kong.Node) string { panic("boom") }}},
			`(?s)^panic: boom\n.*TestCheckSpec_hooks`,
		},
		{
			"value error",
			kongusage.Config{ValueKDL: []kongusage.ValueKDLFunc{func(*kong.Node, *kong.Value) string { return `{ help "x"; }` }}},
			`^UsageKDL: app: flag --help: Config.ValueKDL\[0\]: help is not accepted$`,
		},
		{
			"value panic",
			kongusage.Config{ValueKDL: []kongusage.ValueKDLFunc{func(*kong.Node, *kong.Value) string { panic("boom") }}},
			`(?s)^panic: boom\n.*TestCheckSpec_hooks`,
		},
	}
	for _, tc := range tests {
		tt.Run(tc.name, func(tt *testing.T) {
			tt.Parallel()
			t := check.T(tt)

			checkFatal(t, checkSpec(t, &testConfigCLI{cfg: tc.cfg}), tc.want)
		})
	}
}
