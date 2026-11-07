package kongusage

import (
	"fmt"
	"slices"
	"strings"

	"github.com/alecthomas/kong"

	"github.com/powerman/kongusage/internal/kdl"
)

// MinUsageVersion is the oldest supported usage version.
const MinUsageVersion = "6.12.0"

// converter converts the Kong model of a CLI to nodes of a usage spec by docs/mapping.md.
type converter struct {
	cfg            Config
	hyphenPrefixed bool
	version        string
	helpFlag       *kong.Flag
	// notes describe silent fixes of the Kong model (docs/design.md).
	notes []string
}

// convert returns the top-level nodes of the usage spec of k
// and notes on the parts of the Kong model it has fixed silently.
func convert(k *kong.Kong, cfg Config) (nodes []*kdl.Node, notes []string) {
	c := &converter{
		cfg:            cfg,
		hyphenPrefixed: hyphenPrefixed(k),
		version:        versionVar(k),
		helpFlag:       k.Model.HelpFlag,
	}
	return c.application(k.Model), c.notes
}

// notef adds a note on a silent fix of the entity of the command cmd.
func (c *converter) notef(cmd *kong.Node, entity, format string, args ...any) {
	c.notes = append(c.notes, cmd.FullPath()+": "+entity+": "+fmt.Sprintf(format, args...))
}

func (c *converter) application(app *kong.Application) []*kdl.Node {
	nodes := []*kdl.Node{
		kdl.NewNode("min_usage_version", kdl.String(MinUsageVersion)),
		kdl.NewNode("name", kdl.String(app.Name)),
		kdl.NewNode("bin", kdl.String(app.Name)),
	}
	if app.Help != "" {
		nodes = append(nodes, kdl.NewNode("about", kdl.String(app.Help)))
	}
	if app.Detail != "" {
		nodes = append(nodes, kdl.NewNode("long_about", kdl.String(app.Detail)))
	}
	if c.version != "" {
		nodes = append(nodes, kdl.NewNode("version", kdl.String(c.version)))
	}
	nodes = append(nodes, kdl.NewNode("disable_help_flag", kdl.Bool(true)))
	if hasSubcommands(app.Node) {
		nodes = append(nodes, kdl.NewNode("disable_help_subcommand", kdl.Bool(true)))
	}
	if c.version != "" {
		nodes = append(nodes, kdl.NewNode("disable_version_flag", kdl.Bool(true)))
	}
	return append(nodes, c.members(app.Node)...)
}

// members returns the nodes of flags, positional arguments and subcommands of the command n.
func (c *converter) members(n *kong.Node) []*kdl.Node {
	var nodes []*kdl.Node
	for _, f := range n.Flags {
		nodes = append(nodes, c.flag(n, f))
	}
	for _, p := range n.Positional {
		nodes = append(nodes, c.positional(n, p))
	}
	order := 0
	for _, child := range n.Children {
		switch child.Type {
		case kong.CommandNode:
			nodes = append(nodes, c.command(child, order))
			order++
		case kong.ApplicationNode, kong.ArgumentNode:
			// TODO(stage 4): docs/mapping.md "Branching positional arguments".
		}
	}
	return nodes
}

