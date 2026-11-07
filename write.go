package kongusage

import (
	"io"

	"github.com/alecthomas/kong"

	"github.com/powerman/kongusage/internal/kdl"
)

// WriteSpec writes the usage spec of the CLI k to w in KDL.
//
// It panics on an invalid cfg or on usage features of k it cannot merge
// (use [CheckSpec] in your tests to ensure this does not happen).
// Returns an error when writing to w fails.
func WriteSpec(w io.Writer, k *kong.Kong, cfg Config) error {
	cfg, err := cfg.resolve()
	if err != nil {
		panic(err)
	}

	// TODO: Merge usage features (stage 5):
	//   - Build the tree of usage features as docs/usage-features.md "Merge" describes:
	//     UsageKDL of the root, then top-down for every field UsageKDL of its type
	//     unless an ancestor has described its node, then its tag merged into the node
	//     by the rules of docs/usage-features.md, not by those of usage:
	//     SpecCommand::merge in lib/src/spec/cmd.rs is only a hint for undecided cases.
	//     Parse KDL v2 only and emit values anew by the type of each node and property:
	//     panic on a syntax error, a type annotation, an entity missing in Kong,
	//     a node, value or property not accepted, a node given in both forms
	//     or a single node given twice in one source.
	//     Leave the rest of validation to `usage lint` (CheckSpec).
	//     Find UsageKDL of a command type as Kong finds Help (build.go): on the address
	//     of the field, with methods promoted from embedded structs, which is how a reusable
	//     command is usually embedded; panic when Go drops an ambiguous promoted method,
	//     i.e. the type does not implement it while several embedded fields do.
	//     Call UsageKDL of a value type on reflect.New(T), so that a nil pointer is not used.
	//     Nodes only the root has are ignored on other nodes so that the root of one CLI may
	//     be a command of another one, as usage drops them on mount;
	//     about_md becomes help_md, as usage turns about of a plugin into help.
	//   - Add the tree to the spec derived from Kong.
	return kdl.Write(w, convert(k, cfg))
}
