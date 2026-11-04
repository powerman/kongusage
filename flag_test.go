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

	UsageSpec kongusage.Flag `hidden:""`
}

func TestFlag_BeforeReset(tt *testing.T) {
	tt.Parallel()
	t := check.T(tt).MustAll()

	var out bytes.Buffer
	exit := -1
	parser, err := kong.New(&testCLI{},
		kong.Name("test"),
		kong.Writers(&out, &out),
		kong.Exit(func(code int) { exit = code }),
	)
	t.Nil(err)
	_, _ = parser.Parse([]string{"--usage-spec"})
	t.Equal(exit, 0)
	t.Match(out.String(), `(?m)^flag --name `)
}
