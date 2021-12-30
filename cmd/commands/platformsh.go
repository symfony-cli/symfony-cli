package commands

import (
	"bytes"
	"crypto/md5"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/local/php"
	"github.com/symfony-cli/symfony-cli/util"
	"github.com/symfony-cli/terminal"
)

// Bump whenever we want to be sure we get a recent version of the psh CLI
var internalVersion = []byte("3")

type platformshCLI struct {
	path       string
	definition pshdefinition
}

type pshdefinition struct {
	Namespaces []pshnamespace `json:"namespaces"`
	Commands   []pshcommand   `json:"commands"`
}

type pshnamespace struct {
	ID           string   `json:"id"`
	CommandNames []string `json:"commands"`
}

type pshcommand struct {
	Name        string   `json:"name"`
	Usage       []string `json:"usage"`
	Description string   `json:"description"`
	Help        string   `json:"help"`
	Hidden      bool     `json:"hidden"`
}

func NewPlatformShCLI() (*platformshCLI, error) {
	home, err := homedir.Dir()
	if err != nil {
		return nil, err
	}

	// the Platform.sh CLI is always available on the containers
	// thanks to the configurator
	if !util.InCloud() {
		if err := installPlatformPhar(home); err != nil {
			return nil, console.Exit(err.Error(), 1)
		}
	}
	dir := filepath.Join(home, ".platformsh", "bin")
	p := &platformshCLI{
		path: filepath.Join(dir, "platform"),
	}
	if err := p.parseCommands(home, dir); err != nil {
		return nil, console.Exit(err.Error(), 1)
	}
	return p, nil
}

var nextCmd = &console.Command{
	Category: "self",
	Name:     "use-next",
	Aliases:  []*console.Alias{{Name: "use-next"}},
	Usage:    "Use the next version of SymfonyCloud",
	Hidden:   console.Hide,
	Action: func(c *console.Context) error {
		file, err := os.Create(filepath.Join(util.GetHomeDir(), ".use-next"))
		if err != nil {
			return err
		}
		file.Close()
		home, err := homedir.Dir()
		if err != nil {
			return err
		}
		installPlatformPhar(home)
		terminal.Println("<info>You're now all set to use the next version of SymfonyCloud!</>")
		return nil
	},
}

var platformshInstallCLICmd = &console.Command{
	Category: "self",
	Name:     "install-platform-sh-cli",
	Usage:    "Install Platform.sh CLI (useful in a build hook in a container)",
	Hidden:   console.Hide,
	Action: func(c *console.Context) error {
		// running this empty command will trigger the installation of the platform CLI
		return nil
	},
}

func (p *platformshCLI) parseCommands(home, dir string) error {
	// Cache commands list based on PHAR checksum
	cacheDir := filepath.Join(home, ".platformsh", ".symfony", "cache")
	if _, err := os.Stat(cacheDir); err != nil {
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			return err
		}
	}

	var pharPath = filepath.Join(dir, "platform")
	hasher := md5.New()
	if s, err := ioutil.ReadFile(pharPath); err != nil {
		hasher.Write(s)
	}

	cachePath := filepath.Join(cacheDir, "platform-commands-cache-"+hex.EncodeToString(hasher.Sum(nil)))

	// If cache isn't available for this version, fetch the payload from JSON platform binary
	var commandsJSON []byte
	var err error
	if commandsJSON, err = ioutil.ReadFile(cachePath); err != nil {
		var buf bytes.Buffer
		e := p.executor([]string{"list", "--format=json"})
		e.Dir = dir
		e.Stdout = &buf
		if ret := e.Execute(false); ret != 0 {
			return errors.Errorf("unable to list commands: %s", buf.String())
		}

		commandsJSON = buf.Bytes()
		ioutil.WriteFile(cachePath, commandsJSON, 0600)
	}

	// Fix PHP types
	cleanOutput := bytes.ReplaceAll(commandsJSON, []byte(`"arguments":[]`), []byte(`"arguments":{}`))
	if err := json.Unmarshal(cleanOutput, &p.definition); err != nil {
		return err
	}

	return nil
}

func installPlatformPhar(home string) error {
	cacheDir := filepath.Join(home, ".platformsh", ".symfony", "cache")
	if _, err := os.Stat(cacheDir); err != nil {
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			return err
		}
	}
	var versionPath = filepath.Join(cacheDir, "internal_version")
	dir := filepath.Join(home, ".platformsh", "bin")
	if _, err := os.Stat(filepath.Join(dir, "platform")); err == nil {
		// check "API version" (we never upgrade automatically the psh CLI except if we need to if our code would not be compatible with old versions)
		if v, err := ioutil.ReadFile(versionPath); err == nil && bytes.Equal(v, internalVersion) {
			return nil
		}
	}

	// FIXME: On Windows, there is an installer, how do we do?
	spinner := terminal.NewSpinner(terminal.Stdout)
	spinner.PrefixText = "Download additional CLI tools..."
	spinner.Start()
	defer spinner.Stop()
	resp, err := http.Get("https://platform.sh/cli/installer")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	installer, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	installerPath := filepath.Join(home, "platformsh-installer.php")
	ioutil.WriteFile(installerPath, installer, 0666)
	defer os.Remove(installerPath)

	var stdout bytes.Buffer
	e := &php.Executor{
		Dir:        home,
		BinName:    "php",
		Args:       []string{"php", installerPath},
		SkipNbArgs: 1,
		Stdout:     &stdout,
		Stderr:     &stdout,
	}
	if ret := e.Execute(false); ret == 1 {
		return errors.Errorf("unable to setup platformsh CLI: %s", stdout.String())
	}

	return ioutil.WriteFile(versionPath, internalVersion, 0644)
}

