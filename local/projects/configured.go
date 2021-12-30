package projects

type ConfiguredProject struct {
	Port    int
	Scheme  string
	Domains []string
}

func GetConfiguredAndRunning(proxyProjects, runningProjects map[string]*ConfiguredProject) (map[string]*ConfiguredProject, error) {
	projects := proxyProjects
	for dir, project := range runningProjects {
		if p, ok := projects[dir]; ok {
			p.Port = project.Port
			p.Scheme = project.Scheme
		} else {
			projects[dir] = &ConfiguredProject{
				Port:   project.Port,
				Scheme: project.Scheme,
			}
		}
	}
	return projects, nil
}
