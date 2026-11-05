package kongusage

import (
	"reflect"

	"github.com/alecthomas/kong"
)

// Config controls generation of the usage spec.
// The zero value of every field means "unset, use default".
// Usage features of the CLI are supplied by [KDLProvider] instead.
type Config struct {
	// NoDisplayOrder disables `display_order` of commands,
	// which keeps the order of their declaration in help of usage;
	// without it help of usage sorts commands by name.
	NoDisplayOrder bool
}

// ConfigProvider may be implemented by the root of a Kong CLI
// to give [Flag] and [CheckSpec] a [Config].
type ConfigProvider interface {
	UsageConfig() Config
}

// resolve applies defaults to c, normalizes and validates it.
func (c Config) resolve() (Config, error) {
	return c, nil
}

// rootConfig returns the Config of the root of k, or zero Config.
func rootConfig(k *kong.Kong) Config {
	if cli, ok := reflect.TypeAssert[ConfigProvider](k.Model.Target.Addr()); ok {
		return cli.UsageConfig()
	}
	return Config{}
}
