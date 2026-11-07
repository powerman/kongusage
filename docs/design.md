# Design

kongusage converts the Kong model of a CLI into a usage spec
as [mapping.md](mapping.md) describes.

The spec describes the CLI as Kong behaves, as exactly as the usage spec allows.
Parsing semantics matter as much as docs:
`usage diff` checks compatibility of two releases of the CLI by them.
A construct which breaks completion by a supported usage version is omitted,
even when it would describe the CLI more exactly:
no completion of a value is better than a wrong one.
When both express the same,
properties of commands are preferred to spec-level nodes, which a mise mount ignores.

A part of the Kong model which Kong takes but usage cannot,
e.g. a placeholder with whitespace, is fixed silently in the spec
rather than make the whole spec fail.
Such a fix may surprise the author of the CLI, and it usually means a small mistake
which the author would correct if told about it,
so `CheckSpec` logs each fix by `t.Logf`: what was fixed, why, and how.
A rule of the mapping which applies whatever the author does is not a fix and is not logged.
Mistakes in usage features are not fixed: they are errors.

## Usage version

usage-lib refuses a spec with an unknown node or property,
so one of them newer than the consumer breaks the whole spec, silently in mise.
The consumer is not chosen by the author of the CLI:
it is usage-lib of user's `mise` or `usage` tools.
Therefore the spec uses only nodes, values and properties
known to the oldest supported usage version
and sets `min_usage_version` to it,
so an older usage reports the version it needs before refusing the spec.
One added later, including one supplied by the author ([Usage features](usage-features.md)),
is not accepted until the oldest supported version reaches it.

## Supported versions

- Kong v1.17.0.
- usage versions released during the last three months,
  but not older than 6.12.0, which added `complete` of flags and positionals:
  usage-lib of mise releases and the `usage` CLI.
  Now these are usage-lib 6.12.0 (mise 2026.9.17) to 7.1.5 (mise 2026.11.0)
  and the `usage` CLI up to 7.2.0.
- Tests check every spec by the `usage` CLI of the oldest supported version
  and of the newest one in `mise.lock`.

## Lost features

Every Kong and usage feature is handled explicitly in the code.
A feature which cannot be expressed is handled at least by a no-op
naming the reason and the usage version.
Lost Kong features may later be emitted as KDL comments (`// kong: …`),
never as help text, which stays fully controlled by the author.

## mise tasks

A mise task takes the spec of the Kong CLI it runs (see README), with `raw_args = true`,
which leaves all parsing to Kong:

- by `usage = 'mount run="mise run -q <task> --usage-spec"'`:
  the spec is never stale, but mise uses it for completion only;
- or by `usage = 'include file="{{ config_root }}/<file>"'` from a generated file:
  mise also uses it for `mise tasks info` and `mise generate task-docs`,
  but the file must be kept up to date,
  and a missing or broken file breaks the task and completion of every task
  ([mise](mise.md#include)).

Either way:

- The mount command runs the CLI through `mise run`,
  because mise inserts its own global flags after the first word of the mount command
  ([mise](mise.md#mount-command)).
- Completion uses only the root command of the spec.
  Spec-level nodes still matter for an included spec, which docs and info show as a whole.
- kongusage does not assume a `dir` of the task,
  although completed relative paths are wrong when it differs from the shell's directory
  ([mise](mise.md#task-directory)),
  so a CLI has to resolve paths from the user relative to `MISE_ORIGINAL_CWD`.
