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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	compose "github.com/compose-spec/compose-go/cli"
	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/phpstore"
	"github.com/symfony-cli/symfony-cli/book"
	"github.com/symfony-cli/symfony-cli/git"
	"github.com/symfony-cli/symfony-cli/local/php"
	"github.com/symfony-cli/symfony-cli/local/upsun"
	"github.com/symfony-cli/symfony-cli/util"
	"github.com/symfony-cli/terminal"
)

type CloudService struct {
	Name    string
	Type    string
	Version string
}

var localNewCmd = &console.Command{
	Category: "local",
	Name:     "new",
	Aliases:  []*console.Alias{{Name: "new"}},
	Usage:    "Create a new Symfony project",
	Flags: []console.Flag{
		dirFlag,
		&console.StringFlag{
			Name:  "version",
			Usage: `The version of the Symfony skeleton (a version or one of "lts", "stable", "next", or "previous")`,
		},
		&console.BoolFlag{Name: "full", Usage: "Use github.com/symfony/website-skeleton (deprecated, use --webapp instead)"},
		&console.BoolFlag{Name: "demo", Usage: "Use github.com/symfony/demo"},
		&console.BoolFlag{Name: "webapp", Usage: "Add the webapp pack to get a fully configured web project"},
		&console.BoolFlag{Name: "api", Usage: "Add the api pack to get a fully configured api project"},
		&console.BoolFlag{Name: "book", Usage: "Clone the Symfony: The Fast Track book project"},
		&console.BoolFlag{Name: "docker", Usage: "Enable Docker support"},
		&console.BoolFlag{Name: "no-git", Usage: "Do not initialize Git"},
		&console.BoolFlag{Name: "upsun", Usage: "Initialize Upsun configuration"},
		&console.BoolFlag{Name: "cloud", Usage: "Initialize Platform.sh configuration"},
		&console.StringSliceFlag{Name: "service", Usage: "Configure some services", Hidden: true},
		&console.BoolFlag{Name: "debug", Usage: "Display commands output"},
		&console.StringFlag{Name: "php", Usage: "PHP version to use"},
	},
	Args: console.ArgDefinition{
		{Name: "directory", Optional: true, Description: "Directory of the project to create"},
	},
	Before: func(c *console.Context) error {
		if !c.Bool("no-git") {
			return CheckGitIsAvailable(c)
		}

		return nil
	},
	Action: func(c *console.Context) error {
		dir := c.Args().Get("directory")
		if c.String("dir") != "" {
			dir = c.String("dir")
		}
		if dir == "" {
			return console.Exit("A directory must be passed as an argument or via the --dir option", 1)
		}
		if _, err := os.Stat(dir); err == nil {
			// directory exists, but is it empty?
			empty, err := isEmpty(dir)
			if err != nil {
				return errors.Wrapf(err, "Unable to access project directory %s", dir)
			}
			if !empty {
				return console.Exit(fmt.Sprintf("Project directory %s is not empty", dir), 1)
			}
		}
		if !filepath.IsAbs(dir) {
			var err error
			if dir, err = filepath.Abs(dir); err != nil {
				return errors.Wrapf(err, "Project directory %s is not accessible", dir)
			}
		}

		symfonyVersion := c.String("version")

		if c.Bool("book") {
			if symfonyVersion == "" {
				versions, err := book.Versions()
				if err != nil {
					return errors.Wrap(err, "unable to get book versions")
				}
				terminal.Println("The --version flag is required for the Symfony book; available versions:")
				for _, v := range versions {
					terminal.Println(fmt.Sprintf(" - %s", v))
				}
				return console.Exit("", 1)
			}

			book := &book.Book{
				Dir:         dir,
				Debug:       c.Bool("debug"),
				Force:       false,
				AutoConfirm: true,
			}

			return book.Clone(symfonyVersion)
		}

		if symfonyVersion != "" && c.Bool("demo") {
			return console.Exit("The --version flag is not supported for the Symfony Demo", 1)
		}
		if c.Bool("webapp") && c.Bool("api") {
			return console.Exit("The --api flag cannot be used with --webapp", 1)
		}
		withCloud := c.Bool("cloud") || c.Bool("upsun")
		if len(c.StringSlice("service")) > 0 && !withCloud {
			return console.Exit("The --service flag cannot be used without --cloud or --upsun", 1)
		}
		if withCloud && c.Bool("no-git") {
			return console.Exit("The --no-git flag cannot be used with --cloud or --upsun", 1)
		}

		s := terminal.NewSpinner(terminal.Stderr)
		s.Start()
		defer s.Stop()

		minorPHPVersion, err := forcePHPVersion(c.String("php"), dir)
		if err != nil {
			return err
		}

		if err := createProjectWithComposer(c, dir, symfonyVersion); err != nil {
			return err
		}

		if c.String("php") != "" && !withCloud {
			if err := createPhpVersionFile(c.String("php"), dir); err != nil {
				return err
			}
		}

		if !c.Bool("no-git") {
			if err := initProjectGit(c, dir); err != nil {
				return err
			}
		}

		if c.Bool("webapp") {
			if err := runComposer(c, dir, []string{"require", "webapp"}, c.Bool("debug")); err != nil {
				return err
			} else if !c.Bool("no-git") {
				buf, err := git.AddAndCommit(dir, []string{"."}, "Add webapp packages", c.Bool("debug"))
				if err != nil {
					fmt.Print(buf.String())
					return err
				}
			}
		}

		if c.Bool("api") {
			if err := runComposer(c, dir, []string{"require", "api"}, c.Bool("debug")); err != nil {
				return err
			}
		}

		if withCloud {
			if err := runComposer(c, dir, []string{"require", "platformsh"}, c.Bool("debug")); err != nil {
				return err
			}
			if !c.Bool("no-git") {
				buf, err := git.AddAndCommit(dir, []string{"."}, "Add more packages", c.Bool("debug"))
				if err != nil {
					fmt.Print(buf.String())
					return err
				}
			}
			brand := upsun.PlatformshBrand
			if c.Bool("upsun") {
				brand = upsun.UpsunBrand
			}
			if err := initCloud(c, brand, minorPHPVersion, dir); err != nil {
				return err
			}
		}

		adir, err := filepath.Abs(dir)
		if err != nil {
			adir = dir
		}
		ui := terminal.SymfonyStyle(terminal.Stdout, terminal.Stdin)
		ui.Success(fmt.Sprintf("Your project is now ready in %s", adir))
		return nil
	},
}

