package usagekdl

import (
	"fmt"
	"slices"
	"strings"
)

// Error is the value of a panic on a mistake in usage features.
type Error struct {
	msg string
}

func (e *Error) Error() string { return "UsageKDL: " + e.msg }

// failf panics with an Error on a mistake in usage features of the entity where.
func failf(where []string, format string, args ...any) {
	panic(&Error{msg: strings.Join(append(slices.Clone(where), fmt.Sprintf(format, args...)), ": ")})
}
