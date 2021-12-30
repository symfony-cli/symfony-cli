package util

import (
	"os"
	"path/filepath"
	"strings"
)

func IsGoRun() bool {
	// Unfortunately, Golang does not expose that we are currently using go run
	// So we detect the main binary is (or used to be ;)) "go" and then the
	// current binary is within a temp "go-build" directory.
	_, exe := filepath.Split(os.Getenv("_"))
	argv0, _ := os.Executable()

	return exe == "go" && strings.Contains(argv0, "go-build")
}
