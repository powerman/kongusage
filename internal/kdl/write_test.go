package kdl_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	gokdl "github.com/njreid/gokdl2"
	"github.com/powerman/check"

	"github.com/powerman/kongusage"
	"github.com/powerman/kongusage/internal/kdl"
	"github.com/powerman/kongusage/internal/usagetest"
)

// Strings which need escaping or sanitizing, with the strings they become.
var testStrings = []struct {
	name string
	in   string
	want string
	kdl  string
}{
	{"empty", "", "", `""`},
	{"plain", "text", "text", `"text"`},
	{"keyword", "true", "true", `"true"`},
	{"quote", `a"b`, `a"b`, `"a\"b"`},
	{"backslash", `a\b`, `a\b`, `"a\\b"`},
	{"slash", "a/b", "a/b", `"a/b"`},
	{"hash", "#a r#\"b\"#", "#a r#\"b\"#", `"#a r#\"b\"#"`},
	{"braces", "{a; b}", "{a; b}", `"{a; b}"`},
	{"newline", "a\nb", "a\nb", `"a\nb"`},
	{"tab", "a\tb", "a\tb", `"a\tb"`},
	{"non-ASCII", "é ∞ 😀", "é ∞ 😀", `"é ∞ 😀"`},
	{"CRLF", "a\r\nb", "a\nb", `"a\nb"`},
	{"CR", "a\rb", "a\nb", `"a\nb"`},
	{"VT FF", "a\vb\fc", "a\nb\nc", `"a\nb\nc"`},
	{"NEL", "a\u0085b", "a\nb", `"a\nb"`},
	{"LS PS", "a\u2028b\u2029c", "a\nb\nc", `"a\nb\nc"`},
	{"C0", "a\x00b\x1bc\bd", "abcd", `"abcd"`},
	{"DEL", "a\x7fb", "ab", `"ab"`},
	{"C1", "a\u0080b\u009bc", "abc", `"abc"`},
	{"BOM", "\uFEFFa\uFEFF", "a", `"a"`},
	{"bidi", "a\u200Eb\u200F\u202Ac\u202E\u2066d\u2069", "abcd", `"abcd"`},
	{"invalid UTF-8", "a\xffb\xe2\x82", "a\uFFFDb\uFFFD", "\"a\uFFFDb\uFFFD\""},
}

func write(t *check.C, nodes ...*kdl.Node) string {
	t.Helper()
	var buf bytes.Buffer
	err := kdl.Write(&buf, nodes)
	t.Must(t.Nil(err))
	return buf.String()
}

func TestWrite(tt *testing.T) {
	tt.Parallel()
	t := check.T(tt).MustAll()

	t.Equal(write(t), "")

	cmd := kdl.NewNode("cmd", kdl.String("run")).
		SetProp("help", kdl.String("Run.")).
		SetProp("hide", kdl.Bool(false)).
		SetProp("display_order", kdl.Int(2)).
		SetProp("hide", kdl.Bool(true))
	cmd.Comment = []string{"Comment.", "", "Multi\nline.  "}
	flag := kdl.NewNode("flag", kdl.String("-f --force"))
	flag.Comment = []string{"Nested."}
	arg := kdl.NewNode("arg", kdl.String("<file>")).AddChild(
		kdl.NewNode("choices", kdl.String("a"), kdl.String("b")),
		kdl.NewNode("exit_code", kdl.Int(-3), kdl.String("Text.")),
	)
	cmd.AddChild(flag, arg)
	nodes := []*kdl.Node{
		kdl.NewNode("min_usage_version", kdl.String("6.12.0")),
		kdl.NewNode("bin", kdl.String("app")),
		cmd,
		kdl.NewNode("z_last").SetProp("b", kdl.Int(0)).SetProp("a", kdl.Bool(false)),
	}
	t.Equal(write(t, nodes...), `min_usage_version "6.12.0"
bin "app"
// Comment.
//
// Multi
// line.
cmd "run" help="Run." hide=#true display_order=2 {
    // Nested.
    flag "-f --force"
    arg "<file>" {
        choices "a" "b"
        exit_code -3 "Text."
    }
}
z_last b=0 a=#false
`)
}

func TestWrite_strings(tt *testing.T) {
	tt.Parallel()

	for _, tc := range testStrings {
		tt.Run(tc.name, func(tt *testing.T) {
			tt.Parallel()
			t := check.T(tt).MustAll()

			n := kdl.NewNode("n", kdl.String(tc.in)).SetProp("p", kdl.String(tc.in))
			n.Comment = []string{tc.in}
			out := write(t, n)
			var want strings.Builder
			for line := range strings.SplitSeq(tc.want, "\n") {
				want.WriteString(strings.TrimRight("// "+line, " ") + "\n")
			}
			want.WriteString("n " + tc.kdl + " p=" + tc.kdl + "\n")
			t.Equal(out, want.String())

			// The parser of gokdl2 v0.6.1 takes U+FFFD for invalid UTF-8.
			if strings.ContainsRune(tc.want, '\uFFFD') {
				return
			}
			doc, err := gokdl.ParseWithOptions(strings.NewReader(out),
				gokdl.ParseOptions{Version: gokdl.ParseVersionV2})
			t.Nil(err)
			t.Len(doc.Nodes, 1)
			t.Equal(doc.Nodes[0].Arguments[0].ResolvedValueV2(), tc.want)
			p, _ := doc.Nodes[0].Properties.Get("p")
			t.Equal(p.ResolvedValueV2(), tc.want)
		})
	}
}

