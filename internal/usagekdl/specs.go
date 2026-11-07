package usagekdl

//nolint:gochecknoglobals,goconst // Read-only table.
var (
	// Specs lists accepted usage features in the order they are written:
	// after the nodes derived from Kong, except complete and choices,
	// which take the place of the derived ones.
	Specs = []*Spec{
		{Name: "about_md", Places: AtRoot, values: []valueSpec{str}},
		{Name: "author", Places: AtRoot, values: []valueSpec{str}},
		{Name: "license", Places: AtRoot, values: []valueSpec{str}},
		{Name: "repository", Places: AtRoot, values: []valueSpec{str}},
		{Name: "source_code_link_template", Places: AtRoot, values: []valueSpec{str}},
		{Name: "logo", Places: AtRoot, values: []valueSpec{str}, Props: []PropSpec{{"style", str}}},
		{Name: "help_md", Places: AtCmd | AtArg, form: textForm, values: []valueSpec{str}},
		{Name: "long_help", Places: AtArg, form: textForm, values: []valueSpec{str}},
		{Name: "before_help", Places: AtRoot | AtCmd, form: textForm, values: []valueSpec{str}},
		{Name: "before_long_help", Places: AtRoot | AtCmd, form: textForm, values: []valueSpec{str}},
		{Name: "before_help_md", Places: AtCmd, form: textForm, values: []valueSpec{str}},
		{Name: "after_help", Places: AtRoot | AtCmd, form: textForm, values: []valueSpec{str}},
		{Name: "after_long_help", Places: AtRoot | AtCmd, form: textForm, values: []valueSpec{str}},
		{Name: "after_help_md", Places: AtCmd, form: textForm, values: []valueSpec{str}},
		{Name: "note", Places: AtArg, repeatable: true, values: []valueSpec{str}},
		{Name: "warning", Places: AtArg, repeatable: true, values: []valueSpec{str}},
		{
			Name: "example", Places: AtRoot | AtCmd, repeatable: true, values: []valueSpec{str},
			Props: []PropSpec{{"header", str}, {"help", str}, {"lang", str}},
		},
		{Name: "exit_code", Places: AtRoot | AtCmd, repeatable: true, values: []valueSpec{integer, str}},
		{
			Name: "output", Places: AtRoot | AtCmd, repeatable: true, values: []valueSpec{str},
			Props: []PropSpec{
				{"media_type", str},
				{"framing", enum("text", "json", "jsonl")},
				{"help", str},
				{"schema", str},
				{"default", boolean},
				{"hide", boolean},
			},
		},
		{Name: "effect", Places: AtCmd | AtArg, form: shortForm, values: []valueSpec{enum("read", "write", "destructive")}},
		{Name: "surface", Places: AtRoot | AtCmd | AtArg, form: shortForm, values: []valueSpec{str}},
		{Name: "available_if", Places: AtRoot | AtCmd | AtArg, form: shortForm, values: []valueSpec{str}, rest: str},
		{Name: "hide_short_help", Places: AtArg, form: shortForm, values: []valueSpec{boolean}},
		{Name: "hide_long_help", Places: AtArg, form: shortForm, values: []valueSpec{boolean}},
		{Name: "subcommand_help_heading", Places: AtRoot | AtCmd, form: shortForm, values: []valueSpec{str}},
		{Name: "subcommand_value_name", Places: AtRoot | AtCmd, form: shortForm, values: []valueSpec{str}},
		{Name: "next_line_help", Places: AtRoot | AtCmd, form: shortForm, values: []valueSpec{boolean}},
		{Name: "flatten_help", Places: AtRoot | AtCmd, form: shortForm, values: []valueSpec{boolean}},
		{Name: "term_width", Places: AtRoot | AtCmd, form: shortForm, values: []valueSpec{natural}},
		{Name: "max_term_width", Places: AtRoot | AtCmd, form: shortForm, values: []valueSpec{natural}},
		{Name: "default_subcommand_help", Places: AtRoot, values: []valueSpec{boolean}},
		completeSpec,
		choicesSpec,
	}
	// completeSpec replaces the completion derived from Kong.
	completeSpec = &Spec{
		Name: "complete", Places: AtArg,
		Props: []PropSpec{
			// Types which usage 6.12.0 knows, but config_keys and config_values,
			// which complete the `config` node, which is not accepted.
			{"type", enum("path", "file", "dir", "executable", "command", "command_args",
				"username", "hostname", "url", "email", "none", "unknown")},
			{"run", str},
			{"descriptions", boolean},
			{"delegate", str},
		},
	}
	// choicesSpec is merged by parts, unlike other features (see docs/usage-features.md "Merge").
	choicesSpec = &Spec{
		Name: "choices", Places: AtArg, rest: str,
		Props:    []PropSpec{{"strict", boolean}},
		Children: []*Spec{choiceSpec},
	}
	choiceSpec = &Spec{
		Name: "choice", values: []valueSpec{str},
		Props: []PropSpec{{"help", str}, {"hide", boolean}},
	}
)
