/*
 * Copyright (c) 2024-present Fabien Potencier <fabien@symfony.com>
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

import (
	"testing"
)

func TestGetConfiguredAndRunning(t *testing.T) {
	proxyProjects := map[string]*ConfiguredProject{
		"~/app1": {
			Scheme: "https",
			Port:   8000,
		},
		"/var/www/app2": {
			Scheme: "http",
			Port:   8001,
		},
	}

	runningProjects := map[string]*ConfiguredProject{
		"~/app1": {
			Scheme: "https",
			Port:   8000,
		},
		"/var/www/app2": {
			Scheme: "http",
			Port:   8001,
		},
		"/var/www/app3": {
			Scheme: "ftp",
			Port:   8002,
		},
	}

	projects, err := GetConfiguredAndRunning(proxyProjects, runningProjects)
	if err != nil {
		t.Errorf("Error was not expected: %v", err)
	}

	if len(projects) != 3 {
		t.Errorf("Expected 2 projects, got %d", len(projects))
	}

	if projects["~/app1"].Port != 8000 {
		t.Errorf("Expected 8000, got %d", projects["~/app1"].Port)
	}

	if projects["~/app1"].Scheme != "https" {
		t.Errorf("Expected \"https\", got %s", projects["~/app1"].Scheme)
	}

	if projects["/var/www/app2"].Port != 8001 {
		t.Errorf("Expected 8001, got %d", projects["/var/www/app2"].Port)
	}

	if projects["/var/www/app2"].Scheme != "http" {
		t.Errorf("Expected \"http\", got %s", projects["/var/www/app2"].Scheme)
	}

	if projects["/var/www/app3"].Port != 8002 {
		t.Errorf("Expected 8002, got %d", projects["/var/www/app3"].Port)
	}

	if projects["/var/www/app3"].Scheme != "ftp" {
		t.Errorf("Expected \"ftp\", got %s", projects["/var/www/app3"].Scheme)
	}
}