func TestWrite_names(tt *testing.T) {
	tt.Parallel()
	t := check.T(tt).MustAll()

	tests := []struct {
		name string
		kdl  string
	}{
		{"", `""`},
		{"a", `a`},
		{"é", `é`},
		{"a b", `"a b"`},
		{"1a", `"1a"`},
		{"-", `-`},
		{"+", `+`},
		{".", `.`},
		{"-.", `-.`},
		{"+-1", `+-1`},
		{"-.x", `-.x`},
		{"..5", `..5`},
		{"--flag", `--flag`},
		{"true1", `true1`},
		{"-nan", `-nan`},
		{"#a", `"#a"`},
		{"a=b", `"a=b"`},
		{"a/b", `"a/b"`},
		{`a"b`, `"a\"b"`},
		{`a\b`, `"a\\b"`},
		{"a\nb", `"a\nb"`},
		{"a\tb", `"a\tb"`},
	}
	for _, tc := range tests {
		n := kdl.NewNode(tc.name).SetProp(tc.name, kdl.Int(1))
		n.AddChild(kdl.NewNode(tc.name))
		out := write(t, n)
		t.Equal(out, tc.kdl+" "+tc.kdl+"=1 {\n    "+tc.kdl+"\n}\n", tc.name)

		doc, err := gokdl.ParseWithOptions(strings.NewReader(out),
			gokdl.ParseOptions{Version: gokdl.ParseVersionV2})
		t.Nil(err, tc.name)
		t.Equal(doc.Nodes[0].Name.ValueString(), tc.name)
		_, ok := doc.Nodes[0].Properties.Get(tc.name)
		t.True(ok, tc.name)
		t.Equal(doc.Nodes[0].Children[0].Name.ValueString(), tc.name)
	}
}

func TestWrite_panic(tt *testing.T) {
	tt.Parallel()
	t := check.T(tt)

	names := []string{
		"true", "false", "null", "inf", "-inf", "nan",
		"-1x", "+1", ".5", "-.5", "+.5",
		"a\x1bb", "a\rb", "a\u0085b", "a\u2028b", "a\uFEFFb", "a\u200Eb", "a\xffb",
	}
	for _, name := range names {
		t.PanicMatch(func() { write(t, kdl.NewNode(name)) }, `invalid KDL name`, name)
		t.PanicMatch(func() { write(t, kdl.NewNode("n").SetProp(name, kdl.Int(1))) },
			`invalid KDL name`, name)
		t.PanicMatch(func() { write(t, kdl.NewNode("n").AddChild(kdl.NewNode(name))) },
			`invalid KDL name`, name)
	}
	t.PanicMatch(func() { write(t, kdl.NewNode("n", kdl.Value{})) }, `invalid KDL value`)
	t.PanicMatch(func() { write(t, kdl.NewNode("n").SetProp("p", kdl.Value{})) },
		`invalid KDL value`)
}

// TestWrite_usage checks the written strings are what usage reads.
func TestWrite_usage(tt *testing.T) {
	tt.Parallel()
	t := check.T(tt).MustAll()

	nodes := make([]*kdl.Node, 0, 3+len(testStrings))
	nodes = append(nodes,
		kdl.NewNode("min_usage_version", kdl.String(kongusage.MinUsageVersion)),
		kdl.NewNode("bin", kdl.String("app")),
		kdl.NewNode("about", kdl.String("App.")),
	)
	for i, tc := range testStrings {
		n := kdl.NewNode("flag", kdl.String("--f"+string(rune('a'+i)))).
			SetProp("help", kdl.String("Help."))
		n.Comment = []string{tc.in}
		n.AddChild(kdl.NewNode("long_help", kdl.String("x"+tc.in+"x")))
		nodes = append(nodes, n)
	}
	spec := write(t, nodes...)

	usagetest.Lint(t, spec, kongusage.MinUsageVersion)
	for _, version := range []string{"", kongusage.MinUsageVersion} {
		out := usagetest.Output(t, version, strings.NewReader(spec), "generate", "json", "-f", "-")
		var got struct {
			Cmd struct {
				Flags []struct {
					HelpLong string `json:"help_long"`
				} `json:"flags"`
			} `json:"cmd"`
		}
		t.Nil(json.Unmarshal([]byte(out), &got))
		t.Len(got.Cmd.Flags, len(testStrings))
		for i, tc := range testStrings {
			t.Equal(got.Cmd.Flags[i].HelpLong, "x"+tc.want+"x", version, tc.name)
		}
	}
}
