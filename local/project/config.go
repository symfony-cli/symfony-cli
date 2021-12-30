package project

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/symfony-cli/console"
	"gopkg.in/yaml.v2"
)

// Config is the struct taken by New (should not be used for anything else)
type Config struct {
	HomeDir      string
	ProjectDir   string
	DocumentRoot string `yaml:"document_root"`
	Passthru     string `yaml:"passthru"`
	PreferedPort int    `yaml:"prefered_port"`
	PKCS12       string `yaml:"p12"`
	Logger       zerolog.Logger
	AppVersion   string
	AllowHTTP    bool `yaml:"allow_http"`
	NoTLS        bool `yaml:"no_tls"`
	Daemon       bool `yaml:"daemon"`
}

type FileConfig struct {
	Proxy struct {
		Domains []string `yaml:"domains"`
	} `yaml:"proxy"`
	HTTP    *Config            `yaml:"http"`
	Workers map[string]*Worker `yaml:"workers"`
}

type Worker struct {
	Cmd   []string `yaml:"cmd"`
	Watch []string `yaml:"watch"`
}

func NewConfigFromContext(c *console.Context, projectDir string) (*Config, *FileConfig, error) {
	config := &Config{}
	var fileConfig *FileConfig
	var err error
	fileConfig, err = newConfigFromFile(filepath.Join(projectDir, ".symfony.local.yaml"))
	if err != nil {
		return nil, nil, err
	}
	if fileConfig != nil {
		if fileConfig.HTTP == nil {
			fileConfig.HTTP = &Config{}
		} else {
			config = fileConfig.HTTP
		}
		if fileConfig.Workers == nil {
			fileConfig.Workers = make(map[string]*Worker)
		}
	}
	config.AppVersion = c.App.Version
	config.ProjectDir = projectDir
	if c.IsSet("document-root") {
		config.DocumentRoot = c.String("document-root")
	}
	if c.IsSet("passthru") {
		config.Passthru = c.String("passthru")
	}
	if c.IsSet("port") {
		config.PreferedPort = c.Int("port")
	}
	if config.PreferedPort == 0 {
		config.PreferedPort = 8000
	}
	if c.IsSet("allow-http") {
		config.AllowHTTP = c.Bool("allow-http")
	}
	if c.IsSet("p12") {
		config.PKCS12 = c.String("p12")
	}
	if c.IsSet("no-tls") {
		config.NoTLS = c.Bool("no-tls")
	}
	if c.IsSet("daemon") {
		config.Daemon = c.Bool("daemon")
	}
	return config, fileConfig, nil
}

// Should only be used when for customers
func newConfigFromFile(configFile string) (*FileConfig, error) {
	if _, err := os.Stat(configFile); err != nil {
		return nil, nil
	}

	contents, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var fileConfig FileConfig
	if err := yaml.Unmarshal(contents, &fileConfig); err != nil {
		return nil, err
	}

	if err := fileConfig.parseWorkers(); err != nil {
		return nil, err
	}

	return &fileConfig, nil
}

func (c *FileConfig) parseWorkers() error {
	if c.Workers == nil {
		c.Workers = make(map[string]*Worker)
		return nil
	}

	if v, ok := c.Workers["yarn_encore_watch"]; ok && v == nil {
		c.Workers["yarn_encore_watch"] = &Worker{
			Cmd: []string{"yarn", "encore", "dev", "--watch"},
		}
	}
	if v, ok := c.Workers["messenger_consume_async"]; ok && v == nil {
		c.Workers["messenger_consume_async"] = &Worker{
			Cmd:   []string{"symfony", "console", "messenger:consume", "async"},
			Watch: []string{"config", "src", "templates", "vendor"},
		}
	}

	for k, v := range c.Workers {
		if v == nil {
			return errors.Errorf("The \"%s\" worker entry in \".symfony.local.yaml\" cannot be empty.", k)
		}
	}

	return nil
}
