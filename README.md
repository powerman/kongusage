# kongusage

[![License MIT](https://img.shields.io/badge/license-MIT-royalblue.svg)](LICENSE)
[![Go version](https://img.shields.io/github/go-mod/go-version/powerman/kongusage?color=blue)](https://go.dev/)
[![Test](https://img.shields.io/github/actions/workflow/status/powerman/kongusage/test.yml?label=test)](https://github.com/powerman/kongusage/actions/workflows/test.yml)
[![Coverage Status](https://raw.githubusercontent.com/powerman/kongusage/gh-badges/coverage.svg)](https://github.com/powerman/kongusage/actions/workflows/test.yml)
[![Release](https://img.shields.io/github/v/release/powerman/kongusage?color=blue)](https://github.com/powerman/kongusage/releases/latest)
[![Go Reference](https://pkg.go.dev/badge/github.com/powerman/kongusage.svg)](https://pkg.go.dev/github.com/powerman/kongusage)

![Linux | amd64 arm64 armv7 ppc64le s390x riscv64](https://img.shields.io/badge/Linux-amd64%20arm64%20armv7%20ppc64le%20s390x%20riscv64-royalblue)
![macOS | amd64 arm64](https://img.shields.io/badge/macOS-amd64%20arm64-royalblue)
![Windows | amd64 arm64](https://img.shields.io/badge/Windows-amd64%20arm64-royalblue)

kongusage generates a [usage spec](https://usage.jdx.dev/)
of a CLI built with [Kong](https://github.com/alecthomas/kong).
With the spec, the usage CLI provides shell completion of the program
for bash, zsh, fish, nu and PowerShell,
[mise](https://mise.jdx.dev/) completes arguments of tasks which run it,
and `usage` generates docs and checks compatibility between releases.
A flag added to the CLI is enough to print the spec;
features which Kong lacks, like examples or completion of values by a command,
are added to it in KDL.

## Usage

Add a flag to the Kong CLI struct:

```go
type CLI struct {
    // ...
    UsageSpec kongusage.Flag
}
```

With `kong.DefaultEnvars` add the tag `env:"-"` to the flag:
only the command line sets it, so help should not show a variable for it.

Then the program prints its [usage spec](https://usage.jdx.dev/) in KDL by `--usage-spec`.

### Usage-only features

The flag alone is enough to generate a spec, but the spec includes only what Kong knows.
Features of usage which Kong has no counterpart for,
e.g. examples, long help, notes or completion of values,
are given in KDL by the method `UsageKDL()` of the root, command, flag and positional types
and by the tag `usage-kdl` of their fields:

```go
type CLI struct {
    UsageSpec kongusage.Flag `env:"-"`
    Force     bool           `help:"Overwrite files." usage-kdl:"effect=destructive"`
    Deploy    DeployCmd      `cmd:"" help:"Deploy the release."`
    Hosts     HostsCmd       `cmd:"" help:"Print known hosts, one per line." hidden:""`
}

func (CLI) UsageKDL() string {
    return `
license MIT
cmd "deploy" {
    example "my-kong-cli deploy --env=prod web1" header="Deploy to production"
    flag "--env <ENV>" { note "Deploying to prod needs approval." }
}
`
}

type DeployCmd struct {
    Env  string `placeholder:"ENV" enum:"prod,dev" default:"dev" help:"Environment."`
    Host Host   `arg:""                                          help:"Host to deploy to."`
}

type Host string

func (Host) UsageKDL() string {
    return `arg "" { complete run="my-kong-cli hosts" }`
}
```

The [package doc](https://pkg.go.dev/github.com/powerman/kongusage#hdr-Usage_features)
explains the sources and where to put which feature;
[docs/usage-features.md](docs/usage-features.md) lists the accepted nodes
and how they are merged.
When the tag changes the node given by `UsageKDL()`, mind that:

- `long_help` and `help_md` are different nodes, so one does not replace the other;
  the same holds for `before_help` and `before_help_md`, `after_help` and `after_help_md`.
- `note` and `warning` are different nodes too.
- One `example` of the tag replaces all examples given by `UsageKDL()`,
  and so do `note`, `warning`, `exit_code` and `output`.
- `complete` is replaced as a whole, e.g. `complete type=file` drops `run` of the old one.

### Testing

Add a test which checks the spec, and install the [usage CLI](https://usage.jdx.dev/cli/)
where tests run (the test fails without it and names the version to install):

```go
func TestUsageSpec(t *testing.T) {
    kongusage.CheckSpec(t, kong.Must(&CLI{}))
}
```

The test is required for a CLI with usage-only features:
`--usage-spec` panics on a mistake in them, and `CheckSpec` finds it.
We recommend it for any CLI, as it also reports errors and warnings of `usage lint`
and logs each part of the CLI which the spec fixes silently,
e.g. a placeholder which usage cannot take as a value name;
but a CLI which only adds the flag can do without it.

Set `KONGUSAGE_TEST_SKIP_USAGE=1` where usage is not available (e.g. some CI platforms):
then the test does not run usage and checks only usage-only features.

## Mapping Kong to usage

[docs/mapping.md](docs/mapping.md) describes how Kong maps to the spec
and which Kong features are lost.

## Example: shell completion

The [usage CLI](https://usage.jdx.dev/cli/completions) generates a completion script
which reads the spec from the installed program:

```sh
usage generate completion zsh my-kong-cli --usage-cmd "my-kong-cli --usage-spec" --install
```

Replace `zsh` by `bash`, `fish`, `nu` or `powershell` for another shell.
The script runs `usage` and `my-kong-cli --usage-spec` on each Tab,
so completion always matches the installed version of the program,
and `usage` must be in `$PATH` wherever the script is used.

## Example: completion of a mise task

A [mise](https://mise.jdx.dev/) task which runs the program can take the spec for completion:

```toml
[tasks.my-kong-cli]
usage = 'mount run="mise run -q my-kong-cli --usage-spec"'
raw_args = true # Kong parses arguments, usage is only for completion.
run = 'my-kong-cli'
```

mise runs the `mount` command only to complete arguments of the task,
so the spec is never stale.
`raw_args = true` makes mise pass arguments to `run` as is instead of parsing them by the spec,
so `--help` and invalid flags reach Kong.

An alternative takes the spec from a file generated by `my-kong-cli --usage-spec`,
so `mise tasks info` and `mise generate task-docs` show it too:

```toml
[tasks.my-kong-cli]
usage = 'include file="{{ config_root }}/my-kong-cli.usage.kdl"'
raw_args = true
run = 'my-kong-cli'
```

The file must be regenerated whenever the program changes.
A missing or broken file makes the task fail
and silently breaks completion of every task of the project.

### Path completion

Completed paths are relative to the shell's directory,
while the task runs in its `dir`, by default the mise config root directory,
so a relative path completed in another directory is wrong for the program.
A program which takes paths from the user as a task has to resolve them
relative to `MISE_ORIGINAL_CWD` when it is set,
unless the task runs in the shell's directory (`dir = '{{cwd}}'`)
or the user runs the task only from the mise config root directory.

## Example: compatibility check

[`usage diff`](https://usage.jdx.dev/cli/) reports breaking changes of the command line
since the released version, and exits 1 when there are any:

```sh
my-kong-cli --usage-spec | usage diff released.usage.kdl -
```

A new version of kongusage may change the spec of an unchanged CLI,
which is reported as a change too.

## Alternatives

- [kong-usage](https://github.com/gaojunran/usage-integrations/tree/main/packages/kong-usage)
  also generates a usage spec of a Kong CLI.
  It is a minimal solution, enough for simple CLIs:
  it maps commands, flags and positional arguments with their basic properties.
  A few mappings its README lists are not emitted (`version`, `subcommand_required`),
  and flags of a command are not completed after its subcommand.

**kongusage** (this project) aims to be a complete one:
it describes as much of the behaviour of Kong as usage can express
(e.g. flag groups, default commands, inherited flags, completion of values by their type),
documents what is lost, takes usage-only features from the author of the CLI,
and checks the spec by `usage lint` in a test.

Shell completion of a Kong CLI without usage,
which neither the usage CLI nor mise tasks can use:

- [kongplete](https://github.com/willabides/kongplete): completion by posener/complete.
- [kong-completion](https://github.com/jotaen/kong-completion): completion for bash, zsh, fish.
