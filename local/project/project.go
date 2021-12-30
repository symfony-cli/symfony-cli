package project

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	lhttp "github.com/symfony-cli/symfony-cli/local/http"
	"github.com/symfony-cli/symfony-cli/local/php"
)

// Project represents a PHP project
type Project struct {
	HTTP       *lhttp.Server
	PHPServer  *php.Server
	Logger     zerolog.Logger
	homeDir    string
	projectDir string
}

// New creates a new PHP project
func New(c *Config) (*Project, error) {
	documentRoot, err := realDocumentRoot(c.ProjectDir, c.DocumentRoot)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	passthru, err := realPassthru(documentRoot, c.Passthru)
	p := &Project{
		Logger:     c.Logger.With().Str("source", "HTTP").Logger(),
		homeDir:    c.HomeDir,
		projectDir: c.ProjectDir,
		HTTP: &lhttp.Server{
			DocumentRoot: documentRoot,
			PreferedPort: c.PreferedPort,
			Logger:       c.Logger,
			PKCS12:       c.PKCS12,
			AllowHTTP:    c.AllowHTTP,
			Appversion:   c.AppVersion,
		},
	}
	if err != nil {
		msg := "unable to detect the front controller"
		if passthru != "index.html" {
			msg += ", disabling the PHP server"
		}
		p.Logger.Warn().Err(err).Msg(msg)
	} else if c.Passthru == "index.html" {
		p.HTTP.Callback = func(w http.ResponseWriter, r *http.Request, env map[string]string) error {
			http.ServeFile(w, r, "/index.html")
			return nil
		}
	} else {
		p.PHPServer, err = php.NewServer(c.HomeDir, c.ProjectDir, documentRoot, passthru, c.Logger)
		if err != nil {
			return nil, err
		}
		p.HTTP.Callback = p.PHPServer.Serve
	}
	return p, nil
}

// realDocumentRoot returns the absolute document root
func realDocumentRoot(dir, documentRoot string) (string, error) {
	if documentRoot == "" {
		documentRoot = guessDocumentRoot(dir)
	} else if !filepath.IsAbs(documentRoot) {
		documentRoot = filepath.Join(dir, documentRoot)
	}
	return strings.TrimRight(documentRoot, string(os.PathSeparator)) + string(os.PathSeparator), nil
}

// realPassthru returns the cached passthru
// or try to guess a new one if not configured
func realPassthru(documentRoot, passthru string) (string, error) {
	if passthru == "" {
		passthru = guessPassthru(documentRoot)
	}
	passthru = "/" + strings.Trim(passthru, "/")
	controller := filepath.Join(documentRoot, passthru)
	if _, err := os.Stat(controller); err != nil {
		return "", errors.Wrapf(err, `Passthru script "%s" does not exist under %s`, passthru, documentRoot)
	}
	return passthru, nil
}

func guessDocumentRoot(path string) string {
	// for Symfony: check if public-dir is setup in composer.json first
	if b, err := ioutil.ReadFile(filepath.Join(path, "composer.json")); err == nil {
		var f map[string]interface{}
		if err := json.Unmarshal(b, &f); err == nil {
			if f1, ok := f["extra"]; ok {
				extra := f1.(map[string]interface{})
				if f2, ok := extra["public-dir"]; ok {
					return filepath.Join(path, f2.(string))
				}
			}
		}
	}

	docroots := []string{
		"public",
		"web",
		"docroot", // Drupal
	}
	for _, docroot := range docroots {
		if _, err := os.Stat(filepath.Join(path, docroot)); err == nil {
			return filepath.Join(path, docroot)
		}
	}
	return path
}

func guessPassthru(path string) string {
	indexes := []string{
		"index_dev.php",
		"index.php",
		"app_dev.php",
		"app.php",
	}
	for _, index := range indexes {
		if _, err := os.Stat(filepath.Join(path, index)); err == nil {
			return index
		}
	}
	return "index.php"
}
