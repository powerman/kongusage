package usagekdl

import (
	"slices"

	"github.com/powerman/kongusage/internal/kdl"
)

// Set holds usage features of one entity: the root, a command, a flag or a positional,
// with those of descendant entities it describes.
// A nil Set has no usage features.
type Set struct {
	features map[*Spec][]*feature
	entities []*describedEntity
}

// describedEntity holds usage features of a descendant entity.
type describedEntity struct {
	key  string // Its node name and value as kongusage writes them, e.g. `flag "-f --force"`.
	set  *Set
	used bool // Found in the spec.
}

// ParseRoot parses usage features of the root given as top-level nodes in text,
// or returns nil when text has no nodes.
// The source where names the root and the source of text, for messages.
func ParseRoot(where []string, text string) *Set {
	p := parser{where: where}
	nodes := p.parse(text)
	if len(nodes) == 0 {
		return nil
	}
	return p.entity(nil, AtRoot, nil, nodes)
}

// ParseCommand parses usage features of a command given in text as one node `cmd ""`,
// or as top-level nodes of a CLI mounted as the command,
// or returns nil when text has no nodes.
// The source where names the command and the source of text, for messages.
func ParseCommand(where []string, text string) *Set {
	p := parser{where: where}
	nodes := p.parse(text)
	switch {
	case len(nodes) == 0:
		return nil
	case len(nodes) == 1 && nodes[0].Name == cmdNode && hasEmptyName(nodes[0]):
		return p.entity([]string{`cmd ""`}, AtCmd, nodes[0].Props, nodes[0].Children)
	default:
		return p.entity(nil, AtRoot, nil, nodes).mount()
	}
}

// ParseValue parses usage features of a flag or a positional given in text
// as one node `flag ""` or `arg ""`, or returns nil when text has no nodes.
// The source where names the flag or the positional and the source of text, for messages.
func ParseValue(where []string, text string) *Set {
	p := parser{where: where}
	nodes := p.parse(text)
	switch {
	case len(nodes) == 0:
		return nil
	case len(nodes) == 1 && (nodes[0].Name == flagNode || nodes[0].Name == argNode) && hasEmptyName(nodes[0]):
		return p.entity([]string{nodes[0].Name + ` ""`}, AtArg, nodes[0].Props, nodes[0].Children)
	default:
		failf(p.where, `it must return one node flag "" or arg ""`)
		panic("unreachable")
	}
}

// ParseTag parses usage features of an entity at the place AtCmd or AtArg
// given in text as properties and child nodes of its node, without the node name.
// The source where names the entity and the source of text, for messages.
func ParseTag(where []string, text string, at Place) *Set {
	p := parser{where: where}
	// Positions in parse errors include the prepended name.
	nodes := p.parse("_ " + text)
	if len(nodes) != 1 || len(nodes[0].Values) > 0 {
		failf(p.where, "it must hold properties and child nodes of one node")
	}
	return p.entity(nil, at, nodes[0].Props, nodes[0].Children)
}

func newSet() *Set {
	return &Set{features: make(map[*Spec][]*feature)}
}

// entityKey returns the key of a descendant entity with the node name and the value.
func entityKey(name string, value kdl.Value) string {
	return name + " " + value.String()
}

// Entity returns usage features s describes for the descendant entity
// with the node (its child nodes do not matter) and marks them used, or returns nil.
func (s *Set) Entity(node *kdl.Node) *Set {
	if s == nil {
		return nil
	}
	e := s.find(entityKey(node.Name, node.Values[0]))
	if e == nil {
		return nil
	}
	e.used = true
	return e.set
}

// CheckUsed panics when s describes a descendant entity which is not in the spec.
func (s *Set) CheckUsed(where []string) {
	if s == nil {
		return
	}
	for _, e := range s.entities {
		if !e.used {
			failf(where, "%s is not in the spec", e.key)
		}
	}
}

// Merge changes s by src as docs/usage-features.md "Merge" describes and returns s,
// or returns src when s is nil.
func (s *Set) Merge(src *Set) *Set {
	switch {
	case s == nil:
		return src
	case src == nil:
		return s
	}
	for spec, fs := range src.features {
		switch {
		case fs[0].remove:
			delete(s.features, spec)
		case spec == choicesSpec && s.get(spec) != nil:
			s.get(spec).mergeChoices(fs[0])
		default:
			s.features[spec] = fs
		}
	}
	for _, e := range src.entities {
		if old := s.find(e.key); old != nil {
			old.set.Merge(e.set)
		} else {
			s.entities = append(s.entities, e)
		}
	}
	return s
}

