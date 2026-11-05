# mise

Behaviour of [mise](https://mise.jdx.dev/) 2026.9.17–2026.11.0
for a task which mounts a usage spec:

```toml
[tasks.t]
usage = 'mount run="mise run -q t --usage-spec"'
raw_args = true
run = 't'
```

or includes it from a file ([Include](#include)):

```toml
[tasks.t]
usage = 'include file="{{ config_root }}/t.usage.kdl"'
raw_args = true
run = 't'
```

Undocumented behaviour is marked so: it may change in any release.

## Completion

- Completion scripts of `mise completion` call `mise __complete_word__`,
  which completes tasks by the code of `usage complete-word`
  of the same version as the embedded usage-lib.
  mise neither patches usage nor post-processes a mounted spec,
  so completion of a task is the completion of the `usage` CLI of that version,
  including [mount](usage.md#mount) limitations.
- Completion of an included spec uses only its root command and top-level `complete`
  (undocumented), as a mount does except for `complete`:
  `default_subcommand`, `version` and the like have no effect there.
- Only completion runs the mount command.
  `mise run <task> --help`, `mise tasks info`, `mise generate task-docs`, `mise usage`
  and parsing of the task's arguments do not,
  and `raw_args = true` passes all arguments to the task as is.
- A mount command which fails or prints a spec usage-lib cannot parse
  makes mise complete files, silently.
- Without root positionals in the mounted spec the task keeps mise's hidden catch-all
  `[ARGS]…`, so an unknown first word is accepted silently instead of being an error.
  An included spec always keeps it, after its root positionals (undocumented),
  so extra words after them are accepted silently too.

## Mount command

- The mount command runs on every Tab, without a cache.
  Its environment has `__USAGE` but nothing naming mise or its version.
- Global flags of mise typed after `run` are inserted after the first word of the mount
  command: `mise run -q t <Tab>` runs `mise -q run -q t --usage-spec`.
  A mount command starting with the task's binary would fail.

## Include

`include file="<path>"` in `usage` merges the whole spec from the file
into the spec of the task.

- The path must be absolute: `{{ config_root }}` and other templates are rendered
  in `usage` of a TOML task.
- The file is read on every use, without a cache,
  so it is stale only until it is generated again.
- Unlike a mount, the whole spec is used by `mise tasks info`, `mise generate task-docs`
  and, without `raw_args`, by `mise run <task> --help` and parsing of the task's arguments.
  With `raw_args = true` all arguments, including `--help`, still reach the task as is.
- mise sets `name` and `bin` of the spec to the task's name (undocumented).
  `about` and `long_about` replace the task's `description` in help and docs,
  while completion and `mise tasks ls` keep the `description`.
- A missing file or a spec usage-lib cannot parse in a TOML task (undocumented):
  the task fails to run, even with `raw_args = true`;
  `mise tasks info`, `mise generate task-docs` (for every task) and `mise tasks validate` fail;
  completion of every task and of task names falls back to files, silently.

## Task directory

The task runs in its `dir` (by default the config root),
while completion lists files of the shell's current directory,
so completed relative paths are wrong whenever the two differ.
`MISE_ORIGINAL_CWD` names the shell's directory inside the task.
`dir = "{{cwd}}"` runs the task in the shell's directory,
which only suits a task that does not depend on its directory.

## Completion of values

Measured with mise 2026.10.21 (usage-lib 7.1.3) in zsh and bash.

- Files (fallback or `complete type="unknown"`) are completed by the shell itself.
- Concrete candidates (`choices`, `type="file"`/`"dir"` matches, output of `complete run=`)
  are inserted badly: zsh adds no space after any of them, bash adds one even after `dir/`,
  and bash breaks values containing `=` (`--c=alpha` becomes `--c=--c=alpha`).
- `choices` match the typed prefix case-sensitively (`ignore_case` does not change this),
  cannot complete values starting with `-`,
  and cost about 4 µs per value of the whole spec on every Tab.
  Showing them in zsh takes about 10 ms for 1000 values and 0.7 s for 10000.
  The help of a `choice` is shown by zsh and fish, not by bash.
- `complete run=` runs by `sh -c` in the shell's directory with stdin closed,
  with `__USAGE` but without the `MISE_*` variables of the task,
  and since mise 2026.10.19 only for a trusted config.
  It adds about 2 ms to a Tab, or about 77 ms when it calls `mise run`.
  An empty output or a failing command makes mise complete files.
