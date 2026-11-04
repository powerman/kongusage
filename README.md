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

## Usage

Add a hidden flag to the Kong CLI struct:

```go
type CLI struct {
    // ...
    UsageSpec kongusage.Flag `hidden:"" help:"Print usage spec (KDL) of this CLI."`
}
```

Then the program prints its [usage spec](https://usage.jdx.dev/) in KDL by `--usage-spec`.

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
