package kongusage

import "github.com/alecthomas/kong"

// Notes returns the notes on silent fixes of the Kong model of k.
func Notes(k *kong.Kong) []string {
	return newConverter(k, Config{}).Notes()
}
