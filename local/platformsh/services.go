package platformsh

type service struct {
	Type     string
	Versions serviceVersions
}

type serviceVersions struct {
	Deprecated []string
	Supported  []string
}

func ServiceLastVersion(name string) string {
	for _, s := range availableServices {
		if s.Type == name {
			versions := s.Versions.Supported
			if len(versions) == 0 {
				versions = s.Versions.Deprecated
			}
			if len(versions) > 0 {
				return versions[len(versions)-1]
			}
		}
	}
	return ""
}
