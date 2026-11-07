package kdl_test

import (
	"math"
	"testing"

	"github.com/powerman/check"

	"github.com/powerman/kongusage/internal/kdl"
)

func TestParse(tt *testing.T) {
	tt.Parallel()
	t := check.T(tt).MustAll()

	nodes, err := kdl.Parse(`
a "s" 1 0x10 1.5 1e2 #true #null z=1 b=2 z=3 {
    child; /- skipped
}
b
`)
	t.Nil(err)
	t.Len(nodes, 2)
	a := nodes[0]
	t.Equal(a.Name, "a")
	values := make([]any, 0, len(a.Values))
	for _, v := range a.Values {
		values = append(values, v.Interface())
	}
	t.DeepEqual(values, []any{"s", int64(1), int64(16), 1.5, 100.0, true, nil})
	t.DeepEqual(a.Props, []kdl.Prop{{Name: "b", Value: kdl.Int(2)}, {Name: "z", Value: kdl.Int(3)}})
	t.Len(a.Children, 1)
	t.Equal(a.Children[0].Name, "child")
	t.DeepEqual(nodes[1], &kdl.Node{Name: "b"})

	nodes, err = kdl.Parse("")
	t.Nil(err)
	t.Len(nodes, 0)
}

func TestParse_error(tt *testing.T) {
	tt.Parallel()
	t := check.T(tt)

	tests := []struct {
		in   string
		want string
	}{
		{`a "x`, `.`},
		{`a true`, `.`}, // KDL v1.
		{"a \"�\"", `invalid UTF8`},
		{`(t)a`, `^node a: type annotations are not supported: \(t\)$`},
		{`a (t)1`, `^node a: type annotations are not supported: \(t\)$`},
		{`a p=(t)1`, `^node a: property p: type annotations are not supported: \(t\)$`},
		{`a { b (t)1; }`, `^node a: node b: type annotations are not supported: \(t\)$`},
		{"a `1+1`", `^node a: expressions in backticks are not KDL v2$`},
		{`a 9223372036854775808`, `^node a: integer is out of range: 9223372036854775808$`},
	}
	for _, tc := range tests {
		_, err := kdl.Parse(tc.in)
		t.Match(err, tc.want, tc.in)
	}
}

func TestValue(tt *testing.T) {
	tt.Parallel()
	t := check.T(tt)

	parse := func(s string) kdl.Value {
		nodes, err := kdl.Parse("n " + s)
		t.Must(t.Nil(err))
		return nodes[0].Values[0]
	}
	tests := []struct {
		v      kdl.Value
		null   bool
		iface  any
		String string
	}{
		{kdl.String("a\"b"), false, "a\"b", `"a\"b"`},
		{kdl.Int(-3), false, int64(-3), "-3"},
		{kdl.Bool(true), false, true, "#true"},
		{kdl.Bool(false), false, false, "#false"},
		{parse("#null"), true, nil, "#null"},
		{parse("1.5"), false, 1.5, "1.5"},
		{parse("1e2"), false, 100.0, "100.0"},
		{parse("1e100"), false, 1e100, "1e+100"},
		{parse("#inf"), false, math.Inf(1), "#inf"},
		{parse("#-inf"), false, math.Inf(-1), "#-inf"},
	}
	for _, tc := range tests {
		t.Equal(tc.v.IsNull(), tc.null, tc.String)
		t.Equal(tc.v.Interface(), tc.iface, tc.String)
		t.Equal(tc.v.String(), tc.String)
	}
	nan := parse("#nan")
	t.Equal(nan.String(), "#nan")
	t.True(math.IsNaN(nan.Interface().(float64)))
	t.PanicMatch(func() { write(t, kdl.NewNode("n", parse("#null"))) }, `invalid KDL value`)
	t.PanicMatch(func() { write(t, kdl.NewNode("n", parse("1.5"))) }, `invalid KDL value`)
}
