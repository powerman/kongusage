package kongusage

import (
	"fmt"

	"github.com/alecthomas/kong"
	upstream "github.com/gaojunran/usage-integrations/packages/kong-usage"
)

// Flag prints the usage spec of the CLI in KDL and exits, like [kong.VersionFlag].
//
// Declare it as a field with the tag `hidden:""`, so it doesn't clutter the help.
// The name of the flag comes from the name of the field
// (e.g. field UsageSpec gives flag --usage-spec).
//
// In a mise task completed paths are relative to the shell's directory,
// so the program has to resolve paths from the user
// relative to MISE_ORIGINAL_CWD when it is set (see README).
type Flag bool

// BeforeReset implements the Kong hook.
func (Flag) BeforeReset(app *kong.Kong) error {
	_, err := fmt.Fprint(app.Stdout, upstream.GenerateKDL(app))
	if err != nil {
		return err
	}
	app.Exit(0)
	return nil
}
