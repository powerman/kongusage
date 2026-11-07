package usagekdl

import (
	"slices"
	"strings"

	"github.com/powerman/kongusage/internal/kdl"
)

// Names of nodes of entities which have usage features.
const (
	cmdNode  = "cmd"
	flagNode = "flag"
	argNode  = "arg"
)

// hasEmptyName reports whether the only value of n is an empty string.
func hasEmptyName(n *kdl.Node) bool {
	return len(n.Values) == 1 && n.Values[0] == kdl.String("")
}

// parser parses usage features given by one source.
type parser struct {
	where []string // The entity and the source, for messages.
}

// parse parses text in KDL v2.
func (p parser) parse(text string) []*kdl.Node {
	nodes, err := kdl.Parse(text)
	if err != nil {
		failf(p.where, "%v", err)
	}
	return nodes
}

// entity parses usage features of an entity at the place
// given as properties and child nodes of its node, or as top-level nodes for the root.
func (p parser) entity(path []string, at Place, props []kdl.Prop, children []*kdl.Node) *Set {
	where := append(slices.Clone(p.where), path...)
	s := newSet()
	for _, prop := range props {
		spec := p.spec(where, at, prop.Name)
		if spec.form == childOnly {
			failf(where, "%s is not accepted as a property", spec.Name)
		}
		f := &feature{spec: spec, remove: prop.Value.IsNull()}
		if !f.remove {
			f.values = []kdl.Value{p.value(append(where, prop.Name), spec.values[0], prop.Value)}
		}
		s.add(where, f)
	}
	for _, n := range children {
		switch n.Name {
		case cmdNode, flagNode, argNode:
			key, set := p.nested(path, at, n)
			if s.find(key) != nil {
				failf(where, "%s is given twice", key)
			}
			s.entities = append(s.entities, &describedEntity{key: key, set: set})
		default:
			s.add(where, p.feature(where, p.spec(where, at, n.Name), n))
		}
	}
	return s
}

// nested parses a node n of a descendant entity of the entity at the place.
func (p parser) nested(path []string, at Place, n *kdl.Node) (string, *Set) {
	where := append(slices.Clone(p.where), path...)
	if at == AtArg {
		failf(where, "%s is not accepted on flags and positionals", n.Name)
	}
	if len(n.Values) != 1 {
		failf(where, "%s needs one value: its name in the spec", n.Name)
	}
	if _, ok := n.Values[0].Interface().(string); !ok {
		failf(where, "%s needs a string value: its name in the spec", n.Name)
	}
	key := entityKey(n.Name, n.Values[0])
	nestedAt := AtArg
	if n.Name == cmdNode {
		nestedAt = AtCmd
	}
	return key, p.entity(append(slices.Clone(path), key), nestedAt, n.Props, n.Children)
}

// spec returns the spec of the feature name accepted at the place.
func (parser) spec(where []string, at Place, name string) *Spec {
	spec := lookup(name)
	if spec == nil {
		failf(where, "%s is not accepted", name)
	}
	if spec.Places&at == 0 {
		failf(where, "%s is not accepted on %s", name, at)
	}
	return spec
}

// feature parses the node n of a feature spec.
func (p parser) feature(where []string, spec *Spec, n *kdl.Node) *feature {
	where = append(slices.Clone(where), n.Name)
	if len(n.Values) == 1 && n.Values[0].IsNull() && len(n.Props) == 0 && len(n.Children) == 0 &&
		spec != choiceSpec {
		return &feature{spec: spec, remove: true}
	}
	f := &feature{
		spec:   spec,
		values: p.values(where, spec, n.Values),
		props:  p.props(where, spec, n.Props),
	}
	for _, child := range n.Children {
		if len(spec.Children) == 0 || child.Name != spec.Children[0].Name {
			failf(where, "child node %s is not accepted", child.Name)
		}
		choice := p.feature(where, spec.Children[0], child)
		if f.choice(choice.values[0]) != nil {
			failf(where, "choice %s is given twice", choice.values[0])
		}
		f.choices = append(f.choices, choice)
	}
	return f
}

// values parses values of a feature spec.
func (p parser) values(where []string, spec *Spec, values []kdl.Value) []kdl.Value {
	if len(values) < len(spec.values) || (len(values) > len(spec.values) && spec.rest.typ == typeNone) {
		failf(where, "%d values given, %s", len(values), spec.arity())
	}
	parsed := make([]kdl.Value, len(values))
	for i, v := range values {
		vs := spec.rest
		if i < len(spec.values) {
			vs = spec.values[i]
		}
		parsed[i] = p.value(where, vs, v)
	}
	return parsed
}

// props parses properties of a feature spec and returns them in the order of the spec.
// Properties of features merged by parts may be `#null`.
func (p parser) props(where []string, spec *Spec, props []kdl.Prop) []kdl.Prop {
	for _, prop := range props {
		if !slices.ContainsFunc(spec.Props, func(ps PropSpec) bool { return ps.Name == prop.Name }) {
			failf(where, "property %s is not accepted", prop.Name)
		}
	}
	merged := spec == choicesSpec || spec == choiceSpec
	var parsed []kdl.Prop
	for _, ps := range spec.Props {
		i := slices.IndexFunc(props, func(prop kdl.Prop) bool { return prop.Name == ps.Name })
		if i == -1 {
			continue
		}
		v := props[i].Value
		if !merged || !v.IsNull() {
			v = p.value(append(where, ps.Name), ps.value, v)
		}
		parsed = append(parsed, kdl.Prop{Name: ps.Name, Value: v})
	}
	return parsed
}

// value returns v as a value of the type of vs.
// Values are built anew so that they are written as usage expects:
// e.g. gokdl2 writes 1e2 as 100.0, which usage refuses as an integer.
func (parser) value(where []string, vs valueSpec, v kdl.Value) kdl.Value {
	switch x := v.Interface().(type) {
	case string:
		if vs.typ == typeString {
			if len(vs.enum) > 0 && !slices.Contains(vs.enum, x) {
				failf(where, "%s is not one of %s", v, strings.Join(vs.enum, ", "))
			}
			return kdl.String(x)
		}
	case bool:
		if vs.typ == typeBool {
			return kdl.Bool(x)
		}
	case int64:
		if vs.typ == typeInt || (vs.typ == typeUint && x >= 0) {
			return kdl.Int(int(x))
		}
	}
	failf(where, "%s is not %s", v, vs.typ)
	panic("unreachable")
}
