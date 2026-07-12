package php

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
)

// isNixWrapper checks if the file is a Nix wrapper binary with a valid wrapped PHP script.
// Nix wraps executables with a compiled binary that sets up the environment and calls the
// actual script (stored as .<name>-wrapped in the same directory). Nix profiles expose these
// wrappers via symlinks (e.g., ~/.nix-profile/bin/composer -> /nix/store/.../bin/composer),
// so symlinks are resolved first to locate the companion wrapped file in the Nix store.
// Note: PHP must also be available since the Executor still requires a PHP binary for
// configuration.
func isNixWrapper(path string) bool {
	if path == "" {
		return false
	}

	// Resolve symlinks to handle Nix profile paths
	// (e.g., /etc/profiles/per-user/foo/bin/composer -> /nix/store/.../bin/composer)
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}

	dir := filepath.Dir(realPath)
	base := filepath.Base(realPath)
	wrappedPath := filepath.Join(dir, "."+base+"-wrapped")

	if _, err := os.Stat(wrappedPath); err == nil {
		return isPHPScriptDirect(wrappedPath)
	}

	return false
}

// isPHPScriptDirect checks if the file itself is a PHP script
func isPHPScriptDirect(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	byteSlice, _, err := reader.ReadLine()
	if err != nil {
		return false
	}

	if bytes.Equal(byteSlice, []byte("<?php")) {
		return true
	}

	return bytes.HasPrefix(byteSlice, []byte("#!/")) && bytes.HasSuffix(byteSlice, []byte("php"))
}