func isEmpty(dir string) (bool, error) {
	f, err := os.Open(dir)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

func initCloud(c *console.Context, brand upsun.CloudBrand, minorPHPVersion, dir string) error {
	terminal.Printfln("* Adding %s configuration", brand)

	cloudServices, err := parseCloudServices(dir, c.StringSlice("service"))
	if err != nil {
		return err
	}

	// FIXME: display or hide output based on debug flag
	_, err = createRequiredFilesProject(brand, dir, "app", "", minorPHPVersion, cloudServices, c.Bool("dump"), c.Bool("force"))
	if err != nil {
		return err
	}

	if !c.Bool("no-git") {
		return nil
	}
	buf, err := git.AddAndCommit(dir, []string{"."}, fmt.Sprintf("Add %s configuration", brand), c.Bool("debug"))
	if err != nil {
		fmt.Print(buf.String())
	}
	return err
}

func parseCloudServices(dir string, services []string) ([]*CloudService, error) {
	// from CLI flag
	cloudServices, err := parseCLIServices(services)
	if err != nil {
		return nil, err
	}

	// from Docker Compose configuration
	cloudServices = append(cloudServices, parseDockerComposeServices(dir)...)

	return cloudServices, nil
}

func parseCLIServices(services []string) ([]*CloudService, error) {
	var cloudServices []*CloudService

	for _, config := range services {
		// up to 3 parts -> database:postgresql:13
		var service *CloudService
		parts := strings.Split(config, ":")
		if len(parts) == 1 {
			// service == name
			service = &CloudService{Name: parts[0], Type: parts[0], Version: upsun.ServiceLastVersion(parts[1])}
		} else if len(parts) == 2 {
			service = &CloudService{Name: parts[0], Type: parts[1], Version: upsun.ServiceLastVersion(parts[1])}
		} else if len(parts) == 3 {
			service = &CloudService{Name: parts[0], Type: parts[1], Version: parts[2]}
		} else {
			return nil, errors.Errorf("unable to parse service \"%s\"", config)
		}
		cloudServices = append(cloudServices, service)
	}
	return cloudServices, nil
}

func parseDockerComposeServices(dir string) []*CloudService {
	var cloudServices []*CloudService

	options, err := compose.NewProjectOptions(nil, compose.WithWorkingDirectory(dir), compose.WithDefaultConfigPath, compose.WithConfigFileEnv, compose.WithEnv(os.Environ()))
	if err != nil {
		return nil
	}
	project, err := compose.ProjectFromOptions(options)
	if err != nil {
		return nil
	}

	seen := map[string]bool{}
	for _, service := range project.Services {
		for _, port := range service.Ports {
			var s *CloudService
			switch port.Target {
			case 3306:
				s = &CloudService{Type: "mysql"}
			case 5432:
				s = &CloudService{Type: "postgresql"}
			case 6379:
				s = &CloudService{Type: "redis"}
			case 11211:
				s = &CloudService{Type: "memcached"}
			case 5672:
				s = &CloudService{Type: "rabbitmq"}
			case 9200:
				s = &CloudService{Type: "elasticsearch"}
			case 27017:
				s = &CloudService{Type: "mongodb"}
			case 9092:
				s = &CloudService{Type: "kafka"}
			}
			_, done := seen[service.Name]
			if s != nil && !done {
				seen[service.Name] = true
				s.Name = service.Name
				parts := strings.Split(service.Image, ":")
				s.Version = regexp.MustCompile(`\d+(\.\d+)?`).FindString(parts[len(parts)-1])
				serviceLastVersion := upsun.ServiceLastVersion(s.Type)
				if s.Version == "" {
					s.Version = serviceLastVersion
				} else if s.Version > serviceLastVersion {
					terminal.Printf("Unsupported %s version %s using version %s\n", s.Type, s.Version, serviceLastVersion)
					s.Version = serviceLastVersion
				}
				cloudServices = append(cloudServices, s)
			}
		}
	}
	return cloudServices
}

func initProjectGit(c *console.Context, dir string) error {
	terminal.Println("* Setting up the project under Git version control")
	terminal.Printfln("  (running git init %s)\n", dir)
	// Only force the branch to be "main" when running a Cloud context to make
	// onboarding simpler.
	if buf, err := git.Init(dir, c.Bool("cloud") || c.Bool("upsun"), c.Bool("debug")); err != nil {
		fmt.Print(buf.String())
		return err
	}
	buf, err := git.AddAndCommit(dir, []string{"."}, "Add initial set of files", c.Bool("debug"))
	if err != nil {
		fmt.Print(buf.String())
	}
	return err
}

func createProjectWithComposer(c *console.Context, dir, version string) error {
	if c.Bool("demo") {
		terminal.Println("* Creating a new Symfony Demo project with Composer")
	} else if version != "" {
		if version == "lts" || version == "previous" || version == "stable" || version == "next" || version == "dev" {
			var err error
			version, err = getSpecialVersion(version)
			if err != nil {
				return err
			}
		}

		terminal.Printfln("* Creating a new Symfony %s project with Composer", version)
	} else {
		terminal.Println("* Creating a new Symfony project with Composer")
	}

	repo := "symfony/skeleton"
	if r := os.Getenv("SYMFONY_REPO"); r != "" {
		repo = r
	} else if c.Bool("full") {
		terminal.SymfonyStyle(terminal.Stdout, terminal.Stdin).Warning("The --full flag is deprecated, use --webapp instead.")
		repo = "symfony/website-skeleton"
	} else if c.Bool("demo") {
		repo = "symfony/symfony-demo"
	}

	if ok, _ := regexp.MatchString("^\\d+\\.\\d+$", version); ok {
		version = version + ".*"
	}

	return runComposer(c, "", []string{"create-project", repo, dir, version}, c.Bool("debug"))
}

func runComposer(c *console.Context, dir string, args []string, debug bool) error {
	var (
		buf bytes.Buffer
		out io.Writer = &buf
		err io.Writer = &buf
	)
	if debug {
		out = os.Stdout
		err = os.Stderr
	} else {
		args = append(args, "--no-interaction")
	}
	env := []string{}
	if c.Bool("docker") {
		env = append(env, "SYMFONY_DOCKER=1")
	}

	if err := php.Composer(dir, args, env, out, err, os.Stderr, terminal.Logger); err.ExitCode() != 0 {
		terminal.Println(buf.String())
		return err
	}
	return nil
}

func getSpecialVersion(version string) (string, error) {
	resp, err := http.Get("https://symfony.com/all-versions.json")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var versions map[string]interface{}
	if err := json.Unmarshal(body, &versions); err != nil {
		return "", err
	}

	v := versions[version].(string)
	if version == "next" || version == "dev" {
		v += ".x@dev"
	}

	return v, nil
}

func forcePHPVersion(v, dir string) (string, error) {
	store := phpstore.New(util.GetHomeDir(), true, nil)
	if v == "" {
		minor, _, _, err := store.BestVersionForDir(dir)
		if err != nil {
			return "", err
		}
		return strings.Join(strings.Split(minor.Version, ".")[0:2], "."), nil
	}
	if _, err := version.NewVersion(v); err != nil {
		return "", errors.Errorf("unable to parse PHP version \"%s\"", v)
	}
	// check that the version is available
	if !store.IsVersionAvailable(v) {
		return "", errors.Errorf("PHP version \"%s\" is not available locally", v)
	}
	os.Setenv("FORCED_PHP_VERSION", v)
	return strings.Join(strings.Split(v, ".")[0:2], "."), nil
}

func createPhpVersionFile(v, dir string) error {
	file := filepath.Join(dir, ".php-version")
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return errors.Wrapf(err, "unable to create %s", file)
	}
	if _, err = f.WriteString(v + "\n"); err != nil {
		f.Close()
		return errors.Wrapf(err, "unable to write %s", file)
	}
	if err = f.Close(); err != nil {
		return errors.Wrapf(err, "unable to close %s", file)
	}
	return nil
}
