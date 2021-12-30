package platformsh

func IsPhpExtensionAvailable(ext, phpVersion string) bool {
	versions, ok := availablePHPExts[ext]
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
