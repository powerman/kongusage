// Package kongusage adds [usage] spec support for Kong CLI.
//
// A CLI declares a [Flag], which prints the spec of the CLI
// for a shell completion script, a mise task, docs or a compatibility check.
// [WriteSpec] can be used to generate the spec without using a [Flag].
//
// How Kong maps to the spec and which Kong features are lost is described in [docs/mapping.md].
//
// # Usage features
//
// Features of usage which have no counterpart in Kong are supplied by the author of the CLI
// in KDL v2, as nodes of the usage spec:
//
//   - by [KDLProvider] of the named root, command, flag or positional argument type;
//   - by the tag `usage-kdl` of the field of a command, a flag or a positional argument.
//
// UsageKDL of the root returns top-level nodes of the spec.
// UsageKDL of another type returns the node for a field of that type
// with an empty value, as the type does not know the name of the field:
// `cmd ""`, `flag ""` or `arg ""` (the last two are interchangeable)
// with properties and child nodes.
// Nested `cmd`, `flag` and `arg` nodes describe descendants of a node,
// named exactly as in the spec printed by [Flag],
// e.g. `flag "-f --force"` or `arg "<file>..."`;
// UsageKDL of the type of a field is ignored when an ancestor describes its node.
//
// Typical features, as UsageKDL of the root would give them:
//
//	license MIT
//	repository "https://github.com/example/app"
//	cmd "deploy" effect=destructive {
//	    help_md "Deploys the **current** release."
//	    example "app deploy --env=prod web1" header="Deploy to production"
//	    exit_code 3 "The deployment was rolled back."
//	    // The flag has placeholder:"ENV" and enum:"prod,dev".
//	    flag "--env <ENV>" long_help="Environment to deploy to." {
//	        note "Deploying to prod needs approval."
//	        choices { choice prod help="Production."; }
//	    }
//	    arg "<host>" { complete run="app hosts"; }
//	}
//
// The tag holds properties and child nodes of the node of its field,
// without its name and value, and merges with the node given by UsageKDL (if any):
//
//   - `usage-kdl:"effect=destructive"` adds the property or changes its value;
//   - `usage-kdl:"{ note \"Slow.\" }"` adds the child node
//     or replaces all child nodes of that name;
//   - `usage-kdl:"long_help=\"Text.\""` and `usage-kdl:"{ long_help \"Text.\" }"`
//     are the same node, as usage accepts it in either form;
//   - `usage-kdl:"effect=#null"` or `usage-kdl:"{ example #null }"` removes it;
//   - `usage-kdl:"{ flag \"--force\" effect=destructive }"` on a command
//     changes the node of its flag, keeping the rest of the node.
//
// Where to put features:
//
//   - UsageKDL of the root: features of the whole CLI, long texts,
//     and features of commands without a named type.
//   - UsageKDL of a reusable type: features which hold wherever the type is used,
//     e.g. `effect` of a command or `complete` of a value.
//   - The tag: short features of one field, and changes of the node of its type for it.
//
// For example:
//
//	type CLI struct {
//		Force bool     `help:"Overwrite files." usage-kdl:"effect=destructive"`
//		Run   RunCmd   `cmd:"" help:"Run a container."`
//		Rm    RmCmd    `cmd:"" help:"Remove a container." usage-kdl:"effect=destructive"`
//		Ls    struct{} `cmd:"" help:"List containers."`
//	}
//
//	func (CLI) UsageKDL() string {
//		return `
//	license MIT
//	flag "--force" long_help="Overwrites existing files without asking."
//	cmd "ls" effect=read
//	`
//	}
//
//	type RunCmd struct {
//		Image string `arg:"" help:"Image to run."`
//	}
//
//	func (RunCmd) UsageKDL() string {
//		return `cmd "" effect=write { example "app run alpine"; }`
//	}
//
//	type RmCmd struct {
//		ID ContainerID `arg:"" help:"Container." usage-kdl:"{ note \"Stops it first.\" }"`
//	}
//
//	type ContainerID string
//
//	func (ContainerID) UsageKDL() string {
//		return `arg "" { complete run="app ls"; }`
//	}
//
// The spec printed by [Flag] then has these nodes, besides other ones derived from Kong:
//
//	license MIT
//	flag "--force" help="Overwrite files." effect=destructive {
//	    long_help "Overwrites existing files without asking."
//	}
//	cmd "run" help="Run a container." effect=write {
//	    example "app run alpine"
//	    arg "<image>" help="Image to run."
//	}
//	cmd "rm" help="Remove a container." effect=destructive {
//	    arg "<id>" help="Container." {
//	        complete run="app ls"
//	        note "Stops it first."
//	    }
//	}
//	cmd "ls" help="List containers." effect=read
//
// Which nodes, values and properties are accepted
// and how exactly the tag changes the node is described in [docs/usage-features.md].
//
// A CLI must have a test which calls [CheckSpec], with the `usage` CLI installed,
// because [Flag] panics on a mistake in usage features.
//
// [usage]: https://usage.jdx.dev/
// [docs/mapping.md]: https://github.com/powerman/kongusage/blob/main/docs/mapping.md
// [docs/usage-features.md]: https://github.com/powerman/kongusage/blob/main/docs/usage-features.md
package kongusage
