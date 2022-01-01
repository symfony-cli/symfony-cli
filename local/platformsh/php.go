package platformsh

import "strings"

func IsPhpExtensionAvailable(ext, phpVersion string) bool {
	versions, ok := availablePHPExts[strings.ToLower(ext)]
	if !ok {
		return false
	}
	for _, v := range versions {
		if v == phpVersion {
			return true
		}
	}
	return false
}
