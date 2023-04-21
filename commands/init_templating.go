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

package commands

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"github.com/symfony-cli/symfony-cli/git"
	"github.com/symfony-cli/symfony-cli/local/platformsh"
	"github.com/symfony-cli/symfony-cli/util"
	"github.com/symfony-cli/terminal"
	"gopkg.in/yaml.v2"
)

const templatesGitRepository = "https://github.com/symfonycorp/cloud-templates.git"

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error { return nil }

func createRequiredFilesProject(rootDirectory, projectSlug, templateName string, minorPHPVersion string, cloudServices []*CloudService, dump, force bool) ([]string, error) {
	createdFiles := []string{}
	templates, err := getTemplates(rootDirectory, templateName, minorPHPVersion)
	if err != nil {
		return nil, errors.Wrap(err, "could not determine template to use")
	}
	publicDirectory := "public"

	if fi, err := os.Stat(filepath.Join(rootDirectory, "web")); err == nil && fi.IsDir() {
		publicDirectory = "web"
	}
	frontController := "index.php"
	// Symfony installations can also have a front controller named app.php!
	if _, err := os.Stat(filepath.Join(rootDirectory, publicDirectory, "app.php")); err == nil {
		frontController = "app.php"
	}
	availablePHPExtensions := map[string][]string{
		"postgresql": {"pdo_pgsql"},
		"redis":      {"redis"},
		"rabbitmq":   {"amqp"},
	}
	phpExts := append(phpExtensions(rootDirectory), "apcu", "mbstring", "sodium", "xsl", "blackfire")
	for _, service := range cloudServices {
		if v, ok := availablePHPExtensions[service.Type]; ok {
			phpExts = append(phpExts, v...)
		}
	}
	sort.Strings(phpExts)
	serviceDiskSizes := map[string]string{
		"postgresql": "1024",
	}

	data := &struct {
		Slug             string
		FrontController  string
		PublicDirectory  string
		PhpVersion       string
		PHPExtensions    []string
		Services         []*CloudService
		ServiceDiskSizes map[string]string
	}{
		Slug:             projectSlug,
		FrontController:  frontController,
		PublicDirectory:  publicDirectory,
		PhpVersion:       minorPHPVersion,
		PHPExtensions:    phpExts,
		Services:         cloudServices,
		ServiceDiskSizes: serviceDiskSizes,
	}

	for file, templateText := range templates {
		file = filepath.Join(rootDirectory, file)
		var f io.WriteCloser
		if dump {
			f = nopCloser{terminal.Stdout}
			fmt.Fprintf(f, "\n<info># %s:</>\n", file)
		} else if _, err := os.Stat(file); !force && (err == nil || !os.IsNotExist(err)) {
			terminal.Logger.Warn().Msgf("%s already exists, template generation skipped\n", file)
			continue
		} else {
			createdFiles = append(createdFiles, file)

			if dir := filepath.Dir(file); dir != "" {
				if err := os.MkdirAll(dir, 0755); err != nil {
					return createdFiles, errors.WithStack(errors.Wrapf(err, "unable to create directory %s", dir))
				}
			}
			f, err = os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
			if err != nil {
				return createdFiles, errors.WithStack(errors.Wrapf(err, "unable to create %s", file))
			}
		}

		if err = templateText.Execute(f, data); err != nil {
			return createdFiles, errors.WithStack(errors.Wrapf(err, "unable to write to %s", file))
		}
		if err = f.Close(); err != nil {
			return createdFiles, errors.WithStack(errors.Wrapf(err, "unable to close %s", file))
		}
	}

	return createdFiles, nil
}

func isValidURL(toTest string) bool {
	u, err := url.Parse(toTest)
	if err != nil {
		return false
	}

	if u.Host == "" {
		return false
	}

	return true
}

func isValidFilePath(toTest string) bool {
	f, err := os.Stat(toTest)

	if err != nil {
		return false
	}

	if f.IsDir() {
		return false
	}

	return true
}

