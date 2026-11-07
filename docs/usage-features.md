# Usage features

Features of the [usage spec](https://usage.jdx.dev/spec/) which Kong has no counterpart for,
so the author of the CLI supplies them.
It is correct for usage 6.12.0–7.2.0.
What kongusage derives from the Kong model is described in [Mapping](mapping.md).

Kong semantics (parsing, validation) is supported only together with its implementation in Kong,
so the spec cannot describe a CLI which behaves differently.

## Sources

The author supplies features in KDL v2, as a subset of nodes of the usage spec:

- by the optional method `UsageKDL()` (`kongusage.KDLProvider`)
  of the named root, command, flag and positional argument types;
- by the optional struct tag `usage-kdl` of the field of a command,
  a flag or a positional argument.

`UsageKDL()` of the root returns top-level nodes of the spec.
`UsageKDL()` of another type returns the node for a field of that type
with an empty value, as the type does not know the name of the field:
`cmd ""`, `flag ""` or `arg ""` (the last two are interchangeable)
with properties and child nodes.
`UsageKDL()` of the type of a command which returns top-level nodes instead
is the root of another CLI used as a command:
its nodes become child nodes of the command,
`about_md` becomes `help_md` and other nodes which only the root has are ignored.
An empty result is the same as no `UsageKDL()`.

Nested `cmd`, `flag` and `arg` nodes describe descendants of a node.
They are named exactly as in the spec kongusage generates,
e.g. `cmd "run"`, `flag "-f --force"`, `arg "<file>..."`;
a name which is not in the spec is an error.

The tag holds properties and child nodes of the node of its field, without its name and value,
e.g. `usage-kdl:"effect=destructive { note \"See --backup.\" }"`.

## Merge

Usage features are collected from the root down:

1. `UsageKDL()` of the root gives the first nodes.
2. For every field of a command, a flag or a positional argument, top-down:
   `UsageKDL()` of the type of the field gives its node,
   unless an ancestor has already described the node;
   then the tag of the field changes the node.

Only the tag is merged.
Where there was no clear better way, the rules follow how usage merges an included spec
into a command, though they differ from it (e.g. usage replaces all flags at once):

- A node which usage accepts both as a property and as a child node
  is the same node in either form.
- `cmd`, `flag`, `arg` and `choice` nodes change the node of the same value,
  `choices` changes the node of the same name, and its values replace the old ones if given.
- Any other node or property replaces the one of the same name,
  a repeatable node all nodes of that name at once (e.g. every `example`).
- A node with the only value `#null`, or a property with the value `#null`,
  removes the node or property of the same name.

## Validation

- Only the nodes below are accepted, with the values, properties and child nodes listed;
  any other node, property or child node is an error,
  as is a value of an enumeration unknown to kongusage
  (e.g. `effect` other than `read`, `write`, `destructive`, or an unknown `complete type`).
- A node given in both forms, or a single node given twice, in one source is an error,
  as is a type annotation.
- A node of `UsageKDL()` of a type which does not apply to the field
  (e.g. a `choice` for a value outside the `enum` of the field)
  is an error unless the tag of the field removes it.
- A node, value or property added after the oldest supported usage version
  is not accepted until that version reaches it ([Design](design.md#usage-version)).
- Other validation is left to `usage lint`, which `kongusage.CheckSpec` runs.
  A CLI must have a test which calls `CheckSpec`, with the `usage` CLI installed.

## Accepted

Properties are given in parentheses, values in angle brackets.

The root:

- `about_md`, `author`, `license`, `repository`, `source_code_link_template`;
- `logo` (`style`), `default_subcommand_help`.

The root and commands:

- `before_help`, `after_help`, `before_long_help`, `after_long_help`;
- `example` (`header`, `help`, `lang`), `exit_code <code> "<help>"`;
- `surface`, `available_if`;
- `output` (`media_type`, `framing`, `help`, `schema`, `default`, `hide`);
- help layout: `subcommand_help_heading`, `subcommand_value_name`, `next_line_help`,
  `flatten_help`, `term_width`, `max_term_width`.

Commands:

- `help_md`, `before_help_md`, `after_help_md`;
- `effect`.

Flags and positionals:

- `long_help`, `help_md`, `note`, `warning`;
- `effect`, `surface`, `available_if`;
- `hide_short_help`, `hide_long_help`;
- `complete` (`type`, `run`, `descriptions`, `delegate`),
  which replaces the completion chosen by [Mapping](mapping.md#completion-of-values);
- `choices` (`strict`) with child nodes `choice` (`help`, `hide`):
  of a value without an enum only with `strict=#false`, i.e. suggested values,
  which replace the completion chosen by [Mapping](mapping.md#completion-of-values);
  of a value with an enum only `choice` nodes for its values,
  merged into the `choices` derived from the enum.

## Not accepted

Any other node is an error. Notable ones and why they are not accepted:

Nodes and properties which [Mapping](mapping.md) derives from Kong:
the spec describes the behaviour of Kong, so they come from Kong only;
and `min_usage_version`, which kongusage sets itself ([Design](design.md#usage-version)).

Semantics, which needs its Kong implementation; how to do it is to be designed first:

- relationships beyond `xor`, `and` and `lastwins`:
  `requires` (one-way), `conflicts`, `required_if`, `required_if_eq`, `required_if_eq_all`,
  `required_unless`, `required_unless_all`, `requires_if`, `default_if`, `exclusive`;
- number of values: `var_min`, `var_max`;
- validation: `validate`, `validate_error`; choices `ignore_case`, `run`,
  `choice` `alias`;
- `value_optional`, `default_missing`;
- deprecation, which usage reports on use:
  `deprecated`, `deprecated_warn_at`, `deprecated_remove_at`, `deprecated_env`;
- `arg_required_else_help`;
- `unknown_env` (`prefix`, `allow`);
- `config`: usage reads config files by it and fills values from them;
- `multicall` and `view` (`name`, `bin`, `root`, `globals`, `global`).

Behaviour which Kong's parser does not have:

- `external_subcommand`, `restart_token`, `clause`, `sigil`;
- `value_terminator`, `require_equals`, `allow_negative_numbers`,
  `delimiter` of positionals, `double_dash` other than in [Mapping](mapping.md#passthrough);
- `args_override_self=#false`, `subcommand_negates_reqs`, `args_conflicts_with_subcommands`,
  `subcommand_precedence_over_arg`, `allow_missing_positional`, `dont_delimit_trailing_values`:
  Kong's behaviour is fixed, and the default of usage matches it
  or the case cannot arise in Kong.

Constructs which Kong does already:

- `long_version`: shown only by the built-in `--version` of usage,
  while a Kong CLI prints its version itself;
- `usage`: the synopsis is generated from the same spec;
- `value_names`: a Kong positional has one name;
- `action` `help_short`, `help_long`, `help_all`: Kong has one help flag.

Other constructs:

- `include`, `flagset`, `use`: constructs for writing a spec by hand;
- `mount`: the Kong model is static and complete;
- `choices env`: needs the `unstable_choices_env` feature of usage-lib, which mise lacks;
- `builtin`: marks entries materialized by the generators of usage;
- `hide_env_values`: has no effect in usage;
- `select` (also of `output`): has to agree with the `enum` of its flag,
  which is not checked yet;
- `help_template`: its placeholders depend on the usage version,
  and a wrong one makes usage refuse the whole spec,
  which `usage lint` of a newer usage does not find;
- top-level `complete`: use `complete` of the flag or positional.

## Open questions

- `deprecated*` is not accepted as semantics, because usage warns when a deprecated entry
  is used. If it is rather presentation, it becomes accepted.
