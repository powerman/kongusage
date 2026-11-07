# Mapping of Kong to usage spec

How kongusage maps a [Kong](https://github.com/alecthomas/kong) CLI
to a [usage spec](https://usage.jdx.dev/spec/), and which Kong features are lost.
It is correct for Kong v1.17.0, usage 6.12.0–7.3.0 and mise 2026.9.17–2026.10.24.

The input is the Kong model: the `kong.Application` tree built by `kong.New`.
The output is a spec for minimal supported usage version (6.12.0).
Behaviour of usage and mise referred to below is described in [usage](usage.md)
and [mise](mise.md).

## Applied by Kong

Kong applies these before the model is read, so they need no mapping:

- tags `-`, `name`, `prefix`, `envprefix`, `xorprefix`, `embed`, `set`,
  `kong.Signature`, `kong.Plugins` and embedded interfaces;
- interpolation of `${…}` into `help`, `default`, `enum`, `env` and `placeholder`
  (`Node.Detail` is not interpolated and is mapped as is);
- options `Name`, `Description`, `Vars`, `Embed`, `DynamicCommand`, `IgnoreFields`,
  `FlagNamer`, `DefaultEnvars`, `AutoGroup`, `Groups`, `ExplicitGroups`, `PostBuild`,
  `TypeMapper`, `KindMapper`, `ValueMapper`, `NamedMapper`.

## Application

- `Model.Name` → `name` and `bin`.
  Kong takes it from `kong.Name()` or else from `os.Args[0]`,
  so `go run` gives a temporary name.
- `Model.Help` (`kong.Description()`) → `about`.
- `Model.Detail` (the root implements `kong.HelpProvider`) → `long_about`.
- Flags and positionals of the root → top-level `flag` and `arg`.
- `kong.Vars{"version": …}` → `version`, see [Version flag](#version-flag).

## Commands

- Command → `cmd "<name>"`; `Help` → `help`; `Detail` → `long_help`.
- `aliases:"a,b"` → `alias "a" "b"`, visible as in Kong's help.
- `hidden:""` → `hide=#true`.
- Order of declaration → `display_order=<index>` among siblings, unless disabled in `Config`:
  terminal help of usage otherwise sorts commands by name.
- A required subcommand → `subcommand_required=#true`.
  Kong requires one when the node has a non-hidden child command, no default command,
  and no `Run` method (on the value or on its address).
- `default:"1"`, `default:"withargs"`: see [Default commands](#default-commands).
- `passthrough:""`: see [Passthrough](#passthrough).
- Positional with subcommands:
  see [Branching positional arguments](#branching-positional-arguments).

## Flags

- Flag → `flag "-<short> --<name> <VALUE>"`; for value names see [Value names](#value-names).
  The value is omitted for bool and counter flags, detected by `Value.IsBool()`
  as Kong's help does
  (parsing ignores `IsBool()`, so a custom type reporting it may still take a value).
- `help:"…"` → `help`.
  The model holds the text without the `($ENV)` suffix, which Kong adds when it prints help.
- `${default}`, `${enum}`, `${env}` in `help` (`Value.OrigHelp`) →
  `hide_default_value=#true`, `hide_possible_values=#true`, `hide_env=#true`:
  otherwise terminal help of usage repeats the value after the text
  (its Markdown docs show it anyway).
- `required:""` → `required=#true`; `optional:""` → nothing.
- `default:"X"` → `default="X"`, or `default=#true`/`#false` for a bool flag.
- `hidden:""` → `hide=#true`.
- `negatable:""` → `negate="--no-<name>"`; `negatable:"X"` → `negate="--X"`.
  `usage generate md` does not show the negation.
- `type:"counter"` → `count=#true`.
- `enum:"a,b"` → `choices "a" "b"` inside the flag, together with `default` if any.
  On a slice flag both Kong and usage check every value after splitting.
- `env:"A,B"` → `env="A"` and `env_fallback "B"`.
  Both take the first variable which is set, even when it is empty.
  `env:"-"` (stays in the model without `DefaultEnvars`) → nothing.
  `kong.VersionFlag` → no `env`: Kong prints the version for any value of its variable,
  even `false`, which `DefaultEnvars` makes likely (e.g. `APP_VERSION`).
- `and:"g"` → `requires` on every member naming every other member,
  by a child node when there are several (`requires "--b" "--c"`).
  `required:""` on any member makes Kong require every member,
  so every member gets `required=#true`.
  A flag in several `and` groups is required by Kong only if the last group requires it
  (its README implies every group): the spec follows the model.
- `lastwins:"g"` → `overrides` on every member naming every other member.
- `xor:"g"`, `lastwins:"g"` with `required:""`:
  see [Flag relationship groups](#flag-relationship-groups).
- `aliases:"f,flg"`: see [Flag aliases](#flag-aliases).
- `WithHyphenPrefixedParameters(true)` → `allow_hyphen_values=#true`
  on every flag which takes a value.
  Without the option nothing is emitted, although usage accepts a negative number
  as a value (`--num -1`), which Kong refuses.
- Help and version flags: see [Help flag](#help-flag) and [Version flag](#version-flag).

## Positional arguments

- Positional → `arg "<name>"` when required, `arg "[name]"` when optional.
  Kong positionals are required unless `optional:""` or `default:"…"` is set.
- Slice or map positional → `arg "<name>..."` or `arg "[name]..."`, without `delimiter`:
  Kong does not split positionals by `sep`, contrary to its README.
- `help`, `default`, `env`, `enum` → as for flags.
- `hidden:""`: see [Hidden positional arguments](#hidden-positional-arguments).

## Values

- A slice or map is detected by the kind of the Go type,
  because `Value.IsSlice()` is false for a named slice type (`type Tags []string`),
  which Kong still accumulates and splits.
  See [Slice and map flags](#slice-and-map-flags).
- Map flag → `var=#true`, value name `KEY=VALUE`.
- `type:"filecontent"` and `kong.FileContentFlag` are not slices:
  Kong reads a whole file into the `[]byte`.
- Custom mappers and `MapperValue` types → a flag with a value,
  or a bool flag when `IsBool()` reports so.
- Completion: see [Completion of values](#completion-of-values).

## Mappings with alternatives

### Flags inherited by subcommands

Kong accepts the flags of every ancestor after a subcommand (`my-kong-cli sub --verbose`)
and checks their `required`, `xor` and `and` on every command of the path.

Chosen: `global=#true` on every flag of every command which has subcommands,
not only of the root; flags of leaf commands stay plain.

```kdl
flag "-v --verbose" global=#true
cmd "sub" { flag "-l --leaf <leaf>" }
```

usage then accepts and completes the flag after a subcommand of its command only,
lists it under "Global flags" in help of descendants, and checks its requirements.
A global flag of a nested command is not global at the root:
`x --af` is refused, `x a b c --af` accepted, as in Kong.

Rejected:

- Plain flags of the parent:
  usage accepts them only before the subcommand,
  and does not check their `required` and `requires` once a subcommand is selected (a bug).
- A copy of the flags in every descendant: duplicates docs, loses "Global Flags".

### Slice and map flags

Kong splits every occurrence of a slice flag by `sep` (default `,`, `none` disables,
`\,` escapes) and of a map flag by `mapsep` (default `;`), and appends the values.

Chosen: `var=#true delimiter="<sep>"`, without `delimiter` for `none`.

```kdl
flag "-t --tags <TAGS>" var=#true delimiter=","
flag "-m --map <KEY=VALUE>" var=#true delimiter=";"
```

usage splits values the same way, including the value of an environment variable
(contrary to its docs), except for escaping.
Completion after a delimiter (`-t a,<Tab>`) offers nothing.

Rejected: no `delimiter`: `choices` would refuse `a,b`, which Kong accepts.

### Bool flags

Kong accepts `--flag=<v>` with `true`, `1`, `yes`, `false`, `0`, `no` in any case.

Chosen: `bool_value=#true` on every bool flag.

```kdl
flag "--color" negate="--no-color" bool_value=#true
```

usage then accepts `--flag=true` and `--flag=false` (inverted for the `negate` spelling,
as in Kong) in addition to a bare `--flag`; help and completion do not change.

Rejected: a plain bool flag: usage silently drops the value of `--flag=false` and binds true.

### Passthrough

`passthrough:""` (mode `all`) or `passthrough:"partial"` on a positional slice
stops parsing of flags once its first value is reached and keeps a later `--` as a value.
`passthrough:""` on a command makes its only `[]string` positional take everything
after the command name, including `--help` and `--`.

Chosen: `double_dash="automatic"` on the positional,
which is also every positional of a passthrough command.
`disable_help_flag=#true` on every command ([Help flag](#help-flag))
keeps usage from taking a leading `--help` of a passthrough command as its built-in help,
but see [Passthrough commands](#passthrough-commands).

```kdl
cmd "exec" disable_help_flag=#true {
  arg "[args]..." double_dash="automatic"
}
```

Its documented meaning is Kong's, and completion stops offering flags after the first value.
usage drops the first `--` after the first value (undocumented);
later ones are kept since 7.0.0 and dropped too before it.

Rejected: `double_dash="preserve"`:
keeps `--` as a value, but flags are still parsed and completed after the first value.

### Default commands

`default:"1"` on a command selects it when nothing follows its parent;
Kong takes any tag value except `withargs` as `1`, so `Node.DefaultCmd` is read.
`default:"withargs"` also accepts the flags and arguments of the default command
without its name.
Kong allows a default command under any command, usage only at the root.

Chosen at the root:

```kdl
default_subcommand "serve"
default_subcommand_flags #true // only for default:"withargs"
```

usage routes an unknown first word to it
(with `default_subcommand_flags`, also a flag of the default command),
marks it `(default)` in help, and root completion also offers its flags.
These are spec-level nodes, which have no effect through a mise mount.
Below the root: see [Kong only](#default-commands-below-the-root).

Rejected: `default_subcommand_on_empty #true`.
It would also select the default command on an empty command line, exactly as Kong does,
but it breaks root completion in usage 6.12.0–7.3.0:
`x <Tab>` offers the default command's arguments instead of commands.
It is to be emitted once the oldest supported usage version fixes this;
a test detects the fix.

### Help flag

Kong adds `-h, --help` to the root unless `kong.NoDefaultHelp()`
(`Model.HelpFlag`, help "Show context-sensitive help.").
Kong has no `help` subcommand.
usage has built-in `-h`, `--help` and `-?` on every command,
and lists a `help` subcommand in terminal help of a command with subcommands,
which its parser refuses.

Chosen: the built-ins are disabled, and Kong's help flag is declared as a global flag.

```kdl
disable_help_flag #true
disable_help_subcommand #true
flag "-h --help" action="help" global=#true help="Show context-sensitive help."
cmd "sub" disable_help_flag=#true disable_help_subcommand=#true {
    cmd "leaf" disable_help_flag=#true
}
```

The built-in flags are disabled by `disable_help_flag` on every command, as it is not inherited.
Spec-level `disable_help` would give the same docs, completion and parsing,
but has no effect through a mount.
Neither removes the listed `help` subcommand,
so `disable_help_subcommand` is emitted on every command with subcommands.
The flag is omitted for `NoDefaultHelp()`.

usage then accepts and completes only the spellings Kong accepts
(usage before 7.2.0 does not complete the built-ins at all),
and docs list `-h --help` with Kong's help text among global flags of the root.

Rejected: the built-ins, with `disable_help`/`disable_help_flag` only for `NoDefaultHelp()`:
one global flag less in docs, but usage accepts `-?`, lists a `help` subcommand,
and does not complete `--help` before 7.2.0.

### Version flag

A Kong CLI usually declares `Version kong.VersionFlag` printing `kong.Vars{"version": …}`;
the model shows it as an ordinary bool flag of type `kong.VersionFlag`.
usage adds built-in `-V` and `--version` when the spec has `version`.

Chosen:

```kdl
version "1.2.3"
disable_version_flag #true
flag "--version" action="version" global=#true help="Show version."
```

Only the spellings Kong accepts are completed and parsed.
Without a `kong.VersionFlag` only `version` and `disable_version_flag` are emitted.
Through a mise mount `version` has no effect and `--version` is an ordinary flag.

Rejected:

- `version` with a plain bool flag: usage also accepts `-V`, and completes it since 7.2.0.
- No `version`: docs lose the version.

### Flag aliases

`aliases:"f,flg"` on a flag adds `-f` and `--flg`; Kong's help does not show them.

Chosen: hidden aliases.

```kdl
flag "--flag" { alias "-f" "--flg" hide=#true }
```

They are parsed but not completed.
usage still shows a hidden short alias in its terminal help and synopses (a bug).

Rejected: visible spellings (`flag "-f --flag --flg"`).

### Value names

Chosen: Kong's placeholder (`Flag.FormatPlaceHolder()`), as shown by Kong's help,
without quotes and the `,...`/`;...` tail, which usage shows by itself:
`--tags <TAGS>`, `--env <STRING>`, `--map <KEY=VALUE>`.
Positionals use their name, as in Kong.

Kong uses the default value as the placeholder when there is no `placeholder`;
kongusage does not, as usage shows the default by itself
and a value name changes completion:
such a flag gets the placeholder Kong would give it without the default,
i.e. `KEY=VALUE` for a map, otherwise the name of its Go type or of the flag in upper case
(an `int` with `default:"8080"` gives `--port <INT>`, Kong's help shows `--port=8080`).

A placeholder may be any text, but usage refuses the whole spec
when a value name has whitespace or `...`/`…`,
and takes `<<FILE>>` as the name `<FILE>`, which breaks completion by the name.
So one pair of `<>` around the placeholder is removed,
and a placeholder which is empty or still has characters usage refuses
is replaced the same way as one derived from the default.

Such a fix of a placeholder is logged by `CheckSpec` ([Design](design.md)),
a placeholder which is not used because it is the default is not.

usage infers the completion of a value without its own `complete` from its name,
so a placeholder like `HOSTNAME`, `URL` or `DIR`, or a positional named `url`,
changes completion (see [usage](usage.md#completion-of-values)).
This is accepted, although `complete type="unknown"` would prevent it.

Rejected:

- The `placeholder` tag, otherwise the flag name in upper case:
  simpler, but differs from Kong's help more often.
- The default as the value name, as in Kong's help:
  repeats the default and may change completion (`default:"none"` completes nothing).
- Characters usage refuses replaced by `_` (`<a_b>`): a name neither Kong nor the author uses.

### Completion of values

usage completes files for a value without `complete`, `choices`
or a value name it recognises.

Chosen: `complete` by the Go type:

- `type="none"` for numbers, `time.Duration`, `time.Time` and `*url.URL`,
  also as elements of slices and pointers;
- `type="dir"` for `existingdir` and `kong.ChangeDirFlag`
  (usage falls back to files when no directory matches, a bug);
- nothing, so files, for strings, `path`, `existingfile`, `filecontent`, `*os.File`,
  `kong.FileContentFlag`, `kong.NamedFileContentFlag`, `kong.ConfigFlag` and custom types;
- enums complete their choices.

```kdl
flag "--num <INT>" { complete type="none" }
flag "--dir <DIR>" { complete type="dir" }
```

Strings keep the file fallback: it is the default of shells and of completion libraries.
In mise, completed relative paths are wrong when the task's `dir`
is not the shell's directory ([mise](mise.md#task-directory)).

Rejected:

- `type="dir"` only: numbers would complete files.
- Nothing, relying on value names: same.
- Completion computed for the directory of a mise task by `complete run=`:
  postponed, probably not worth its cost.

### Flag relationship groups

`xor:"g"` allows at most one flag of the group per command, exactly one with `required:""`.
`lastwins:"g"` with `required:""` needs at least one member.
A flag may be in several `xor` groups.

Chosen: one `group` per Kong group on the command declaring its flags.

```kdl
group "g1" "--a" "--b"                               // xor
group "g2" "--c" "--d" required=#true                // xor + required
group "g3" "--e" "--f" required=#true multiple=#true // lastwins + required
```

A group of one flag is skipped: usage needs at least two members.
`required` of the members themselves is not emitted.
Kong's README says `required` on members requires the group,
but Kong requires it only when at least two members are `required` (issue #516):
the spec follows Kong, so a group is `required` when at least two members are.

Rejected: pairwise `conflicts` for `xor`: cannot express "required", grows quadratically.

### Unknown flags

Kong refuses an unknown flag, except where a passthrough positional of mode `all` is next.
usage by default binds an unknown flag-like word as a positional value.

Chosen: nothing, so parsing by usage is more permissive than Kong.

Rejected: `unknown_flags "error"` at the root
and `unknown_flags="value"` on commands with a passthrough:
an unknown flag earlier on the line then breaks completion,
the root node has no effect through a mise mount,
and `passthrough:"partial"` cannot be expressed exactly anyway.

### Hidden positional arguments

Kong accepts `hidden:""` on a positional but still shows it in help,
and its README documents `hidden` for commands and flags only.

Chosen: ignore the tag, as Kong's help does.

Rejected: `hide=#true`, which follows the author's intent.

### Group headings

`group:"key"` puts a flag or a command under a heading in Kong's help;
with `kong.ExplicitGroups` or `kong.Groups` the group has a title and a description,
otherwise the title is the key.

Chosen: `help_heading="<Title>"` on the flag or `cmd`,
and `heading "<Title>" help="<Description>"` on every command which declares a member
(for a command, on its parent, where it is listed),
when the group has a description, which usage requires.
A command without a group is under the group of its closest ancestor, as in Kong's help.
usage renders the description in Markdown only, repeats it for a heading with global flags,
and ignores `help_heading` of commands in Markdown.

Rejected: `heading` only on the root; no `heading`.

## Kong only

Kong features usage 7.3.0 cannot express, and what is lost.

### Branching positional arguments

`my-kong-cli rename <name> to <name>`: a positional `arg` struct followed by subcommands.
usage binds the positional, then refuses the following command name.
Emitted: the positional, without the subcommands,
because completing them where Kong expects a value would be wrong.
Lost: completion and docs of everything below the positional.

### Default commands below the root

Lost: the default mark in help and completion of the default command's flags at the parent.

### Counter with an explicit value

Kong sets a counter by `--verbose=3`; usage counts it as one occurrence.
Lost: parsing by usage.

### Other bool values

Kong accepts `yes`, `no`, `1`, `0` in any case, on the command line and in variables,
and refuses other values.
usage accepts only `true` and `false` on the command line,
and takes `1`, `true`, `True`, `TRUE` in a variable as true and anything else as false.
Lost: parsing by usage.

### Escaped separators

Kong's `\,` keeps a separator inside a value; usage splits it.
Lost: parsing by usage.

### Typed and validated values

`format:"…"` of `time.Time`, numeric types, `existingfile`, `existingdir`
and custom mappers validate values in Kong; usage values are strings.
Lost: validation by usage.

### Positional slices stop at a flag

In Kong a slice positional takes consecutive values only:
`sub x -v y` fails, `sub x y -v` works.
usage collects values on both sides of a flag.
Lost: nothing, parsing by usage is more permissive.

### Flags which do not select the default command

A flag whose type implements `kong.IgnoreDefault` prevents selecting the default command.
Lost: parsing by usage.

### Passthrough commands

usage still parses and completes global flags of ancestors, Kong's help flag included,
before the first value of a passthrough command, which Kong passes to the positional.
Lost: parsing and completion by usage.

### `passthrough:"partial"`

Unlike mode `all`, it refuses unknown flags before the first value of the positional,
which usage cannot express per positional (see [Unknown flags](#unknown-flags)).
Lost: parsing by usage.
