package kongusage

import (
	"fmt"
	"slices"
	"strings"

	"github.com/alecthomas/kong"

	"github.com/powerman/kongusage/internal/kdl"
	"github.com/powerman/kongusage/internal/usagekdl"
)

// MinUsageVersion is the oldest supported usage version.
// The spec sets it as `min_usage_version`, and [CheckSpec] requires at least this version.
const MinUsageVersion = "6.12.0"

// converter converts the Kong model of a CLI to nodes of a usage spec by docs/mapping.md.
type converter struct {
	cfg            Config
	k              *kong.Kong
	mappers        *kong.Registry // Kong's default mappers, to tell them from custom ones.
	hyphenPrefixed bool
	version        string

	nodes []*kdl.Node
	notes []string // describe silent fixes of the Kong model (docs/design.md).
}

// newConverter converts the Kong model of k to the usage spec.
func newConverter(k *kong.Kong, cfg Config) *converter {
	c := &converter{
		cfg:            cfg,
		k:              k,
		mappers:        kong.NewRegistry().RegisterDefaults(),
		hyphenPrefixed: hyphenPrefixed(k),
		version:        versionVar(k),
	}
	c.convert()
	return c
}

// Nodes returns the top-level nodes of the usage spec.
func (c *converter) Nodes() []*kdl.Node {
	return c.nodes
}

// Notes returns the notes on the parts of the Kong model fixed silently.
func (c *converter) Notes() []string {
	return c.notes
}

// notef adds a note on a silent fix of the entity of the command cmd.
func (c *converter) notef(cmd *kong.Node, entity, format string, args ...any) {
	c.notes = append(c.notes, cmd.FullPath()+": "+entity+": "+fmt.Sprintf(format, args...))
}

func (c *converter) convert() {
	app := c.k.Model
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
	// Kong's default commands below the root are lost: usage has them at the root only.
	// A flag of a type implementing kong.IgnoreDefault, which prevents selecting the default
	// command in Kong, is lost too: usage cannot express it.
	if def := app.DefaultCmd; def != nil {
		nodes = append(nodes, kdl.NewNode("default_subcommand", kdl.String(def.Name)))
		if def.Tag.Default == "withargs" {
			nodes = append(nodes, kdl.NewNode("default_subcommand_flags", kdl.Bool(true)))
		}
		// default_subcommand_on_empty would select the default command
		// on an empty command line as Kong does, but it breaks root completion
		// (see TestDefaultSubcommandOnEmpty, https://github.com/jdx/usage/issues/1630).
	}
	nodes = append(nodes, kdl.NewNode("disable_help_flag", kdl.Bool(true)))
	if hasSubcommands(app.Node) {
		nodes = append(nodes, kdl.NewNode("disable_help_subcommand", kdl.Bool(true)))
	}
	if c.version != "" {
		nodes = append(nodes, kdl.NewNode("disable_version_flag", kdl.Bool(true)))
	}
	fs := c.rootUsageKDL()
	nodes = append(nodes, fs.Nodes()...)
	nodes = append(nodes, c.members(app.Node, fs)...)
	fs.CheckUsed([]string{app.Name})
	c.nodes = nodes
}

// members returns the nodes of headings, flags, positional arguments, groups of flags
// and subcommands of the command n with usage features fs.
// Kong's unknown flags are not expressed (docs/mapping.md "Unknown flags"):
// usage parses them more permissively.
func (c *converter) members(n *kong.Node, fs *usagekdl.Set) []*kdl.Node {
	nodes := c.headings(n)
	for _, f := range n.Flags {
		nodes = append(nodes, c.flag(n, f, fs))
	}
	for _, p := range n.Positional {
		nodes = append(nodes, c.positional(n, p, fs))
	}
	nodes = append(nodes, c.groups(n)...)
	order := 0
	branched := false
	for _, child := range n.Children {
		switch child.Type {
		case kong.CommandNode:
			nodes = append(nodes, c.command(child, order, fs))
			order++
		case kong.ArgumentNode:
			// usage cannot continue with commands after a positional:
			// only the positional is emitted, its flags and subcommands are lost.
			// Kong takes the first branch whose positional parses,
			// which another positional cannot express, so later branches are lost too.
			// usage features of the type of the branch are lost with it.
			if !branched {
				nodes = append(nodes, c.positional(child, child.Argument, fs))
				branched = true
			}
		case kong.ApplicationNode:
			// Kong never makes an application node a child.
		}
	}
	return nodes
}