func (c *converter) flag(cmd *kong.Node, f *kong.Flag) *kdl.Node {
	isHelp := f == c.helpFlag
	isVersion := isVersionFlag(f)

	node := kdl.NewNode("flag", kdl.String(c.flagName(cmd, f)))

	c.setHelp(node, f.Value)
	// Kong turns required on and-group members into required of the whole group.
	if f.Tag.Required || andGroupRequired(cmd, f) {
		node.SetProp("required", kdl.Bool(true))
	}
	if f.HasDefault {
		node.SetProp("default", c.defaultValue(f))
	}
	if f.Hidden {
		node.SetProp("hide", kdl.Bool(true))
	}
	if f.Tag.Negatable != "" {
		node.SetProp("negate", kdl.String(negation(f)))
	}
	if f.IsCounter() {
		node.SetProp("count", kdl.Bool(true))
	}
	if isCumulative(f.Value) {
		node.SetProp("var", kdl.Bool(true))
		if sep := separator(f); sep != -1 {
			node.SetProp("delimiter", kdl.String(string(sep)))
		}
	}
	switch {
	case isHelp:
		node.SetProp("action", kdl.String("help"))
	case isVersion:
		node.SetProp("action", kdl.String("version"))
	case f.IsBool() && !f.IsCounter():
		node.SetProp("bool_value", kdl.Bool(true))
	}
	// Kong accepts flags of ancestors after a subcommand.
	if isHelp || isVersion || hasSubcommands(cmd) {
		node.SetProp("global", kdl.Bool(true))
	}
	if takesValue(f) && c.hyphenPrefixed {
		node.SetProp("allow_hyphen_values", kdl.Bool(true))
	}
	if len(f.Aliases) > 0 {
		node.AddChild(kdl.NewNode("alias", c.strValues(aliases(f))...).SetProp("hide", kdl.Bool(true)))
	}
	if !isVersion {
		c.setEnv(node, f.Tag.Envs)
	}
	c.setChoices(node, f.Value)
	c.setRelated(node, "requires", andMembers(cmd, f))
	c.setRelated(node, "overrides", groupMembers(cmd, f, func(f *kong.Flag) []string { return f.LastWins }))
	return node
}

func (c *converter) positional(cmd *kong.Node, p *kong.Positional) *kdl.Node {
	name := c.positionalName(cmd, p)
	if p.Required {
		name = "<" + name + ">"
	} else {
		name = "[" + name + "]"
	}
	if isCumulative(p) {
		name += "..."
	}
	node := kdl.NewNode("arg", kdl.String(name))
	c.setHelp(node, p)
	if p.HasDefault {
		node.SetProp("default", kdl.String(p.Default))
	}
	c.setEnv(node, p.Tag.Envs)
	c.setChoices(node, p)
	// TODO(stage 4): docs/mapping.md "Hidden positional arguments".
	return node
}

func (c *converter) command(n *kong.Node, order int) *kdl.Node {
	node := kdl.NewNode("cmd", kdl.String(n.Name))
	if n.Help != "" {
		node.SetProp("help", kdl.String(n.Help))
	}
	if n.Detail != "" {
		node.SetProp("long_help", kdl.String(n.Detail))
	}
	if n.Hidden {
		node.SetProp("hide", kdl.Bool(true))
	}
	if !c.cfg.NoDisplayOrder {
		node.SetProp("display_order", kdl.Int(order))
	}
	if subcommandRequired(n) {
		node.SetProp("subcommand_required", kdl.Bool(true))
	}
	node.SetProp("disable_help_flag", kdl.Bool(true))
	if hasSubcommands(n) {
		node.SetProp("disable_help_subcommand", kdl.Bool(true))
	}
	if len(n.Aliases) > 0 {
		node.AddChild(kdl.NewNode("alias", c.strValues(n.Aliases)...))
	}
	return node.AddChild(c.members(n)...)
}

// flagName returns the spellings of the flag f of the command cmd
// followed by its value name, if it takes one.
func (c *converter) flagName(cmd *kong.Node, f *kong.Flag) string {
	name := "--" + f.Name
	if f.Short != 0 {
		name = "-" + string(f.Short) + " " + name
	}
	if takesValue(f) {
		name += " <" + c.valueName(cmd, f) + ">"
	}
	return name
}

