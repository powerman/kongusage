package kongusage_test

import (
	"strings"
	"testing"

	"github.com/powerman/check"

	"github.com/powerman/kongusage"
	"github.com/powerman/kongusage/internal/usagetest"
)

// TestDefaultSubcommandOnEmpty fails once default_subcommand_on_empty stops breaking
// root completion: then docs/mapping.md "Default commands" is to be revised.
func TestDefaultSubcommandOnEmpty(tt *testing.T) {
	tt.Parallel()
	t := check.T(tt)

	const spec = `
name "app"
bin "app"
default_subcommand "serve"
default_subcommand_on_empty #true
cmd "serve" {
    arg "[mode]" {
        choices "fast" "slow"
    }
}
cmd "other"
`
	for _, version := range []string{"", kongusage.MinUsageVersion} {
		got := usagetest.Output(t, version, strings.NewReader(spec), "complete-word", "-f", "-", "--", "app", "")
		// usage 6.12.0–7.3.0 offer the arguments of the default command instead of commands.
		t.Equal(got, "fast\nslow\n", "usage %q is fixed", version)
	}
}
