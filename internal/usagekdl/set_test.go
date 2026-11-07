package usagekdl_test

import (
	"testing"

	"github.com/powerman/check"

	"github.com/powerman/kongusage/internal/usagekdl"
)

// where names the entity and the source in tests of parsing.
var where = []string{"app", "src"}

func TestParseRoot_error(tt *testing.T) {
	tt.Parallel()
	t := check.T(tt)

	tests := []struct {
		name string
		kdl  string
		want string
	}{
		{"syntax", `about_md "x`, `^UsageKDL: app: src: .`},
		{"unknown", `help "x"`, `^UsageKDL: app: src: help is not accepted$`},
		{"not at root", `effect "read"`, `^UsageKDL: app: src: effect is not accepted on the root$`},
		{"not on cmd", `cmd "run" { note "x"; }`, `^UsageKDL: app: src: cmd "run": note is not accepted on commands$`},
		{"not on flag", `flag "--x" { example "x"; }`, `^UsageKDL: app: src: flag "--x": example is not accepted on flags and positionals$`},
		{"child only", `flag "--x" note="x"`, `: flag "--x": note is not accepted as a property$`},
		{"both forms", `cmd "run" effect=read { effect write; }`, `: cmd "run": effect is given twice$`},
		{"single twice", `license "a"; license "b"`, `: license is given twice$`},
		{"removed and given", `example #null; example "app run x"`, `: example is given twice$`},
		{"entity twice", `cmd "run"; cmd "run"`, `: cmd "run" is given twice$`},
		{"entity on flag", `flag "--force" { flag "--x"; }`, `: flag "--force": flag is not accepted on flags and positionals$`},
		{"entity without name", `cmd`, `: cmd needs one value: its name in the spec$`},
		{"entity with number", `cmd 1`, `: cmd needs a string value: its name in the spec$`},
		{"too few values", `exit_code 1`, `: exit_code: 1 values given, it takes 2 values$`},
		{"too many values", `license "a" "b"`, `: license: 2 values given, it takes one value$`},
		{"no values", `flag "--x" { complete x; }`, `: complete: 1 values given, it takes no values$`},
		{"no rest", `available_if`, `: available_if: 0 values given, it takes at least 1 values$`},
		{"string", `license 1`, `: license: 1 is not a string$`},
		{"boolean", `next_line_help "yes"`, `: next_line_help: "yes" is not a boolean$`},
		{"integer", `exit_code 1.0 "x"`, `: exit_code: 1.0 is not an integer$`},
		{"float", `term_width 1e2`, `: term_width: 100.0 is not a non-negative integer$`},
		{"negative", `term_width -1`, `: term_width: -1 is not a non-negative integer$`},
		{"null value", `exit_code #null "x"`, `: exit_code: #null is not an integer$`},
		{"enum", `cmd "run" effect=delete`, `: cmd "run": effect: "delete" is not one of read, write, destructive$`},
		{"prop enum", `output "x" framing="xml"`, `: output: framing: "xml" is not one of text, json, jsonl$`},
		{"prop", `example "app run" title="x"`, `: example: property title is not accepted$`},
		{"prop null", `example "app run" help=#null`, `: example: help: #null is not a string$`},
		{"child", `example "app run" { lang "sh"; }`, `: example: child node lang is not accepted$`},
		{"choices child", `flag "--x" { choices { value "a"; }; }`, `: choices: child node value is not accepted$`},
		{"choice twice", `flag "--x" { choices { choice a; choice a; }; }`, `: choices: choice "a" is given twice$`},
		{"choice null", `flag "--x" { choices { choice #null; }; }`, `: choices: choice: #null is not a string$`},
	}
	for _, tc := range tests {
		t.PanicMatch(func() { usagekdl.ParseRoot(where, tc.kdl) }, tc.want, tc.name)
	}
}

func TestParseValue_error(tt *testing.T) {
	tt.Parallel()
	t := check.T(tt)

	tests := []struct {
		name string
		kdl  string
	}{
		{"cmd", `cmd ""`},
		{"named", `flag "--x"`},
		{"two nodes", `flag ""; arg ""`},
	}
	for _, tc := range tests {
		t.PanicMatch(func() { usagekdl.ParseValue(where, tc.kdl) },
			`^UsageKDL: app: src: it must return one node flag "" or arg ""$`, tc.name)
	}
}

func TestParseTag_error(tt *testing.T) {
	tt.Parallel()
	t := check.T(tt)

	tests := []struct {
		name string
		kdl  string
		want string
	}{
		{"syntax", `{`, `^UsageKDL: app: src: .`},
		{"value", `x`, `^UsageKDL: app: src: it must hold properties and child nodes of one node$`},
		{"two nodes", `; note x`, `^UsageKDL: app: src: it must hold properties and child nodes of one node$`},
	}
	for _, tc := range tests {
		t.PanicMatch(func() { usagekdl.ParseTag(where, tc.kdl, usagekdl.AtArg) }, tc.want, tc.name)
	}
}

func TestSet_CheckNoValue(tt *testing.T) {
	tt.Parallel()
	t := check.T(tt)

	tests := []struct {
		tag  string
		want string
	}{
		{``, ``},
		{`{ note x; }`, ``},
		{`{ choices a strict=#false; }`, `^UsageKDL: app: flag --x: choices is not accepted on a flag without a value$`},
		{`{ complete type=file; }`, `^UsageKDL: app: flag --x: complete is not accepted on a flag without a value$`},
	}
	for _, tc := range tests {
		s := usagekdl.ParseTag(where, tc.tag, usagekdl.AtArg)
		call := func() { s.CheckNoValue([]string{"app", "flag --x"}) }
		if tc.want == "" {
			t.NotPanic(call, tc.tag)
		} else {
			t.PanicMatch(call, tc.want, tc.tag)
		}
	}
}

func TestSet_CheckValue(tt *testing.T) {
	tt.Parallel()
	t := check.T(tt)

	enum := []string{"prod", "dev"}
	tests := []struct {
		tag  string
		enum []string
		want string
	}{
		{``, nil, ``},
		{``, enum, ``},
		{`{ choices a strict=#false; }`, nil, ``},
		{`{ choices strict=#false { choice a; }; }`, nil, ``},
		{`{ choices { choice prod; }; }`, enum, ``},
		{`{ choices a b; }`, nil, `^UsageKDL: app: flag --x: choices: strict=#false is required without an enum, as Kong accepts any value$`},
		{`{ choices a b strict=#true; }`, nil, `: choices: strict=#false is required without an enum`},
		{`{ choices strict=#false; }`, nil, `^UsageKDL: app: flag --x: choices: values or choice nodes are required$`},
		{`{ choices prod; }`, enum, `^UsageKDL: app: flag --x: choices: only choice nodes are accepted with an enum, which gives the values$`},
		{`{ choices strict=#false { choice prod; }; }`, enum, `: choices: only choice nodes are accepted with an enum`},
		{`{ choices { choice test; }; }`, enum, `^UsageKDL: app: flag --x: choices: choice "test" is not in the enum$`},
	}
	for _, tc := range tests {
		s := usagekdl.ParseTag(where, tc.tag, usagekdl.AtArg)
		call := func() { s.CheckValue([]string{"app", "flag --x"}, tc.enum) }
		if tc.want == "" {
			t.NotPanic(call, tc.tag)
		} else {
			t.PanicMatch(call, tc.want, tc.tag)
		}
	}
}
