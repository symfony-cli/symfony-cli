package git

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

func Fetch(cwd, remote, branch string) error {
	args := []string{"fetch", remote}

	if branch != "" {
		args = append(args, branch)
	}

	_, err := execGitQuiet(cwd, args...)

	return err
}

func Clone(url, dir string) error {
	args := []string{"clone", url, dir}

	return execGit(filepath.Dir(dir), args...)
}

func Push(cwd, remote, ref, remoteRef string) error {
	if ref == "" {
		return errors.New("ref is required when pushing")
	}

	if remoteRef != "" {
		ref = fmt.Sprintf("%s:%s", ref, remoteRef)
	}

	args := []string{"push", "--progress", remote, ref}

	return execGit(cwd, args...)
}

func GetUpstreamBranch(cwd string, remoteNames ...string) string {
	args := []string{"rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}"}

	out, err := execGitQuiet(cwd, args...)
	if err != nil {
		return ""
	}

	upstream := strings.Trim(out.String(), " \n")
	if !strings.Contains(upstream, "/") {
		return ""
	}

	remote := strings.SplitN(upstream, "/", 2)
	for _, remoteName := range remoteNames {
		if remote[0] == remoteName {
			return remote[1]
		}
	}

	return ""
}
