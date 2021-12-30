package php

import (
	"io"
	"os"
)

func shouldSignalBeIgnored(sig os.Signal) bool {
	return false
}

func symlink(oldname, newname string) error {
	source, err := os.Open(oldname)
	if err != nil {
		return err
	}
	defer source.Close()
	destination, err := os.Create(newname)
	if err != nil {
		return err
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)
	return err
}
