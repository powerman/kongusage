package kdl

import (
	"fmt"
	"io"
	"strings"
	"unicode"

	gokdl "github.com/njreid/gokdl2"
	"github.com/njreid/gokdl2/document"
)

// The type of GenerateOptions.Version is in an internal package of gokdl2.
const versionV2 = 2

// Write writes the nodes to w as a KDL v2 document.
//
// The generator escapes strings, so any string is written as valid KDL v2;
// only comments are sanitized (see sanitize).
// It panics on an invalid Value.
func Write(w io.Writer, nodes []*Node) error {
	doc := document.New()
	for _, n := range nodes {
		doc.AddNode(n.document())
	}
	return gokdl.GenerateWithOptions(doc, w, gokdl.GenerateOptions{
		Indent:  "    ",
		Version: versionV2,
	})
}

func (n *Node) document() *document.Node {
	node := document.NewNode()
	node.SetName(n.Name)
	if len(n.Comment) > 0 {
		var lines []string
		for _, line := range n.Comment {
			for l := range strings.SplitSeq(sanitize(line), "\n") {
				lines = append(lines, strings.TrimRight("// "+l, " \t"))
			}
		}
		// The generator writes a newline after each line of Before,
		// and After is written without a space after the node.
		node.Comment = &document.Comment{Before: []byte(strings.Join(lines, "\n"))}
	}
	for _, v := range n.Values {
		node.AddArgumentValue(v.document())
	}
	for _, p := range n.Props {
		node.AddPropertyValue(p.Name, p.Value.document(), "")
	}
	for _, child := range n.Children {
		node.AddNode(child.document())
	}
	return node
}

func (v Value) document() *document.Value {
	switch x := v.v.(type) {
	case string, int64, bool:
		return &document.Value{Value: x}
	default:
		panic(fmt.Sprintf("invalid KDL value %#v", v.v))
	}
}

// sanitize makes s valid in a KDL v2 comment line,
// which the generator writes as is.
//
// Newlines of KDL v2 become `\n`;
// other control characters but `\t`, BOM and bidi controls are removed;
// invalid UTF-8 becomes U+FFFD.
func sanitize(s string) string {
	s = strings.ToValidUTF8(s, "\uFFFD")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\n', r == '\t':
			return r
		case r == '\r', r == '\v', r == '\f', r == '\u0085', r == '\u2028', r == '\u2029':
			return '\n'
		case unicode.IsControl(r),
			r == '\uFEFF',
			r == '\u200E', r == '\u200F',
			'\u202A' <= r && r <= '\u202E',
			'\u2066' <= r && r <= '\u2069':
			return -1
		default:
			return r
		}
	}, s)
}
