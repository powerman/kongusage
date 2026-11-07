package usagekdl_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/powerman/check"

	"github.com/powerman/kongusage/internal/usagekdl"
)

// TestSpecs checks docs/usage-features.md "Accepted" lists the accepted features.
func TestSpecs(tt *testing.T) {
	tt.Parallel()
	t := check.T(tt).MustAll()

	doc, err := os.ReadFile(filepath.Join("..", "..", "docs", "usage-features.md"))
	t.Nil(err)
	_, accepted, ok := strings.Cut(string(doc), "\n## Accepted\n")
	t.True(ok)
	accepted, _, _ = strings.Cut(accepted, "\n## ")

	// A feature is given as `name` or `name <values>`, with its properties in parentheses.
	featureRe := regexp.MustCompile("`([a-z_]+)(?: [^`]*)?`(?: \\(([^)]*)\\))?")
	got := make(map[string]map[string][]string)
	var section string
	for line := range strings.SplitSeq(accepted, "\n") {
		if title, isTitle := strings.CutSuffix(line, ":"); isTitle && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "-") {
			section = title
			got[section] = make(map[string][]string)
			continue
		}
		if section == "" {
			continue
		}
		for _, m := range featureRe.FindAllStringSubmatch(line, -1) {
			props, found := got[section][m[1]]
			if !found {
				props = []string{}
			}
			if m[2] != "" {
				props = strings.Split(strings.NewReplacer("`", "", " ", "").Replace(m[2]), ",")
			}
			got[section][m[1]] = props
		}
	}
	t.DeepEqual(got, acceptedFeatures())
}

// acceptedFeatures returns the names of accepted usage features with the names of their
// properties by the sections of docs/usage-features.md "Accepted".
func acceptedFeatures() map[string]map[string][]string {
	sections := make(map[string]map[string][]string)
	add := func(section string, spec *usagekdl.Spec) {
		if sections[section] == nil {
			sections[section] = make(map[string][]string)
		}
		props := make([]string, 0, len(spec.Props))
		for _, p := range spec.Props {
			props = append(props, p.Name)
		}
		sections[section][spec.Name] = props
	}
	for _, spec := range usagekdl.Specs {
		places := spec.Places
		switch {
		case places == usagekdl.AtRoot:
			add("The root", spec)
		case places&(usagekdl.AtRoot|usagekdl.AtCmd) == usagekdl.AtRoot|usagekdl.AtCmd:
			add("The root and commands", spec)
		case places&usagekdl.AtCmd != 0:
			add("Commands", spec)
		}
		if places&usagekdl.AtArg != 0 {
			add("Flags and positionals", spec)
			for _, child := range spec.Children {
				add("Flags and positionals", child)
			}
		}
	}
	return sections
}
