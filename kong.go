package kongusage

import (
	"reflect"
	"slices"
	"unicode/utf8"

	"github.com/alecthomas/kong"
)

// versionVar returns the variable "version" given to k by kong.Vars,
// which kong.VersionFlag prints.
// Kong keeps the variables in an unexported field only.
// A newer Kong without the field gives "", which loses only `version`
// and is detected by the tests.
func versionVar(k *kong.Kong) string {
	vars := reflect.ValueOf(k).Elem().FieldByName("vars")
	if !vars.IsValid() || vars.Kind() != reflect.Map || vars.Type().Key().Kind() != reflect.String {
		return ""
	}
	v := vars.MapIndex(reflect.ValueOf("version").Convert(vars.Type().Key()))
	if !v.IsValid() || v.Kind() != reflect.String {
		return ""
	}
	return v.String()
}

// hyphenPrefixed reports whether k was built with kong.WithHyphenPrefixedParameters(true).
// Kong keeps the option in an unexported field only.
// A newer Kong without the field gives false, which loses only `allow_hyphen_values`
// and is detected by the tests.
func hyphenPrefixed(k *kong.Kong) bool {
	f := reflect.ValueOf(k).Elem().FieldByName("allowHyphenated")
	return f.IsValid() && f.Kind() == reflect.Bool && f.Bool()
}

func isVersionFlag(f *kong.Flag) bool {
	return f.Target.Type() == reflect.TypeFor[kong.VersionFlag]()
}

// takesValue reports whether the flag f takes a value, as Kong's help shows it.
func takesValue(f *kong.Flag) bool {
	return !f.IsBool() && !f.IsCounter()
}

// placeHolder returns Kong's placeholder of the flag f without the default
// and without the tail of a slice or a map.
func placeHolder(f *kong.Flag) string {
	return formatPlaceHolder(f, func(flag *kong.Flag) {
		if _, ok := flag.Mapper.(kong.PlaceHolderProvider); !ok && flag.PlaceHolder == "" {
			flag.HasDefault = false
		}
	})
}

// plainPlaceHolder returns the placeholder Kong gives the flag f
// without a placeholder, a default and a mapper providing a placeholder,
// and without the tail of a slice or a map:
// "KEY=VALUE" for a map, otherwise the name of its Go type or of the flag in upper case.
func plainPlaceHolder(f *kong.Flag) string {
	return formatPlaceHolder(f, func(flag *kong.Flag) {
		flag.Mapper = nil
		flag.PlaceHolder = ""
		flag.HasDefault = false
	})
}

// formatPlaceHolder returns Kong's placeholder of a copy of the flag f changed by edit,
// without the tail of a slice or a map.
func formatPlaceHolder(f *kong.Flag, edit func(*kong.Flag)) string {
	tag := *f.Tag
	tag.Sep = -1
	tag.MapSep = -1
	value := *f.Value
	value.Tag = &tag
	flag := *f
	flag.Value = &value
	value.Flag = &flag
	edit(&flag)
	return flag.FormatPlaceHolder()
}

// aliases returns the spellings of the aliases of the flag f, as Kong derives them.
func aliases(f *kong.Flag) []string {
	names := make([]string, len(f.Aliases))
	for i, alias := range f.Aliases {
		if utf8.RuneCountInString(alias) == 1 {
			names[i] = "-" + alias
		} else {
			names[i] = "--" + alias
		}
	}
	return names
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

// separator returns the separator Kong splits each value of a slice or map flag f by,
// or -1 when Kong does not split them.
// A custom mapper of a slice or a map is assumed to split them as Kong's mappers do.
func separator(f *kong.Flag) rune {
	if f.Target.Kind() == reflect.Map {
		return f.Tag.MapSep
	}
	return f.Tag.Sep
}

// hasSubcommands reports whether n has child commands, hidden ones included.
func hasSubcommands(n *kong.Node) bool {
	return slices.ContainsFunc(n.Children, func(child *kong.Node) bool {
		return child.Type == kong.CommandNode
	})
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
