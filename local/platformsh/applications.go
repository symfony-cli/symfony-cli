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

package platformsh

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/symfony-cli/terminal"
	yaml "gopkg.in/yaml.v2"
)

var skippedDirectories = map[string]interface{}{
	".git":         nil,
	"vendor":       nil,
	"node_modules": nil,
	"bundles":      nil,
	"cache":        nil,
	"config":       nil,
	"public":       nil,
	"tests":        nil,
	"templates":    nil,
	"assets":       nil,
	"images":       nil,
	"fonts":        nil,
	"js":           nil,
	"src":          nil,
	"var":          nil,
	"web":          nil,
}

type UpsunDotYaml struct {
	Applications map[string]struct {
		LocalApplication `yaml:",inline"`
		Source           struct {
			Root string
		}
	}
}

// Only a wrapper type around LocalApplication used to get Access to
// `source.root` when unmarshalling
type ApplicationsDotYaml []struct {
	LocalApplication `yaml:",inline"`
	Source           struct {
		Root string
	}
}

type LocalWorker struct {
}

type LocalApplication struct {
	DefinitionFile string                 `yaml:"-"`
	LocalRootDir   string                 `yaml:"-"`
	Name           string                 `yaml:"name"`
	Type           string                 `yaml:"type"`
	Workers        map[string]LocalWorker `yaml:"workers"`
}

// ApplicationInterface interface
func (p LocalApplication) GetName() string {
	return p.Name
}

type LocalApplications []LocalApplication

// LocalApplications attaches the methods of Interface to []LocalApplication, sorting in increasing order.
func (p LocalApplications) Len() int           { return len(p) }
func (p LocalApplications) Less(i, j int) bool { return p[i].GetName() < p[j].GetName() }
func (p LocalApplications) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// Sort is a convenience method.
func (p LocalApplications) Sort() { sort.Sort(p) }

func FindLocalApplications(rootDirectory string) LocalApplications {
	apps := LocalApplications{}
	appParser := make(chan string)
	appParsingDone := make(chan bool)

	rootDirectory, err := filepath.EvalSymlinks(rootDirectory)
	if err != nil {
		terminal.Logger.Error().Msgf("Could not eval project root directory: %s\n", err)
		return apps
	}

	brand := GuessCloudFromDirectory(rootDirectory)
	if brand == NoBrand {
		return apps
	}
	go func() {
		for file := range appParser {
			content, err := os.ReadFile(file)
			if err != nil {
				terminal.Logger.Warn().Msgf("Could not read %s file: %s\n", file, err)
				continue
			}

			if brand == UpsunBrand {
				var config UpsunDotYaml
				if err := yaml.Unmarshal(content, &config); err != nil {
					terminal.Logger.Error().Msgf("Could not decode %s YAML file: %s\n", file, err)
					continue
				}
				for name, app := range config.Applications {
					app.Name = name
					app.DefinitionFile = file
					app.LocalRootDir = filepath.Join(rootDirectory, app.Source.Root)
					apps = append(apps, app.LocalApplication)
				}
				continue
			}

			if strings.HasSuffix(file, filepath.Join(".platform", "applications.yaml")) {
				var multiApps ApplicationsDotYaml
				if err := yaml.Unmarshal(content, &multiApps); err != nil {
					terminal.Logger.Error().Msgf("Could not decode %s YAML file: %s\n", file, err)
					continue
				}
				for _, app := range multiApps {
					app.DefinitionFile = file
					app.LocalRootDir = filepath.Join(rootDirectory, app.Source.Root)
					apps = append(apps, app.LocalApplication)
				}
				continue
			}

			app := LocalApplication{
				DefinitionFile: file,
				LocalRootDir:   filepath.Dir(file),
			}
			if err := yaml.Unmarshal(content, &app); err != nil {
				terminal.Logger.Error().Msgf("Could not decode %s YAML file: %s\n", file, err)
				continue
			}
			apps = append(apps, app)
		}
		appParsingDone <- true
	}()

	for _, path := range findAppConfigFiles(brand, rootDirectory) {
		appParser <- path
	}

	close(appParser)
	<-appParsingDone
	apps.Sort()

	return apps
}

func findAppConfigFiles(brand CloudBrand, dir string) []string {
	files := []string{}
	if brand == UpsunBrand {
		dir = filepath.Join(dir, brand.ProjectConfigPath)
		fs, err := os.ReadDir(dir)
		if err != nil {
			return files
		}
		for _, f := range fs {
			if strings.HasSuffix(f.Name(), ".yaml") {
				files = append(files, filepath.Join(dir, f.Name()))
			}
		}
		return files
	}

	separator := string(filepath.Separator)
	rootDirectoryLen := len(dir) + 1
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// prevent panic by handling failure accessing a path
			return nil
		}
		if info.IsDir() {
			// don't go up
			if len(path) < rootDirectoryLen {
				return nil
			}

			// skip known big or useless directory
			if _, skip := skippedDirectories[info.Name()]; skip {
				return filepath.SkipDir
			}

			// don't go too deep down the tree
			if len(strings.Split(path[rootDirectoryLen:], separator)) > 3 {
				return filepath.SkipDir
			}
		}

		if info.Name() == "applications.yaml" || info.Name() == ".platform.app.yaml" {
			files = append(files, path)
		}

		return nil
	})
	return files
}

func GuessSelectedAppByWd(apps LocalApplications) *LocalApplication {
	wd, err := os.Getwd()
	if err != nil || wd == "" {
		return nil
	}

	return GuessSelectedAppByDirectory(wd, apps)
}

func GuessSelectedAppByDirectory(directory string, apps LocalApplications) *LocalApplication {
	if len(apps) == 1 {
		return &apps[0]
	}
	directory, _ = filepath.EvalSymlinks(directory)
	for _, app := range apps {
		if rel, err := filepath.Rel(app.LocalRootDir, directory); err != nil {
			continue
		} else if strings.HasPrefix(rel, "..") {
			continue
		}
		return &app
	}
	return nil
}
