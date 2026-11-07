package kdl

import (
	"errors"
	"fmt"
	"maps"
	"math/big"
	"slices"
	"strings"

	gokdl "github.com/njreid/gokdl2"
	"github.com/njreid/gokdl2/document"
)

// Errors of Parse.
var (
	ErrTypeAnnotation = errors.New("type annotations are not supported")
	ErrExpression     = errors.New("expressions in backticks are not KDL v2")
	ErrOutOfRange     = errors.New("integer is out of range")
	ErrValue          = errors.New("unsupported value")
)

// Parse parses s as a KDL v2 document.
//
// It refuses type annotations and gokdl2's backtick expressions, which are not KDL v2,
// and integers which do not fit into int64.
// Properties of a parsed node are sorted by name, as gokdl2 does not give their order;
// a property given twice keeps the last value, as the KDL spec requires.
func Parse(s string) ([]*Node, error) {
	// The default version of gokdl2 falls back to KDL v1 silently.
	// gokdl2 v0.6.1 refuses a literal U+FFFD as invalid UTF-8, unlike usage: `\u{FFFD}` works.
	doc, err := gokdl.ParseWithOptions(strings.NewReader(s), gokdl.ParseOptions{Version: gokdl.ParseVersionV2})
	if err != nil {
		return nil, err
	}
	return parseNodes(doc.Nodes)
}

func parseNodes(docNodes []*document.Node) ([]*Node, error) {
	var nodes []*Node
	for _, n := range docNodes {
		node, err := parseNode(n)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func parseNode(n *document.Node) (*Node, error) {
	name := n.Name.ValueString()
	if n.Type != "" {
		return nil, fmt.Errorf("node %s: %w: (%s)", name, ErrTypeAnnotation, n.Type)
	}
	node := &Node{Name: name}
	for _, arg := range n.Arguments {
		v, err := parseValue(arg)
		if err != nil {
			return nil, fmt.Errorf("node %s: %w", name, err)
		}
		node.Values = append(node.Values, v)
	}
	props := n.Properties.Unordered()
	for _, key := range slices.Sorted(maps.Keys(props)) {
		v, err := parseValue(props[key])
		if err != nil {
			return nil, fmt.Errorf("node %s: property %s: %w", name, key, err)
		}
		node.Props = append(node.Props, Prop{Name: key, Value: v})
	}
	children, err := parseNodes(n.Children)
	if err != nil {
		return nil, fmt.Errorf("node %s: %w", name, err)
	}
	node.Children = children
	return node, nil
}

func parseValue(v *document.Value) (Value, error) {
	if v.Type != "" {
		return Value{}, fmt.Errorf("%w: (%s)", ErrTypeAnnotation, v.Type)
	}
	switch x := v.Value.(type) {
	case string, int64, float64, bool:
		return Value{v: x}, nil
	case nil:
		return Value{v: null{}}, nil
	case *big.Int:
		return Value{}, fmt.Errorf("%w: %s", ErrOutOfRange, x)
	case document.Expression:
		return Value{}, ErrExpression
	default:
		return Value{}, fmt.Errorf("%w: %#v", ErrValue, x)
	}
}
