/*
 * Copyright (c) 2021-present Fabien Potencier <fabien@symfony.com>
 *
 * This file is part of Symfony CLI project
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <http://www.gnu.org/licenses/>.
 */

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
