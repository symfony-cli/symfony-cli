package book

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"github.com/symfony-cli/phpstore"
	"github.com/symfony-cli/symfony-cli/util"
	"github.com/symfony-cli/terminal"
)

func (b *Book) CheckRepository() error {
	if _, err := os.Stat(filepath.Join(b.Dir, ".git")); os.IsNotExist(err) {
		return errors.New("the current directory is not a clone of the book repository, no .git directory found")
	}
	if !b.Force {
		cmd := exec.Command("git", "config", "--get", "remote.origin.url")
		cmd.Env = os.Environ()
		out, err := cmd.CombinedOutput()
		if err != nil {
			return errors.Errorf("unable to get the Git information:\n%s\n%s", err, out)
		}
		if !strings.HasPrefix(string(out), "https://github.com/the-fast-track/book-") {
			return errors.New("the current directory does not seem to be a clone of the book repository")
		}
	}
	return nil
}

func CheckRequirements() (bool, error) {
	ready := true

	// Git
	if _, err := exec.LookPath("git"); err != nil {
		ready = false
		terminal.Println("<error>[KO]</> Cannot find Git, please install it <href=https://git-scm.com/>https://git-scm.com/</>")
	} else {
		terminal.Println("<info>[OK]</> Git installed")
	}

	// PHP
	minv, err := version.NewVersion("7.2.4")
	if err != nil {
		return false, err
	}
	store := phpstore.New(util.GetHomeDir(), true, nil)
	wd, err := os.Getwd()
	if err != nil {
		return false, err
	}
	v, _, _, _ := store.BestVersionForDir(wd)
	if v == nil {
		ready = false
		terminal.Println("<error>[KO]</> Cannot find PHP, please install it <href=https://php.net/>https://php.net/</>")
	} else {
		if v.FullVersion.GreaterThan(minv) {
			terminal.Printfln("<info>[OK]</> PHP installed version %s (%s)", v.FullVersion, v.PHPPath)
		} else {
			ready = false
			terminal.Printfln("<error>[KO]</> PHP installed; version %s found but we need version 7.2.5+ (%s)", v.FullVersion, v.PHPPath)
		}
	}

	// PHP extensions
	if v != nil {
		exts := map[string]string{
			"json":      "required",
			"session":   "required",
			"ctype":     "required",
			"tokenizer": "required",
			"xml":       "required",
			"intl":      "required",
			"pdo_pgsql": "required",
			"mbstring":  "required",
			"xsl":       "required",
			"openssl":   "required",
			"sodium":    "required",
			"curl":      "optional - needed only for chapter 17 (Panther)",
			"zip":       "optional - needed only for chapter 17 (Panther)",
			"gd":        "optional - needed only for chapter 23 (Imagine)",
			"redis":     "optional - needed only for chapter 31",
			"amqp":      "optional - needed only for chapter 32",
		}
		phpexts := getPhpExtensions(v)
		for ext, reason := range exts {
			if _, ok := phpexts[ext]; !ok {
				if reason == "required" {
					ready = false
					terminal.Printfln(`<error>[KO]</> PHP extension "%s" <error>not found</>, please install it - <comment>%s</>`, ext, reason)
				} else {
					terminal.Printfln(`<warning>[KO]</> PHP extension "%s" <warning>not found</>, <comment>%s</>`, ext, reason)
				}
			} else {
				terminal.Printfln(`<info>[OK]</> PHP extension "%s" installed - <comment>%s</>`, ext, reason)
			}
		}
	}

	// Composer
	if _, err := exec.LookPath("composer"); err != nil {
		ready = false
		terminal.Println("<error>[KO]</> Cannot find Composer, please install it <href=https://getcomposer.org/download/>https://getcomposer.org/download/</>")
	} else {
		terminal.Println("<info>[OK]</> Composer installed")
	}

	// Docker
	if _, err := exec.LookPath("docker"); err != nil {
		ready = false
		terminal.Println("<error>[KO]</> Cannot find Docker, please install it <href=https://www.docker.com/get-started>https://www.docker.com/get-started</>")
	} else {
		terminal.Println("<info>[OK]</> Docker installed")
	}

	// Docker Compose
	if _, err := exec.LookPath("docker-compose"); err != nil {
		ready = false
		terminal.Println("<error>[KO]</> Cannot find Docker Compose, please install it <href=https://docs.docker.com/compose/install/>https://docs.docker.com/compose/install/</>")
	} else {
		terminal.Println("<info>[OK]</> Docker Compose installed")
	}

	// yarn
	if _, err := exec.LookPath("yarn"); err != nil {
		ready = false
		terminal.Println("<error>[KO]</> Cannot find the Yarn package manager, please install it <href=https://yarnpkg.com/>https://yarnpkg.com/</>")
	} else {
		terminal.Println("<info>[OK]</> Yarn installed")
	}

	return ready, nil
}

func getPhpExtensions(php *phpstore.Version) map[string]bool {
	exts := make(map[string]bool)
	var buf bytes.Buffer
	cmd := exec.Command(php.PHPPath, "-m")
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return exts
	}
	for _, ext := range strings.Split(buf.String(), "\n") {
		exts[ext] = true
	}
	return exts
}
