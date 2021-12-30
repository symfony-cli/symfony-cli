package util

import "os"

func InCloud() bool {
	return os.Getenv("PLATFORM_PROJECT_ENTROPY") != ""
}