func PSHInitAppFunc(c *console.Context) error {
	return nil
}

func (p *platformshCLI) PSHMainCommands() []*console.Command {
	names := map[string]bool{
		"cloud:auth:browser-login": true,
		"cloud:projects":           true,
		"cloud:environments":       true,
		"cloud:environment:branch": true,
		"cloud:tunnel:open":        true,
		"cloud:environment:ssh":    true,
		"cloud:environment:push":   true,
		"cloud:domains":            true,
		"cloud:variables":          true,
		"cloud:user:add":           true,
	}
	mainCmds := []*console.Command{}
	for _, command := range p.PSHCommands() {
		if names[command.FullName()] {
			mainCmds = append(mainCmds, command)
		}
	}
	return mainCmds
}

func (p *platformshCLI) WrapHelpPrinter() func(w io.Writer, templ string, data interface{}) {
	currentHelpPrinter := console.HelpPrinter
	return func(w io.Writer, templ string, data interface{}) {
		switch cmd := data.(type) {
		case *console.Command:
			if strings.HasPrefix(cmd.Category, "cloud") {
				p.proxyPSHCmdHelp(w, cmd)
			} else {
				currentHelpPrinter(w, templ, data)
			}
		default:
			currentHelpPrinter(w, templ, data)
		}
	}
}

func (p *platformshCLI) PSHCommands() []*console.Command {
	allCommandNames := map[string]bool{}
	for _, n := range p.definition.Namespaces {
		for _, name := range n.CommandNames {
			allCommandNames[name] = true
		}
		// FIXME: missing the aliases here
	}

	commands := []*console.Command{
		platformshInstallCLICmd,
	}
	for _, command := range p.definition.Commands {
		if strings.Contains(command.Description, "deprecated") || strings.Contains(command.Description, "DEPRECATED") {
			continue
		}
		if command.Name == "list" || command.Name == "help" /* || command.Name == "environment:ssh" */ {
			continue
		}
		if strings.HasPrefix(command.Name, "self:") {
			command.Hidden = true
		}
		namespace := "cloud"
	loop:
		for _, n := range p.definition.Namespaces {
			for _, name := range n.CommandNames {
				if name == command.Name {
					if n.ID != "_global" {
						namespace += ":" + n.ID
					}
					break loop
				}
			}
		}
		name := strings.TrimPrefix("cloud:"+command.Name, namespace+":")
		cmd := &console.Command{
			Category: namespace,
			Name:     name,
			Usage:    command.Description,
			Args: []*console.Arg{
				{Name: "anything", Slice: true, Optional: true},
			},
			FlagParsing: console.FlagParsingSkipped,
			Action:      p.proxyPSHCmd(command),
		}
		if command.Hidden {
			cmd.Hidden = console.Hide
		}
		if namespace != "cloud" {
			cmd.Aliases = append(cmd.Aliases, &console.Alias{Name: command.Name, Hidden: true})
		}
		for _, usage := range command.Usage {
			if allCommandNames[usage] {
				cmd.Aliases = append(cmd.Aliases, &console.Alias{Name: "cloud:" + usage}, &console.Alias{Name: usage, Hidden: true})
			}
		}
		if command.Name == "environment:push" {
			cmd.Aliases = append(cmd.Aliases, &console.Alias{Name: "deploy"}, &console.Alias{Name: "cloud:deploy"})
		}
		commands = append(commands, cmd)
	}
	return commands
}

func (p *platformshCLI) proxyPSHCmd(command pshcommand) console.ActionFunc {
	return func(command pshcommand) console.ActionFunc {
		return func(c *console.Context) error {
			args := os.Args[1:]
			for i := range args {
				args[i] = strings.Replace(args[i], c.Command.UserName, command.Name, 1)
			}
			e := p.executor(args)
			return console.Exit("", e.Execute(false))
		}
	}(command)
}

func (p *platformshCLI) proxyPSHCmdHelp(w io.Writer, command *console.Command) {
	e := p.executor([]string{strings.TrimPrefix(command.FullName(), "cloud:"), "--help", "--ansi"})
	e.Execute(false)
}

func (p *platformshCLI) executor(args []string) *php.Executor {
	e := &php.Executor{
		BinName: "php",
		Args:    append([]string{"php", p.path}, args...),
		ExtraEnv: []string{
			"PLATFORMSH_CLI_APPLICATION_NAME=Platform.sh CLI for Symfony",
			"PLATFORMSH_CLI_APPLICATION_EXECUTABLE=symfony cloud:",
		},
	}
	e.Paths = append([]string{filepath.Dir(p.path)}, e.Paths...)
	return e
}
