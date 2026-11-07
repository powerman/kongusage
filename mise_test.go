package kongusage_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/powerman/check"

	"github.com/powerman/kongusage"
	"github.com/powerman/kongusage/internal/usagecli"
)

type (
	testMiseCLI struct {
		Version kong.VersionFlag `                                           help:"Show version."`
		Mode    string           `       default:"fast"     enum:"fast,slow" help:"Mode."`
		Serve   testMiseServeCmd `cmd:"" default:"withargs"                  help:"Serve."`
		Other   testLeafCmd      `cmd:""                                     help:"Other."`
	}
	testMiseServeCmd struct {
		Port int `help:"Port."`
	}
)

// TestMiseMount checks completion of a spec mounted by a mise task,
// with the limitations of a mount (docs/usage.md "Mount").
func TestMiseMount(tt *testing.T) {
	tt.Parallel()
	t := check.T(tt).MustAll()
	if os.Getenv(usagecli.SkipEnv) != "" {
		t.Skipf("mise is skipped by %s", usagecli.SkipEnv)
	}
	mise, err := exec.LookPath("mise")
	if err != nil {
		t.Fatalf("mise is required (set %s=1 to skip): %v", usagecli.SkipEnv, err)
	}

	dir := t.TempDir()
	spec := spec(t, &testMiseCLI{}, kongusage.Config{}, kong.Vars{"version": "1.2.3"})
	t.Nil(os.WriteFile(filepath.Join(dir, "spec.kdl"), []byte(spec), 0o644))
	t.Nil(os.WriteFile(filepath.Join(dir, "mise.toml"), []byte(`
[tasks.t]
usage = 'mount run="cat spec.kdl"'
raw_args = true
run = 'echo'
`), 0o644))

	tests := []struct {
		line string
		want string
	}{
		// default_subcommand has no effect.
		{"t ", "other\nserve\n"},
		// Neither default_subcommand_flags (no --port) nor version (no -V) has an effect.
		{"t -", "-h\n--help\n--mode\n--version\n"},
		{"t --mode ", "fast\nslow\n"},
		{"t serve -", "-h\n--help\n--mode\n--port\n--version\n"},
		{"t serve --port ", ""},
	}
	for _, tc := range tests {
		cmd := exec.CommandContext(t.Context(), mise,
			"__complete_word__", "--shell", "bash", "--line", "mise run "+tc.line)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "MISE_TRUSTED_CONFIG_PATHS="+dir)
		out, err := cmd.CombinedOutput()
		t.Nil(err, string(out))
		t.Equal(string(out), tc.want, tc.line)
	}
}
