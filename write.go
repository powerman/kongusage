package kongusage

import (
	"io"

	"github.com/alecthomas/kong"

	"github.com/powerman/kongusage/internal/kdl"
)

// WriteSpec writes the usage spec of the CLI k to w in KDL.
//
// It panics on an invalid cfg or on usage features of k it cannot merge
// (use [CheckSpec] in your tests to ensure this does not happen).
// Returns an error when writing to w fails.
func WriteSpec(w io.Writer, k *kong.Kong, cfg Config) error {
	_, err := writeSpec(w, k, cfg)
	return err
}

// writeSpec writes the usage spec as WriteSpec does
// and returns the notes on silent fixes of the Kong model of k.
func writeSpec(w io.Writer, k *kong.Kong, cfg Config) ([]string, error) {
	cfg, err := cfg.resolve()
	if err != nil {
		panic(err)
	}
	c := newConverter(k, cfg)
	return c.Notes(), kdl.Write(w, c.Nodes())
}