func getTemplates(rootDirectory, chosenTemplateName string, minorPHPVersion string) (map[string]*template.Template, error) {
	var foundTemplate *configTemplate

	s := terminal.NewSpinner(terminal.Stderr)
	s.Start()
	defer func() {
		s.Stop()
	}()
	terminal.Println("Updating configuration templates")
	directory := filepath.Join(util.GetHomeDir(), "cache", "templates")
	if f, err := os.Stat(directory); err == nil && f.IsDir() {
		terminal.Logger.Info().Msg("Updating configuration templates cache")
		if err := git.Fetch(directory, templatesGitRepository, "master"); err != nil {
			return nil, errors.Wrap(err, "could not update configuration templates")
		}
		if err := git.ResetHard(directory, "FETCH_HEAD"); err != nil {
			return nil, errors.Wrap(err, "could not update configuration templates")
		}
	} else {
		if err := os.MkdirAll(filepath.Dir(directory), 0755); err != nil {
			return nil, errors.Wrapf(err, "unable to create directory for %s", directory)
		}
		terminal.Logger.Info().Msg("Initial configuration templates fetch")
		if err := git.Clone(templatesGitRepository, directory); err != nil {
			return nil, errors.Wrap(err, "could not fetch configuration templates")
		}
	}

	if isURL, isFile := isValidURL(chosenTemplateName), isValidFilePath(chosenTemplateName); isURL || isFile {
		var (
			templateConfigBytes []byte
			err                 error
		)

		if isFile {
			templateConfigBytes, err = os.ReadFile(chosenTemplateName)
		} else {
			var resp *http.Response
			resp, err = http.Get(chosenTemplateName)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			if resp.StatusCode >= 400 {
				return nil, errors.Errorf("Got HTTP status code >= 400: %s", resp.Status)
			}
			defer resp.Body.Close()

			templateConfigBytes, err = io.ReadAll(resp.Body)
		}

		if err != nil {
			return nil, errors.Wrap(err, "could not apply project template")
		}

		if err := yaml.Unmarshal(templateConfigBytes, &foundTemplate); err != nil {
			return nil, errors.Wrap(err, "could not apply project template")
		}

		terminal.Logger.Info().Msg("Using template " + chosenTemplateName)
	} else {
		files, err := os.ReadDir(directory)
		if err != nil {
			return nil, errors.Wrap(err, "could not read configuration templates")
		}

		for _, file := range files {
			if file.Name() == ".git" {
				continue
			}

			if file.IsDir() {
				terminal.Logger.Warn().Msg(file.Name() + " is not a regular file")
				continue
			}

			templateName := strings.TrimSuffix(file.Name(), ".yaml")[strings.Index(file.Name(), "-")+1:]
			isTemplateChosen := chosenTemplateName == templateName
			if chosenTemplateName != "" && !isTemplateChosen {
				continue
			}

			templateConfigBytes, err := os.ReadFile(filepath.Join(directory, file.Name()))
			if err != nil {
				if isTemplateChosen {
					return nil, errors.Wrap(err, "could not apply configuration template")
				}

				terminal.Logger.Warn().Msg(err.Error())
				continue
			}

			var templateConfig configTemplate

			if err := yaml.Unmarshal(templateConfigBytes, &templateConfig); err != nil {
				if isTemplateChosen {
					return nil, errors.Wrap(err, "could not apply configuration template")
				}

				terminal.Logger.Warn().Msg(err.Error())
				continue
			}

			if chosenTemplateName == "" && !templateConfig.Match(rootDirectory, minorPHPVersion) {
				continue
			}

			terminal.Printf("Using configuration template <info>%s</>\n", templateName)
			foundTemplate = &templateConfig
			break
		}
	}

	if foundTemplate == nil {
		return nil, errors.New("no matching template found")
	}

	phpini, err := os.ReadFile(filepath.Join(directory, "php.ini"))
	if err != nil {
		return nil, errors.New("unable to find the php.ini template")
	}
	servicesyaml := `
{{ range $service := $.Services -}}
{{ $service.Name }}:
    type: {{ $service.Type }}{{ if $service.Version }}:{{ $service.Version }}{{ end }}
{{- if index $.ServiceDiskSizes $service.Type }}
    disk: {{ index $.ServiceDiskSizes $service.Type }}
{{ end -}}

{{ end -}}
`

	templateFuncs := getTemplateFuncs(rootDirectory, minorPHPVersion)

	templates := map[string]*template.Template{
		".platform.app.yaml":      template.Must(template.New("output").Funcs(templateFuncs).Parse(foundTemplate.Template)),
		".platform/services.yaml": template.Must(template.New("output").Funcs(templateFuncs).Parse(servicesyaml)),
		".platform/routes.yaml": template.Must(template.New("output").Funcs(templateFuncs).Parse(`"https://{all}/": { type: upstream, upstream: "{{.Slug}}:http" }
"http://{all}/": { type: redirect, to: "https://{all}/" }
`)),
		"php.ini": template.Must(template.New("output").Funcs(templateFuncs).Parse(string(phpini))),
	}

	for path, tpl := range foundTemplate.ExtraFiles {
		if _, alreadyExists := templates[path]; alreadyExists {
			terminal.Logger.Info().Msgf("Skipping extra file<info>%s</>\n", path)
			continue
		}

		templates[path] = template.Must(template.New("output").Funcs(templateFuncs).Parse(tpl))
	}

	return templates, nil
}

type configTemplate struct {
	Requirements []configRequirement
	ExtraFiles   map[string]string `yaml:"extra_files"`
	Template     string
}

func (c *configTemplate) Match(directory string, minorPHPVersion string) bool {
	for _, req := range c.Requirements {
		if !req.Check(directory, minorPHPVersion) {
			return false
		}
	}

	return true
}

type configRequirement struct {
	Type, Value string
}

func (req configRequirement) Check(directory, minorPHPVersion string) bool {
	if f, ok := getTemplateFuncs(directory, minorPHPVersion)[req.Type].(func(string) bool); ok {
		return f(req.Value)
	}

	terminal.Logger.Error().Msg("unsupported check " + req.Type)
	return false
}

func getTemplateFuncs(rootDirectory, minorPHPVersion string) template.FuncMap {
	return template.FuncMap{
		"file_exists": func(file string) bool {
			_, err := os.Stat(filepath.Join(rootDirectory, file))

			return err == nil
		},
		"has_composer_package": func(pkg string) bool {
			return hasComposerPackage(rootDirectory, pkg)
		},
		"has_php_extension": func(ext string) bool {
			return hasPHPExtension(rootDirectory, ext)
		},
		"php_extensions": func() []string {
			// FIXME: obsolete, replaced by PHPExtensions, should be removed
			return phpExtensions(rootDirectory)
		},
		"php_extension_available": platformsh.IsPhpExtensionAvailable,
		"php_at_least": func(v string) bool {
			minVersion, err := version.NewVersion(v)
			if err != nil {
				panic(err)
			}
			minorPHP, err := version.NewVersion(minorPHPVersion)
			if err != nil {
				panic(err)
			}
			return minorPHP.GreaterThanOrEqual(minVersion)
		},
	}
}
