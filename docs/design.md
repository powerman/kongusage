# Design

kongusage converts the Kong model of a CLI into a usage spec
as [mapping.md](mapping.md) describes.
The output depends on a profile and on a target usage version.

## Profiles

- **`spec`** (the default) describes the CLI as the usage documentation defines every construct,
  as exactly as the usage spec of the target version allows.
  It is meant to be correct for the spec itself rather than for a particular consumer,
  so every consumer of the spec ([usage](usage.md#consumers-of-a-spec)) gets the same CLI.
  Parsing semantics matter as much as docs:
  `usage diff` checks compatibility of two releases of the CLI by them.
- **`mise`** gives the best result in mise of the target version,
  for a task which mounts or includes the spec.
  It follows the observed behaviour of usage-lib of the target version,
  works around its bugs, and omits a construct whose completion would be wrong:
  no completion of a value is better than a wrong one.
  Spec-level nodes, which mise ignores, are still emitted
  as long as they do not break parsing of the spec.

## Target version

usage-lib refuses a spec with an unknown key,
so one key newer than the consumer breaks the whole spec.
Therefore a key added after the oldest supported usage version
is emitted only for target versions which know it.

- The low-level API takes the profile and the target version explicitly
  and reads no environment.
- The `--usage-spec` flag detects the target version,
  e.g. from `__USAGE`, which usage-lib sets for `mount` commands.
  `__USAGE` names neither the consumer nor the profile,
  and scripts of `usage generate completion` do not set it at all,
  so the profile is chosen explicitly.
- An unknown version is taken as the oldest supported one:
  it only loses keys added since, while the newest would break older consumers.
  A version below the supported range is taken as the oldest supported one,
  above it as the newest known one.

## Supported versions

- Kong v1.17.0.
- usage versions released during the last three months,
  but not older than 6.4.0, where completion of a mise task `mount` started:
  usage-lib of mise releases for `mise`, the `usage` CLI for `spec`.
  Now these are usage-lib 6.4.0 (mise 2026.8.13) to 7.1.5 (mise 2026.11.0)
  and the `usage` CLI up to 7.2.0.
- Development and tests use the `usage` CLI 7.1.3 and mise 2026.10.21 (usage-lib 7.1.3).

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

Both use the same `mise` profile:

- The mount command runs the CLI through `mise run`,
  because mise inserts its own global flags after the first word of the mount command
  ([mise](mise.md#mount-command)).
- Completion uses only the root command of the spec,
  so the `mise` profile prefers properties of commands to spec-level nodes.
  Spec-level nodes still matter for an included spec, which docs and info show as a whole.
- kongusage does not assume a `dir` of the task,
  although completed relative paths are wrong when it differs from the shell's directory
  ([mise](mise.md#task-directory)),
  so a CLI has to resolve paths from the user relative to `MISE_ORIGINAL_CWD`.
