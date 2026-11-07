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
// Strings are sanitized to be valid in KDL v2 (see sanitize).
// It panics on an invalid name or Value.
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
	node.SetName(checkName(n.Name))
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
		// AddPropertyValue allocates Properties, unlike Properties.Add.
		node.AddPropertyValue(checkName(p.Name), p.Value.document(), "")
	}
	for _, child := range n.Children {
		node.AddNode(child.document())
	}
	return node
}

// checkName panics on a name which gokdl2 v0.6.1 cannot write as valid KDL v2.
// Remove the checks once gokdl2 is fixed.
func checkName(name string) string {
	// The generator escapes a quoted name like a string value, wrong for KDL v2 (see sanitize),
	// and sanitizing a name would change it silently.
	if sanitize(name) != name {
		panic(fmt.Sprintf("invalid KDL name %q: gokdl2 cannot escape it", name))
	}
	// The generator writes a name bare when its characters are valid in an identifier string,
	// and does not check the rest of the grammar of KDL v2.
	// Property names are written bare by the same check.
	bare := !strings.HasPrefix((&document.Value{Value: name}).NodeNameStringV2(), `"`)
	if bare && !isIdentifierV2(name) {
		panic(fmt.Sprintf("invalid KDL name %q: gokdl2 writes it bare", name))
	}
	return name
}

// isIdentifierV2 reports whether s, made of characters valid in an identifier string,
// is an identifier string by the grammar of KDL v2:
// not a keyword and not looking like a number.
func isIdentifierV2(s string) bool {
	switch s {
	case "true", "false", "null", "inf", "-inf", "nan":
		return false
	}
	if strings.HasPrefix(s, "+") || strings.HasPrefix(s, "-") {
		s = s[1:]
	}
	s = strings.TrimPrefix(s, ".")
	return s == "" || s[0] < '0' || '9' < s[0]
}

func (v Value) document() *document.Value {
	switch x := v.v.(type) {
	case string:
		// With no flags the generator writes a single-line quoted string:
		// its multi-line strings have closing quotes on the last line of the text,
		// which is invalid KDL v2.
		return &document.Value{Value: sanitize(x)}
	case int64, bool:
		return &document.Value{Value: x}
	default:
		panic(fmt.Sprintf("invalid KDL value %#v", v.v))
	}
}

// sanitize makes s valid in a KDL v2 quoted string and in a comment.
// The generator escapes control characters as `\uXX` and invalid UTF-8 as `\ufffd`,
// which is invalid KDL v2 (it needs `\u{XX}`),
// and does not escape other characters which KDL v2 does not allow in a document.
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
