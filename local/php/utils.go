package php

import (
	"bufio"
	"bytes"
	"os"
)

// isPHPScript checks that the provided file is indeed a phar/PHP script (not a .bat file)
func isPHPScript(path string) bool {
	if path == "" {
		return false
	}
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
