package php

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/symfony-cli/terminal"
)

// Bump whenever we want to be sure we get a recent version of the psh CLI
var internalVersion = []byte("3")

func InstallPlatformPhar(home string) error {
	cacheDir := filepath.Join(os.TempDir(), ".symfony", "platformsh", "cache")
	if _, err := os.Stat(cacheDir); err != nil {
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			return errors.WithStack(err)
		}
	}
	var versionPath = filepath.Join(cacheDir, "internal_version")
	dir := filepath.Join(home, ".platformsh", "bin")
	if _, err := os.Stat(filepath.Join(dir, "platform")); err == nil {
		// check "API version" (we never upgrade automatically the psh CLI except if we need to if our code would not be compatible with old versions)
		if v, err := os.ReadFile(versionPath); err == nil && bytes.Equal(v, internalVersion) {
			return nil
		}
	}

	spinner := terminal.NewSpinner(terminal.Stderr)
	spinner.PrefixText = "Download additional CLI tools..."
	spinner.Start()
	defer spinner.Stop()
	resp, err := http.Get("https://platform.sh/cli/installer")
	if err != nil {
		return errors.WithStack(err)
	}
	defer resp.Body.Close()
	installer, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.WithStack(err)
	}

	installerPath := filepath.Join(home, "platformsh-installer.php")
	_ = os.WriteFile(installerPath, installer, 0666)
	defer os.Remove(installerPath)

	var stdout bytes.Buffer
	e := &Executor{
		Dir:        home,
		BinName:    "php",
		Args:       []string{"php", installerPath},
		ExtraEnv:   []string{"PLATFORMSH_CLI_NO_INTERACTION=1"},
		SkipNbArgs: 1,
		Stdout:     &stdout,
		Stderr:     &stdout,
	}
	if ret := e.Execute(false); ret == 1 {
		return errors.Errorf("unable to setup platformsh CLI: %s", stdout.String())
	}

	return errors.WithStack(os.WriteFile(versionPath, internalVersion, 0644))
}
