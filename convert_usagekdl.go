package kongusage

import (
	"reflect"
	"slices"

	"github.com/alecthomas/kong"

	"github.com/powerman/kongusage/internal/usagekdl"
)

// usageKDLTag is the name of the tag with usage features of a field.
const usageKDLTag = "usage-kdl"

// rootUsageKDL returns usage features given by UsageKDL of the root.
func (c *converter) rootUsageKDL() *usagekdl.Set {
	app := c.k.Model
	ptr := app.Target.Addr()
	return usagekdl.ParseRoot(usageKDLSource([]string{app.Name}, ptr.Type().Elem()), usageKDL(ptr))
}

// commandUsageKDL returns usage features of the command n:
// those described by its ancestor, otherwise those given by UsageKDL of its type,
// changed by the tag of its field.
func (*converter) commandUsageKDL(n *kong.Node, described *usagekdl.Set) *usagekdl.Set {
	where := []string{n.FullPath()}
	fs := described
	if fs == nil {
		ptr := n.Target.Addr()
		fs = usagekdl.ParseCommand(usageKDLSource(where, ptr.Type().Elem()), usageKDL(ptr))
	}
	return fs.Merge(tagUsageKDL(where, n.Tag, usagekdl.AtCmd))
}

// valueUsageKDL returns usage features of the flag or the positional where with the value v:
// those described by its ancestor, otherwise those given by UsageKDL of its type,
// changed by the tag of its field.
func (*converter) valueUsageKDL(where []string, described *usagekdl.Set, v *kong.Value) *usagekdl.Set {
	fs := described
	if fs == nil {
		typ := v.Target.Type()
		if typ.Kind() == reflect.Pointer {
			typ = typ.Elem()
		}
		// A new value, so that a value receiver is not called on a nil pointer.
		fs = usagekdl.ParseValue(usageKDLSource(where, typ), usageKDL(reflect.New(typ)))
	}
	fs = fs.Merge(tagUsageKDL(where, v.Tag, usagekdl.AtArg))
	if v.Flag != nil && !takesValue(v.Flag) {
		fs.CheckNoValue(where)
	} else {
		fs.CheckValue(where, enum(v))
	}
	return fs
}

// tagUsageKDL returns usage features of the entity where at the place
// given by the tag of its field.
func tagUsageKDL(where []string, tag *kong.Tag, at usagekdl.Place) *usagekdl.Set {
	text := tag.Get(usageKDLTag)
	if text == "" {
		return nil
	}
	return usagekdl.ParseTag(append(slices.Clone(where), usageKDLTag+" tag"), text, at)
}

// usageKDLSource returns where followed by UsageKDL of the type typ,
// which names the source of usage features in messages.
func usageKDLSource(where []string, typ reflect.Type) []string {
	return append(slices.Clone(where), "UsageKDL of "+typ.String())
}

// usageKDL returns UsageKDL of the value ptr points to, or "" when it has none.
func usageKDL(ptr reflect.Value) string {
	if p, ok := reflect.TypeAssert[KDLProvider](ptr); ok {
		return p.UsageKDL()
	}
	return ""
}