// valueName returns the value name of the flag f of the command cmd
// by docs/mapping.md "Value names".
func (c *converter) valueName(cmd *kong.Node, f *kong.Flag) string {
	orig := placeHolder(f)
	name := trimAngles(orig)
	var reason string
	switch {
	case name == "":
		reason = "is empty"
	case hasSpace(name):
		reason = "has whitespace, which usage refuses in a value name"
	case hasEllipsis(name):
		reason = "has an ellipsis, which usage refuses in a value name"
	default:
		return name
	}
	name = plainPlaceHolder(f)
	fix := "the placeholder tag of the flag"
	if _, ok := f.Mapper.(kong.PlaceHolderProvider); ok {
		fix = "PlaceHolder of its mapper"
	}
	c.notef(cmd, "flag --"+f.Name, "placeholder %q %s, so the spec uses %q; change %s",
		orig, reason, name, fix)
	return name
}

// positionalName returns the name of the positional p of the command cmd in the spec.
// Only a name tag gives a name usage cannot take: it is fixed as a value name of a flag,
// except whitespace, which usage takes in the name of a positional,
// and the replacement is in lower case, as names of positionals usually are.
func (c *converter) positionalName(cmd *kong.Node, p *kong.Positional) string {
	name := trimAngles(p.Name)
	var reason string
	switch {
	case name == "":
		reason = "is empty"
	case hasEllipsis(name):
		reason = "has an ellipsis, which usage takes as a mark of a variadic positional"
	default:
		return name
	}
	value := *p
	value.Name = "arg" // Used for a type without a name, e.g. []string.
	name = strings.ToLower(plainPlaceHolder(&kong.Flag{Value: &value}))
	c.notef(cmd, "positional "+p.Name, "the name %q %s, so the spec uses %q; change the name tag",
		p.Name, reason, name)
	return name
}

// setHelp sets setHelp of a flag or a positional
// and hides values which the setHelp text already shows.
func (*converter) setHelp(node *kdl.Node, v *kong.Value) {
	if v.Help != "" {
		node.SetProp("help", kdl.String(v.Help))
	}
	for _, h := range []struct{ variable, prop string }{
		{"${default}", "hide_default_value"},
		{"${enum}", "hide_possible_values"},
		{"${env}", "hide_env"},
	} {
		if strings.Contains(v.OrigHelp, h.variable) {
			node.SetProp(h.prop, kdl.Bool(true))
		}
	}
}

// setEnv sets the environment variables of a flag or a positional.
func (c *converter) setEnv(node *kdl.Node, envs []string) {
	// `env:"-"` stays in the model without kong.DefaultEnvars.
	envs = slices.DeleteFunc(slices.Clone(envs), func(e string) bool { return e == "-" })
	if len(envs) == 0 {
		return
	}
	node.SetProp("env", kdl.String(envs[0]))
	if len(envs) > 1 {
		node.AddChild(kdl.NewNode("env_fallback", c.strValues(envs[1:])...))
	}
}

// setChoices sets the possible values of a flag or a positional.
func (c *converter) setChoices(node *kdl.Node, v *kong.Value) {
	if v.Enum != "" {
		node.AddChild(kdl.NewNode("choices", c.strValues(v.EnumSlice())...))
	}
}

// setRelated sets the relationship name of node to flags:
// a property for one flag, a child node for several.
func (c *converter) setRelated(node *kdl.Node, name string, flags []string) {
	switch len(flags) {
	case 0:
	case 1:
		node.SetProp(name, kdl.String(flags[0]))
	default:
		node.AddChild(kdl.NewNode(name, c.strValues(flags)...))
	}
}

// strValues returns the KDL string values of ss.
func (*converter) strValues(ss []string) []kdl.Value {
	values := make([]kdl.Value, len(ss))
	for i, s := range ss {
		values[i] = kdl.String(s)
	}
	return values
}

// defaultValue returns the default of the flag f, a boolean for a bool flag.
func (*converter) defaultValue(f *kong.Flag) kdl.Value {
	if f.IsBool() && !f.IsCounter() {
		// Kong parses the default by the mapper of the flag:
		// a custom bool mapper may take other values, which stay strings.
		switch strings.ToLower(f.Default) {
		case "true", "1", "yes":
			return kdl.Bool(true)
		case "false", "0", "no":
			return kdl.Bool(false)
		}
	}
	return kdl.String(f.Default)
}