// headings returns the headings of the groups of the members of the command n
// which have a description, in the order of their first use.
// A command is a member of the closest group of its own or of its ancestors,
// as in Kong's help.
func (*converter) headings(n *kong.Node) []*kdl.Node {
	var groups []*kong.Group
	add := func(g *kong.Group) {
		if g == nil || g.Description == "" {
			return
		}
		// usage tells headings by their titles.
		if !slices.ContainsFunc(groups, func(other *kong.Group) bool { return other.Title == g.Title }) {
			groups = append(groups, g)
		}
	}
	for _, f := range n.Flags {
		add(f.Group)
	}
	for _, child := range n.Children {
		if child.Type == kong.CommandNode {
			add(child.ClosestGroup())
		}
	}
	nodes := make([]*kdl.Node, len(groups))
	for i, g := range groups {
		nodes[i] = kdl.NewNode("heading", kdl.String(g.Title)).SetProp("help", kdl.String(g.Description))
	}
	return nodes
}

// groups returns the groups of flags of the command n by docs/mapping.md
// "Flag relationship groups": every xor group and every required lastwins group.
func (*converter) groups(n *kong.Node) []*kdl.Node {
	var nodes []*kdl.Node
	for _, g := range choiceGroups(n) {
		// usage refuses a group of one flag; Kong requires no flag of such a group.
		if len(g.flags) == 1 {
			continue
		}
		required := choiceGroupRequired(n, g.flags)
		// Overrides of the members express a lastwins group which is not required.
		if g.lastWins && !required {
			continue
		}
		values := []kdl.Value{kdl.String(g.name)}
		for _, f := range g.flags {
			values = append(values, kdl.String("--"+f.Name))
		}
		node := kdl.NewNode("group", values...)
		if required {
			node.SetProp("required", kdl.Bool(true))
		}
		if g.lastWins {
			node.SetProp("multiple", kdl.Bool(true))
		}
		nodes = append(nodes, node)
	}
	return nodes
}

