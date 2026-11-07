package kongusage

import (
	"reflect"
	"slices"

	"github.com/alecthomas/kong"
)

func isVersionFlag(f *kong.Flag) bool {
	return f.Target.Type() == reflect.TypeFor[kong.VersionFlag]()
}

// valueName returns Kong's placeholder of the flag f
// without the default and without the tail of a slice or a map.
// TODO(stage 3): docs/mapping.md "Value names".
func valueName(f *kong.Flag) string {
	tag := *f.Tag
	tag.Sep = -1
	tag.MapSep = -1
	value := *f.Value
	value.Tag = &tag
	flag := *f
	flag.Value = &value
	value.Flag = &flag
	if _, ok := f.Mapper.(kong.PlaceHolderProvider); !ok && f.PlaceHolder == "" {
		value.HasDefault = false
	}
	return flag.FormatPlaceHolder()
}

// negation returns the negative spelling of a negatable flag, as Kong derives it.
func negation(f *kong.Flag) string {
	// Kong marks `negatable:""` by "_" in the tag.
	if f.Tag.Negatable == "_" {
		return "--no-" + f.Name
	}
	return "--" + f.Tag.Negatable
}

// isCumulative reports whether Kong accumulates values of v.
// Value.IsSlice is false for a named slice type, which Kong still accumulates.
func isCumulative(v *kong.Value) bool {
	switch v.Target.Kind() {
	case reflect.Slice, reflect.Map:
		return true
	default:
		return false
	}
}

// hasRun reports whether v or its address has the method Run, as Kong looks it up.
func hasRun(v reflect.Value) bool {
	if !v.IsValid() {
		return false
	}
	if v.MethodByName("Run").IsValid() {
		return true
	}
	return v.CanAddr() && v.Addr().MethodByName("Run").IsValid()
}

// subcommandRequired reports whether Kong requires a subcommand of n,
// as checkMissingChildren of Kong does.
func subcommandRequired(n *kong.Node) bool {
	if n.DefaultCmd != nil || hasRun(n.Target) {
		return false
	}
	for _, child := range n.Children {
		if child.Type == kong.CommandNode && !child.Hidden {
			return true
		}
	}
	return false
}

// andGroupRequired reports whether Kong requires the flag f because of its and-groups.
// Kong sets required of a member by its last group only (checkMissingFlags).
func andGroupRequired(cmd *kong.Node, f *kong.Flag) bool {
	if len(f.And) == 0 {
		return false
	}
	last := f.And[len(f.And)-1]
	for _, other := range cmd.Flags {
		if other.Tag.Required && slices.Contains(other.And, last) {
			return true
		}
	}
	return false
}

// andMembers returns the names of other members of the and-groups of the flag f.
func andMembers(cmd *kong.Node, f *kong.Flag) []string {
	return groupMembers(cmd, f, func(f *kong.Flag) []string { return f.And })
}

// groupMembers returns the names of other flags of the command cmd
// sharing a group with the flag f, in the order of declaration.
// Kong also groups flags of different commands of one command line,
// which usage cannot express: their names are unknown in the scope of an ancestor.
func groupMembers(cmd *kong.Node, f *kong.Flag, groups func(*kong.Flag) []string) []string {
	var names []string
	for _, other := range cmd.Flags {
		if other == f {
			continue
		}
		for _, g := range groups(f) {
			if slices.Contains(groups(other), g) {
				names = append(names, "--"+other.Name)
				break
			}
		}
	}
	return names
}
