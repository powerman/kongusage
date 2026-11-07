package kongusage

import (
	urlpkg "net/url"
	"reflect"
	"slices"
	"time"
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

// enum returns the values of the enum of v, or nil when it has none.
func enum(v *kong.Value) []string {
	if v.Enum == "" {
		return nil
	}
	return v.EnumSlice()
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
		// Kong's mappers of file contents read a whole file into a []byte.
		return v.Tag.Type != "filecontent" &&
			v.Target.Type() != reflect.TypeFor[kong.FileContentFlag]()
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

// completion returns the type of completion of the value v
// by docs/mapping.md "Completion of values", or "" to complete files.
// The registry defaults holds only Kong's default mappers.
func completion(defaults *kong.Registry, v *kong.Value) string {
	// A custom mapper may take any value.
	if !sameMapper(v.Mapper, defaults.ForNamedValue(v.Tag.Type, v.Target)) {
		return ""
	}
	switch v.Tag.Type {
	case "":
		return typeCompletion(defaults, v.Target.Type())
	case "existingdir":
		return "dir"
	default: // path, existingfile, filecontent.
		return ""
	}
}

// typeCompletion returns the type of completion of a value of the type typ
// mapped by Kong's default mapper of typ, or "" to complete files.
func typeCompletion(defaults *kong.Registry, typ reflect.Type) string {
	switch typ {
	case reflect.TypeFor[kong.ChangeDirFlag]():
		return "dir"
	case reflect.TypeFor[time.Duration](), reflect.TypeFor[time.Time](), reflect.TypeFor[*urlpkg.URL]():
		return "none"
	}
	var unnamed reflect.Type
	switch typ.Kind() {
	case reflect.Slice:
		unnamed = reflect.SliceOf(typ.Elem())
	case reflect.Pointer:
		unnamed = reflect.PointerTo(typ.Elem())
	default:
		unnamed = numberType(typ.Kind())
	}
	// A named type may have its own mapper, e.g. by implementing kong.MapperValue.
	if unnamed == nil || !sameMapper(defaults.ForType(typ), defaults.ForType(unnamed)) {
		return ""
	}
	if typ.Kind() == reflect.Slice || typ.Kind() == reflect.Pointer {
		// Kong maps elements by the mappers of their types,
		// which are assumed to be Kong's: they are kept by its unexported registry.
		return typeCompletion(defaults, typ.Elem())
	}
	return "none"
}

// numberType returns the unnamed number type of the kind, or nil for other kinds.
func numberType(kind reflect.Kind) reflect.Type {
	switch kind {
	case reflect.Int:
		return reflect.TypeFor[int]()
	case reflect.Int8:
		return reflect.TypeFor[int8]()
	case reflect.Int16:
		return reflect.TypeFor[int16]()
	case reflect.Int32:
		return reflect.TypeFor[int32]()
	case reflect.Int64:
		return reflect.TypeFor[int64]()
	case reflect.Uint:
		return reflect.TypeFor[uint]()
	case reflect.Uint8:
		return reflect.TypeFor[uint8]()
	case reflect.Uint16:
		return reflect.TypeFor[uint16]()
	case reflect.Uint32:
		return reflect.TypeFor[uint32]()
	case reflect.Uint64:
		return reflect.TypeFor[uint64]()
	case reflect.Float32:
		return reflect.TypeFor[float32]()
	case reflect.Float64:
		return reflect.TypeFor[float64]()
	default:
		return nil
	}
}

// sameMapper reports whether the mappers a and b are the same Kong's mapper.
// Mappers made by one function literal are taken as the same.
func sameMapper(a, b kong.Mapper) bool {
	va, vb := reflect.ValueOf(a), reflect.ValueOf(b)
	if !va.IsValid() || !vb.IsValid() || va.Type() != vb.Type() {
		return false
	}
	if va.Kind() == reflect.Func {
		return va.Pointer() == vb.Pointer()
	}
	return true
}

// isRequired reports whether Kong requires the flag f of the command cmd,
// unless it is a member of an xor or lastwins group.
// Kong turns required on and-group members into required of the whole group.
func isRequired(cmd *kong.Node, f *kong.Flag) bool {
	return f.Tag.Required || andGroupRequired(cmd, f)
}

// choiceGroup is an xor or a lastwins group of flags of one command.
type choiceGroup struct {
	lastWins bool
	name     string
	flags    []*kong.Flag
}

// choiceGroups returns the xor and lastwins groups of flags of the command cmd
// in the order of their first use.
func choiceGroups(cmd *kong.Node) []*choiceGroup {
	var groups []*choiceGroup
	add := func(lastWins bool, name string, f *kong.Flag) {
		i := slices.IndexFunc(groups, func(g *choiceGroup) bool {
			return g.lastWins == lastWins && g.name == name
		})
		if i == -1 {
			i = len(groups)
			groups = append(groups, &choiceGroup{lastWins: lastWins, name: name})
		}
		groups[i].flags = append(groups[i].flags, f)
	}
	for _, f := range cmd.Flags {
		for _, name := range f.Xor {
			add(false, name, f)
		}
		for _, name := range f.LastWins {
			add(true, name, f)
		}
	}
	return groups
}

// choiceGroupRequired reports whether Kong requires one of the flags
// of an xor or lastwins group of the command cmd:
// at least two of them have to be required, contrary to Kong's README (issue #516).
func choiceGroupRequired(cmd *kong.Node, flags []*kong.Flag) bool {
	required := 0
	for _, f := range flags {
		if isRequired(cmd, f) {
			required++
		}
	}
	return required > 1
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
