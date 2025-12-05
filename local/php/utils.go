package php

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
)

// isPHPScript checks that the provided file is indeed a phar/PHP script (not a .bat file)
// It also handles Nix wrappers that wrap the actual PHP script
func isPHPScript(path string) bool {
	if path == "" {
		return false
	}

	if isPHPScriptDirect(path) {
		return true
	}

	// Check for Nix-style wrappers (e.g., composer -> .composer-wrapped)
	return isNixWrapper(path)
}

// isNixWrapper checks if the file is a Nix wrapper binary with a valid wrapped PHP script
func isNixWrapper(path string) bool {
	if path == "" {
		return false
	}

	dir := filepath.Dir(path)
	base := filepath.Base(path)
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
