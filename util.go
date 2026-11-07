package kongusage

import (
	"strings"
	"unicode"
)

// trimAngles removes one pair of angle brackets around s.
func trimAngles(s string) string {
	if len(s) >= 2 && s[0] == '<' && s[len(s)-1] == '>' {
		return s[1 : len(s)-1]
	}
	return s
}

// hasEllipsis reports whether s has "..." or "…".
func hasEllipsis(s string) bool {
	return strings.Contains(s, "...") || strings.Contains(s, "…")
}

// hasSpace reports whether s has whitespace.
// [unicode.IsSpace] matches the White_Space property, as split_whitespace in Rust,
// which usage uses to split the name of a flag.
func hasSpace(s string) bool {
	return strings.ContainsFunc(s, unicode.IsSpace)
}