// flag returns the node of the flag f of the command cmd with usage features cmdUsageKDL.
func (c *converter) flag(cmd *kong.Node, f *kong.Flag, cmdUsageKDL *usagekdl.Set) *kdl.Node {
	isHelp := f == c.k.Model.HelpFlag
	isVersion := isVersionFlag(f)

	node := kdl.NewNode("flag", kdl.String(c.flagName(cmd, f)))
	fs := c.valueUsageKDL([]string{cmd.FullPath(), "flag --" + f.Name}, cmdUsageKDL.Entity(node), f.Value)

	c.setHelp(node, f.Value)
	// Kong requires a member of an xor or lastwins group only together with other members,
	// which the group of flags expresses.
	if isRequired(cmd, f) && len(f.Xor) == 0 && len(f.LastWins) == 0 {
		node.SetProp("required", kdl.Bool(true))
	}
	if f.HasDefault {
		node.SetProp("default", c.defaultValue(f))
	}
	if f.Hidden {
		node.SetProp("hide", kdl.Bool(true))
	}
	if f.Group != nil {
		node.SetProp("help_heading", kdl.String(f.Group.Title))
	}
	if f.Tag.Negatable != "" {
		node.SetProp("negate", kdl.String(negation(f)))
	}
	if f.IsCounter() {
		// Kong also sets a counter by `--verbose=3`, which usage counts as one occurrence.
		node.SetProp("count", kdl.Bool(true))
	}
	c.setCumulative(node, f)
	switch {
	case isHelp:
		node.SetProp("action", kdl.String("help"))
	case isVersion:
		node.SetProp("action", kdl.String("version"))
	case f.IsBool() && !f.IsCounter():
		// Kong also takes yes, no, 1 and 0 in any case, usage only true and false.
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
	c.setChoices(node, f.Value, fs)
	if takesValue(f) {
		c.setComplete(node, f.Value, fs)
	}
	c.setRelations(node, cmd, f)
	return node.AddChild(fs.Apply(node)...)
}

// positional returns the node of the positional p of the command cmd
// with usage features cmdUsageKDL.
func (c *converter) positional(cmd *kong.Node, p *kong.Positional, cmdUsageKDL *usagekdl.Set) *kdl.Node {
	name := c.positionalName(cmd, p)
	if p.Required {
		name = "<" + name + ">"
	} else {
		name = "[" + name + "]"
	}
	if isCumulative(p) {
		// Kong stops a slice at a flag (`x -v y` fails), usage takes values on both sides.
		name += "..."
	}
	node := kdl.NewNode("arg", kdl.String(name))
	fs := c.valueUsageKDL(
		[]string{cmd.FullPath(), "positional " + p.Name},
		cmdUsageKDL.Entity(node),
		p,
	)
	c.setHelp(node, p)
	if p.HasDefault {
		node.SetProp("default", kdl.String(p.Default))
	}
	// `hidden:""` is ignored, as Kong's help shows a hidden positional anyway.
	// Kong refuses unknown flags before the first value of `passthrough:"partial"`,
	// which usage cannot express for one positional.
	if p.PassthroughMode != kong.PassThroughModeNone || cmd.Passthrough {
		node.SetProp("double_dash", kdl.String("automatic"))
	}
	c.setEnv(node, p.Tag.Envs)
	c.setChoices(node, p, fs)
	c.setComplete(node, p, fs)
	return node.AddChild(fs.Apply(node)...)
}

// command returns the node of the command n,
// which has the position order among its siblings,
// with usage features of its parent parentUsageKDL.
func (c *converter) command(n *kong.Node, order int, parentUsageKDL *usagekdl.Set) *kdl.Node {
	node := kdl.NewNode("cmd", kdl.String(n.Name))
	fs := c.commandUsageKDL(n, parentUsageKDL.Entity(node))
	if n.Help != "" {
		node.SetProp("help", kdl.String(n.Help))
	}
	if n.Detail != "" {
		node.SetProp("long_help", kdl.String(n.Detail))
	}
	if n.Hidden {
		node.SetProp("hide", kdl.Bool(true))
	}
	if g := n.ClosestGroup(); g != nil {
		node.SetProp("help_heading", kdl.String(g.Title))
	}
	if !c.cfg.NoDisplayOrder {
		node.SetProp("display_order", kdl.Int(order))
	}
	if subcommandRequired(n) {
		node.SetProp("subcommand_required", kdl.Bool(true))
	}
	// Without the built-in help flags usage does not take a leading `--help`
	// of a passthrough command as help, but it still takes inherited global flags there,
	// Kong's help flag included.
	node.SetProp("disable_help_flag", kdl.Bool(true))
	if hasSubcommands(n) {
		node.SetProp("disable_help_subcommand", kdl.Bool(true))
	}
	if len(n.Aliases) > 0 {
		node.AddChild(kdl.NewNode("alias", c.strValues(n.Aliases)...))
	}
	node.AddChild(fs.Apply(node)...)
	node.AddChild(c.members(n, fs)...)
	fs.CheckUsed([]string{n.FullPath()})
	return node
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

// setCumulative sets `var` of a flag f which accumulates values.
func (*converter) setCumulative(node *kdl.Node, f *kong.Flag) {
	if !isCumulative(f.Value) {
		return
	}
	node.SetProp("var", kdl.Bool(true))
	// usage splits values by the delimiter even where Kong's `\,` escapes it.
	if sep := separator(f); sep != -1 {
		node.SetProp("delimiter", kdl.String(string(sep)))
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

// setChoices sets the possible values of a flag or a positional with usage features fs.
func (*converter) setChoices(node *kdl.Node, v *kong.Value, fs *usagekdl.Set) {
	if choices := fs.Choices(enum(v)); choices != nil {
		node.AddChild(choices)
	}
}

// setComplete sets the completion of a value with usage features fs:
// complete of fs, otherwise of a value without choices by docs/mapping.md "Completion of values".
// Values are not validated by usage, which is lost.
func (c *converter) setComplete(node *kdl.Node, v *kong.Value, fs *usagekdl.Set) {
	if complete := fs.Complete(); complete != nil {
		node.AddChild(complete)
		return
	}
	// usage answers by a closed `complete type` before choices,
	// so type="none" would hide suggested values (see TestWriteSpec_choicesCompletion).
	if v.Enum != "" || fs.HasChoices() {
		return
	}
	if typ := completion(c.mappers, v); typ != "" {
		node.AddChild(kdl.NewNode("complete").SetProp("type", kdl.String(typ)))
	}
}

// setRelations sets the flags which the flag f of the command cmd requires and overrides.
func (c *converter) setRelations(node *kdl.Node, cmd *kong.Node, f *kong.Flag) {
	c.setRelated(node, "requires", andMembers(cmd, f))
	c.setRelated(node, "overrides", groupMembers(cmd, f, func(f *kong.Flag) []string { return f.LastWins }))
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
