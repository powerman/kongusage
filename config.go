package kongusage

import (
	"reflect"

	"github.com/alecthomas/kong"
)

// Design of the hooks NodeKDL and ValueKDL.
//
// A hook exists so that the same UsageKDL or tag isn't repeated on every command or value,
// e.g. exit statuses which a library computes from the Kong model of the CLI.
// It is a computed tag, not a new lowest layer of features under UsageKDL:
//
//   - UsageKDL belongs to a type, so a description by an ancestor replaces it as a whole:
//     the author of a CLI knows better than the author of a library type what the node is.
//     The tag belongs to one instance, the field, so it changes the node instead.
//     A hook is called for one instance too, a node of the Kong model,
//     so it is merged like the tag, right before it.
//   - The lowest layer was tried in a prototype and had worse rules.
//     Its nested `cmd`/`flag`/`arg` nodes silently turned off UsageKDL of the type
//     of the descendant, so the lowest layer disabled a higher one.
//     A hook couldn't change a feature of a type for one place.
//     And the order "ancestor's description over the hook" was hard to explain.
//     As a tag, nested nodes and #null need no special rules:
//     they work as they already do in the tag.
//   - There is no check that a hook overrides nothing (e.g. `exit_code` of UsageKDL):
//     the hook, UsageKDL and the tag of a CLI have one author,
//     and writing in UsageKDL what one's own hook replaces is a mistake of that author.
//     A hook should return correct data by itself rather than be fixed by UsageKDL or the tag.
//   - One form, the form of the tag, for the root and for a command,
//     so one hook serves both without branching, and no separate hook for the root is needed.
//     The forms of UsageKDL (`cmd ""`, top-level nodes) are refused in a hook:
//     they would mix two styles of results for one task.
//   - A slice, not one function which the author of a CLI combines from several.
//     Hooks of different sources don't know each other and may give the same nodes,
//     so their results have to be merged, and gluing texts would merge them arbitrarily.
//     Applying one more "tag" costs nothing and reuses the merge rules of the tag:
//     the author of a CLI may put their own hook after a hook of a library
//     to patch it by the merge syntax of the tag,
//     instead of editing KDL as text or changing the library.
//   - ValueKDL gets *kong.Value for both flags and positional arguments,
//     not separate FlagKDL and PositionalKDL:
//     both accept the same features (one place AtArg), and most hooks look at a struct tag
//     or the type of the value, so they would have to be registered twice.
//     The names and properties of flags and positionals differ, but kongusage derives them.
//     Separate fields can be added compatibly if kongusage starts accepting
//     features of flags only (e.g. `require_equals`).
//   - ValueKDL also gets the command which declares the value:
//     kong.Value has no link to its node, and a hook may need the path of the command
//     or the name of the program (e.g. for `complete run="app complete hosts"`).

// Config controls generation of the usage spec.
// The zero value of every field means "unset, use default".
// Usage features of the CLI are supplied by [KDLProvider] and the tag `usage-kdl`;
// the hooks NodeKDL and ValueKDL give features in the format of the tag
// where writing the KDLProvider or the tag would repeat.
type Config struct {
	// NoDisplayOrder disables `display_order` of commands,
	// which keeps the order of their declaration in help of usage;
	// without it help of usage sorts commands by name.
	NoDisplayOrder bool

	// NodeKDL are hooks called for the root and for every command:
	// each one returns usage features in the format of the tag `usage-kdl`.
	// They are applied in order, after UsageKDL and before the tag of the field
	// (see "Usage features" in the package doc).
	NodeKDL []NodeKDLFunc

	// ValueKDL are hooks called for every flag and positional argument,
	// which return usage features and are applied like NodeKDL.
	ValueKDL []ValueKDLFunc
}

// NodeKDLFunc returns usage features of the root or the command n
// in the format of the tag `usage-kdl`: properties and child nodes of its node,
// without its name and value, e.g. `effect=read { note "Slow." }`, or "" for none.
type NodeKDLFunc func(n *kong.Node) string

// ValueKDLFunc returns usage features of the flag or the positional argument v of cmd
// in the format of the tag `usage-kdl`, e.g. `{ complete run="app hosts" }`, or "" for none.
// v.Flag is nil for a positional argument.
type ValueKDLFunc func(cmd *kong.Node, v *kong.Value) string

// ConfigProvider may be implemented by the root of a Kong CLI
// to give [Flag] and [CheckSpec] a [Config].
type ConfigProvider interface {
	UsageConfig() Config
}

// resolve applies defaults to c, normalizes and validates it.
func (c Config) resolve() (Config, error) {
	return c, nil
}

// rootConfig returns the Config of the root of k, or zero Config.
func rootConfig(k *kong.Kong) Config {
	if cli, ok := reflect.TypeAssert[ConfigProvider](k.Model.Target.Addr()); ok {
		return cli.UsageConfig()
	}
	return Config{}
}
