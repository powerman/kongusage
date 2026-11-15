package kongusage

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/alecthomas/kong"

	"github.com/powerman/kongusage/internal/usagekdl"
)

// usageKDLTag is the name of the tag with usage features of a field.
const usageKDLTag = "usage-kdl"

// rootUsageKDL returns usage features of the root:
// those given by its UsageKDL, changed by Config.NodeKDL.
func (c *converter) rootUsageKDL() *usagekdl.Set {
	app := c.k.Model
	where := []string{app.Name}
	ptr := app.Target.Addr()
	fs := usagekdl.ParseRoot(usageKDLSource(where, ptr.Type().Elem()), usageKDL(ptr))
	return c.mergeNodeKDL(fs, where, app.Node, usagekdl.AtRoot)
}

// commandUsageKDL returns usage features of the command n:
// those described by its ancestor, otherwise those given by UsageKDL of its type,
// changed by Config.NodeKDL and then by the tag of its field.
func (c *converter) commandUsageKDL(n *kong.Node, described *usagekdl.Set) *usagekdl.Set {
	where := []string{n.FullPath()}
	fs := described
	if fs == nil {
		ptr := n.Target.Addr()
		fs = usagekdl.ParseCommand(usageKDLSource(where, ptr.Type().Elem()), usageKDL(ptr))
	}
	fs = c.mergeNodeKDL(fs, where, n, usagekdl.AtCmd)
	return fs.Merge(tagUsageKDL(where, n.Tag, usagekdl.AtCmd))
}

// mergeNodeKDL returns usage features fs of the root or the command n at the place,
// named by where, changed by Config.NodeKDL.
func (c *converter) mergeNodeKDL(fs *usagekdl.Set, where []string, n *kong.Node, at usagekdl.Place) *usagekdl.Set {
	for i, hook := range c.cfg.NodeKDL {
		fs = fs.Merge(hookUsageKDL(where, "NodeKDL", i, hook(n), at))
	}
	return fs
}

// valueUsageKDL returns usage features of the flag or the positional where
// with the value v, declared by cmd:
// those described by its ancestor, otherwise those given by UsageKDL of its type,
// changed by Config.ValueKDL and then by the tag of its field.
func (c *converter) valueUsageKDL(cmd *kong.Node, where []string, described *usagekdl.Set, v *kong.Value) *usagekdl.Set {
	fs := described
	if fs == nil {
		typ := v.Target.Type()
		if typ.Kind() == reflect.Pointer {
			typ = typ.Elem()
		}
		// A new value, so that a value receiver is not called on a nil pointer.
		fs = usagekdl.ParseValue(usageKDLSource(where, typ), usageKDL(reflect.New(typ)))
	}
	for i, hook := range c.cfg.ValueKDL {
		fs = fs.Merge(hookUsageKDL(where, "ValueKDL", i, hook(cmd, v), usagekdl.AtArg))
	}
	fs = fs.Merge(tagUsageKDL(where, v.Tag, usagekdl.AtArg))
	if v.Flag != nil && !takesValue(v.Flag) {
		fs.CheckNoValue(where)
	} else {
		fs.CheckValue(where, enum(v))
	}
	return fs
}

// hookUsageKDL returns usage features of the entity where at the place
// given by text, which the hook i of the field name of Config returned in the format of the tag.
func hookUsageKDL(where []string, name string, i int, text string, at usagekdl.Place) *usagekdl.Set {
	// A hook may build text of several lines, which must not start with an empty one:
	// the name ParseTag prepends has to be on the line of the properties.
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	return usagekdl.ParseTag(append(slices.Clone(where), fmt.Sprintf("Config.%s[%d]", name, i)), text, at)
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
