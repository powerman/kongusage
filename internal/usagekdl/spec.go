// Package usagekdl parses usage features of a CLI, given in KDL v2,
// merges them and adds them to the usage spec as docs/usage-features.md describes.
//
// Mistakes in usage features make it panic with [*Error].
package usagekdl

import "fmt"

// Place is a set of entities of the usage spec: the root, commands, flags and positionals.
type Place uint8

// Places of usage features.
const (
	AtRoot Place = 1 << iota
	AtCmd
	AtArg // A flag or a positional argument.
)

// String names the entities of one place p, for messages.
func (p Place) String() string {
	switch p {
	case AtRoot:
		return "the root"
	case AtCmd:
		return "commands"
	default:
		return "flags and positionals"
	}
}

// form tells how a usage feature is given on a command, a flag or a positional
// and how it is written there; on the root every feature is a node.
type form uint8

const (
	childOnly form = iota // A child node.
	textForm              // A property or a child node, written as a child node.
	shortForm             // A property or a child node, written as a property when it has one value.
)

// valueType is a type of a value or a property of a usage feature.
type valueType uint8

const (
	typeNone valueType = iota
	typeString
	typeBool
	typeInt
	typeUint
)

// String names the type with an article, for messages.
func (t valueType) String() string {
	switch t {
	case typeString:
		return "a string"
	case typeBool:
		return "a boolean"
	case typeInt:
		return "an integer"
	default:
		return "a non-negative integer"
	}
}

// valueSpec describes a value or a property of a usage feature.
type valueSpec struct {
	typ  valueType
	enum []string // Accepted strings, if limited.
}

//nolint:gochecknoglobals // Read-only shorthands for the table.
var (
	str     = valueSpec{typ: typeString}
	boolean = valueSpec{typ: typeBool}
	integer = valueSpec{typ: typeInt}
	natural = valueSpec{typ: typeUint}
)

func enum(values ...string) valueSpec {
	return valueSpec{typ: typeString, enum: values}
}

// PropSpec describes a property of a usage feature given as a node.
type PropSpec struct {
	Name  string
	value valueSpec
}

// Spec describes a usage feature which kongusage accepts:
// a node of the usage spec known to its min_usage_version.
type Spec struct {
	Name     string
	Places   Place
	Props    []PropSpec
	Children []*Spec // Accepted child nodes.

	form       form
	repeatable bool        // Given by several nodes, which change it only all at once.
	values     []valueSpec // Required values.
	rest       valueSpec   // Further values, if any.
}

// arity describes how many values s takes, for messages.
func (s *Spec) arity() string {
	switch {
	case s.rest.typ == typeNone && len(s.values) == 0:
		return "it takes no values"
	case s.rest.typ == typeNone && len(s.values) == 1:
		return "it takes one value"
	case s.rest.typ == typeNone:
		return fmt.Sprintf("it takes %d values", len(s.values))
	default:
		return fmt.Sprintf("it takes at least %d values", len(s.values))
	}
}

// lookup returns the spec of the accepted feature name, or nil.
func lookup(name string) *Spec {
	for _, spec := range Specs {
		if spec.Name == name {
			return spec
		}
	}
	return nil
}
