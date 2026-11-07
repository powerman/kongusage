package usagekdl

import (
	"slices"

	"github.com/powerman/kongusage/internal/kdl"
)

// feature is a usage feature given by the author of the CLI.
type feature struct {
	spec    *Spec
	remove  bool // Given by `#null`, which removes the feature on merge.
	values  []kdl.Value
	props   []kdl.Prop // In the order of spec.Props; `#null` removes the property on merge.
	choices []*feature // Child nodes of choices.
}

// node returns f as a node.
func (f *feature) node() *kdl.Node {
	n := kdl.NewNode(f.spec.Name, f.values...)
	for _, p := range f.props {
		if !p.Value.IsNull() {
			n.SetProp(p.Name, p.Value)
		}
	}
	if f.spec != choicesSpec || len(f.choices) == 0 {
		return n
	}
	// usage 6.12.0–7.3.0 refuse a value given both as a value of choices and as a choice.
	n.Values = nil
	for _, v := range f.values {
		choice := f.choice(v)
		if choice == nil {
			choice = &feature{spec: choiceSpec, values: []kdl.Value{v}}
		}
		n.AddChild(choice.node())
	}
	for _, choice := range f.choices {
		if !slices.Contains(f.values, choice.values[0]) {
			n.AddChild(choice.node())
		}
	}
	return n
}

// choice returns the choice of the value of choices f, or nil.
func (f *feature) choice(value kdl.Value) *feature {
	for _, choice := range f.choices {
		if choice.values[0] == value {
			return choice
		}
	}
	return nil
}

// prop returns the property name of f, or the zero Value when it is not given or is removed.
func (f *feature) prop(name string) kdl.Value {
	for _, p := range f.props {
		if p.Name == name && !p.Value.IsNull() {
			return p.Value
		}
	}
	return kdl.Value{}
}

// mergeProps changes properties of f by the properties of src.
func (f *feature) mergeProps(src *feature) {
	props := make(map[string]kdl.Value)
	for _, p := range f.props {
		props[p.Name] = p.Value
	}
	for _, p := range src.props {
		if p.Value.IsNull() {
			delete(props, p.Name)
		} else {
			props[p.Name] = p.Value
		}
	}
	f.props = f.props[:0]
	for _, ps := range f.spec.Props {
		if v, ok := props[ps.Name]; ok {
			f.props = append(f.props, kdl.Prop{Name: ps.Name, Value: v})
		}
	}
}

// mergeChoices changes choices f by choices src.
func (f *feature) mergeChoices(src *feature) {
	if len(src.values) > 0 {
		f.values = src.values
	}
	f.mergeProps(src)
	for _, choice := range src.choices {
		if old := f.choice(choice.values[0]); old != nil {
			old.mergeProps(choice)
		} else {
			f.choices = append(f.choices, choice)
		}
	}
}
