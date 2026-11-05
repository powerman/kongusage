# usage

Behaviour of [usage](https://usage.jdx.dev/) 6.4.0–7.2.0 which kongusage depends on.
Undocumented behaviour is marked so: it may change in any release.

## Consumers of a spec

- Completion: `usage complete-word`, scripts of `usage generate completion`,
  and mise, which completes by the same code as the `usage` CLI of its usage-lib version.
- Documentation: `usage generate md`, `usage generate manpage`.
- Parsing: `usage explain`, scripts run by usage, mise tasks without `raw_args`.
- Compatibility: `usage diff` compares two specs by their parsing semantics
  and exits 1 on a breaking change.

usage-lib refuses a spec with an unknown key, so no consumer can use it at all.

## Mount

`mount run="<command>"` takes the spec of a command from the output of `<command>`.

- The command runs by `sh -c` in the current directory of the shell, with stdin closed,
  and with `__USAGE=<usage-lib version>` in its environment.
  `__USAGE` is documented for `complete run=` only (undocumented for `mount`).
- Only the root command of the mounted spec is merged into the mounting command
  (undocumented).
  Spec-level nodes have no effect there: `default_subcommand*`, `disable_help`,
  root `unknown_flags`, `unknown_env`, `version`, `about`, top-level `complete` and the like.
  Root flags, positionals, subcommands, properties of commands
  (`disable_help_flag`, `subcommand_required`, …), inline `complete` and `choices` work.

## Completion scripts

`usage generate completion <shell> <bin> --usage-cmd "<bin> --usage-spec"`
generates a script which runs the `--usage-cmd` command in the user's shell on every Tab,
without `__USAGE` in its environment.
With `--cache-key <key>` (e.g. the version of the CLI) the command runs once per key.
The spec is used as a root spec there, so spec-level nodes take effect.

## Relationship properties

`requires`, `overrides` and `conflicts` name one flag as a property (`requires="--b"`)
or several flags as a child node (`requires "--b" "--c"`, since 6.0.0).
A property naming several flags (`requires="--b --c"`) is taken as one bogus flag name
up to 7.1.4 and refused since 7.1.5.

## Completion of values

A value without its own `complete` and `choices` is completed by its value name,
in any case (undocumented):
`<dir>` completes directories, `<COMMAND>` commands on `PATH`,
`<HOSTNAME>` hosts, `<USERNAME>` users, `<URL>`, `<EMAIL>` and `<NONE>` nothing,
any other name files.
The name of a positional is used the same way (`arg "[url]"` completes nothing).

- `complete type="none"` completes nothing.
- `complete type="unknown"` completes files whatever the value name is.
- `complete type="dir"` completes directories, but files when no directory matches,
  contrary to the docs.
- `complete` inside `flag` and `arg` needs 6.12.0.

## Built-in flags and commands

- `-h`, `--help` and `-?` exist on every command;
  `disable_help` removes them everywhere, `disable_help_flag` on one command only.
- `-V` and `--version` exist on the root when the spec has `version`.
- 7.2.0 completes `-h`, `--help`, `-V` and `--version` unless declared flags use them.
- Terminal help of a command with subcommands lists a `help` subcommand
  which the parser refuses.
  `disable_help` does not remove it, contrary to the docs;
  `disable_help_subcommand` does, on one command only.
