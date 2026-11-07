// Package kdl builds a tree of KDL nodes and writes it as a KDL v2 document.
package kdl

// Value is a value of a node or a property: a string, an integer or a boolean.
// The zero Value is invalid.
type Value struct {
	v any
}

// String returns a string Value.
func String(s string) Value { return Value{v: s} }

// Int returns an integer Value.
func Int(i int) Value { return Value{v: int64(i)} }

// Bool returns a boolean Value.
func Bool(b bool) Value { return Value{v: b} }

// Prop is a property of a node.
type Prop struct {
	Name  string
	Value Value
}

// Node is a KDL node.
// Write panics on a few names of nodes and properties which it cannot write.
type Node struct {
	Name string
	// Comment holds lines of the comment written before the node, without `//`.
	Comment  []string
	Values   []Value
	Props    []Prop
	Children []*Node
}

// NewNode returns a node with the name and the values.
func NewNode(name string, values ...Value) *Node {
	return &Node{Name: name, Values: values}
}

// SetProp sets the property name to v.
// It keeps the position of an existing property and appends a new one.
func (n *Node) SetProp(name string, v Value) *Node {
	for i := range n.Props {
		if n.Props[i].Name == name {
			n.Props[i].Value = v
			return n
		}
	}
	n.Props = append(n.Props, Prop{Name: name, Value: v})
	return n
}

// AddChild appends the children to n.
func (n *Node) AddChild(children ...*Node) *Node {
	n.Children = append(n.Children, children...)
	return n
}