// CheckNoValue panics on usage features of s which do not apply to a flag without a value,
// named by where.
func (s *Set) CheckNoValue(where []string) {
	for _, spec := range []*Spec{completeSpec, choicesSpec} {
		if s.get(spec) != nil {
			failf(where, "%s is not accepted on a flag without a value", spec.Name)
		}
	}
}

// CheckValue panics on usage features of s which do not apply to the value
// of the flag or the positional where with the values of its enum, if any.
func (s *Set) CheckValue(where, enum []string) {
	choices := s.get(choicesSpec)
	if choices == nil {
		return
	}
	where = append(slices.Clone(where), choicesSpec.Name)
	if enum == nil {
		if choices.prop("strict") != kdl.Bool(false) {
			failf(where, "strict=#false is required without an enum, as Kong accepts any value")
		}
		if len(choices.values) == 0 && len(choices.choices) == 0 {
			failf(where, "values or choice nodes are required")
		}
		return
	}
	if len(choices.values) > 0 || choices.prop("strict") != (kdl.Value{}) {
		failf(where, "only choice nodes are accepted with an enum, which gives the values")
	}
	for _, choice := range choices.choices {
		if !slices.Contains(enum, choice.values[0].Interface().(string)) {
			failf(where, "choice %s is not in the enum", choice.values[0])
		}
	}
}

// Nodes returns top-level nodes of usage features of the root s.
func (s *Set) Nodes() []*kdl.Node {
	var nodes []*kdl.Node
	for _, spec := range Specs {
		if s.get(spec) != nil {
			for _, f := range s.features[spec] {
				nodes = append(nodes, f.node())
			}
		}
	}
	return nodes
}

// Apply sets usage features of s on the node of a command, a flag or a positional
// and returns its child nodes, but complete and choices, which Complete and Choices return.
func (s *Set) Apply(node *kdl.Node) []*kdl.Node {
	if s == nil {
		return nil
	}
	var children []*kdl.Node
	for _, spec := range Specs {
		f := s.get(spec)
		switch {
		case f == nil || spec == completeSpec || spec == choicesSpec:
		case spec.form == shortForm && len(f.values) == 1:
			node.SetProp(spec.Name, f.values[0])
		default:
			for _, f := range s.features[spec] {
				children = append(children, f.node())
			}
		}
	}
	return children
}

// HasChoices reports whether s has choices.
func (s *Set) HasChoices() bool {
	return s.get(choicesSpec) != nil
}

// Choices returns the node of choices of a flag or a positional with the enum, if any:
// the values of the enum with choice nodes of s, otherwise choices of s, or nil.
func (s *Set) Choices(enum []string) *kdl.Node {
	author := s.get(choicesSpec)
	switch {
	case enum != nil:
		values := make([]kdl.Value, len(enum))
		for i, v := range enum {
			values[i] = kdl.String(v)
		}
		choices := &feature{spec: choicesSpec, values: values}
		if author != nil {
			choices.choices = author.choices
		}
		return choices.node()
	case author != nil:
		return author.node()
	default:
		return nil
	}
}

// Complete returns the node of complete of s, or nil.
func (s *Set) Complete() *kdl.Node {
	if f := s.get(completeSpec); f != nil {
		return f.node()
	}
	return nil
}

// get returns the feature spec of s, or nil when it is not given or is removed.
func (s *Set) get(spec *Spec) *feature {
	if s == nil {
		return nil
	}
	fs := s.features[spec]
	if len(fs) == 0 || fs[0].remove {
		return nil
	}
	return fs[0]
}

// add adds f to s, given by one source.
func (s *Set) add(where []string, f *feature) {
	old := s.features[f.spec]
	if len(old) > 0 && (!f.spec.repeatable || f.remove || old[0].remove) {
		failf(where, "%s is given twice", f.spec.Name)
	}
	s.features[f.spec] = append(old, f)
}

// find returns the descendant entity of s with the key, or nil.
func (s *Set) find(key string) *describedEntity {
	for _, e := range s.entities {
		if e.key == key {
			return e
		}
	}
	return nil
}

// mount turns usage features of the root of a CLI into those of a command,
// as usage does when it mounts a spec as a command:
// about_md becomes help_md, other features only the root has are dropped.
func (s *Set) mount() *Set {
	for spec, fs := range s.features {
		switch {
		case spec.Name == "about_md":
			delete(s.features, spec)
			helpMD := lookup("help_md")
			for _, f := range fs {
				f.spec = helpMD
			}
			s.features[helpMD] = fs
		case spec.Places&AtCmd == 0:
			delete(s.features, spec)
		}
	}
	return s
}
