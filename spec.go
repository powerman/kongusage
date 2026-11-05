package kongusage

import "github.com/alecthomas/kong"

// KDLProvider may be implemented by the root of a Kong CLI
// and by the type of a command, a flag or a positional argument
// to supply usage features in KDL (see "Usage features" in the package doc).
type KDLProvider interface { //nolint:iface // TODO: Used by WriteSpec once implemented.
	UsageKDL() string
}

// CheckSpec fails the test t on mistakes in the usage spec of the CLI k:
//
//   - on mistakes in usage features which make [WriteSpec] and thus [Flag] panic,
//     e.g. a node, value or property which is not accepted,
//     a name of a command, a flag or a positional argument which is not in the spec,
//     or a node derived from Kong (see "Usage features" in the package doc);
//   - on errors and warnings of `usage lint`, e.g. an example which the spec cannot parse.
//
// Use it in a test of the CLI:
//
//	func TestUsageSpec(t *testing.T) {
//		kongusage.CheckSpec(t, kong.Must(&CLI{}))
//	}
//
// The spec is generated with the same [Config] as by [Flag].
// CheckSpec logs each part of the CLI which the spec fixes silently,
// e.g. a placeholder which usage cannot take as a value name,
// as it is likely a mistake which is worth correcting in the CLI.
// It requires the `usage` CLI in PATH, of a version not older than `min_usage_version`;
// otherwise the test fails with the version to install.
func CheckSpec(t interface {
	Helper()
	Logf(format string, args ...any)
	Fatalf(format string, args ...any)
}, k *kong.Kong,
) {
	t.Helper()
	_ = k
	// TODO:
	//   - Run `usage --version`; fail when usage is not in PATH
	//     or is older than min_usage_version.
	//   - Write the spec of k as WriteSpec does, with rootConfig(k),
	//     and log each note of the converter by t.Logf.
	//     Turn a panic of WriteSpec into a failure of t.
	//   - Pipe the spec to `usage lint --warnings-as-errors -`; fail on its non-zero
	//     exit status with its output.
	//     Warnings fail too while kongusage is young; drop that if it fails in practice.
	panic("TODO")
}
