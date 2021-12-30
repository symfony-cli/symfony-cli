package git

import (
	"strings"
)

func GetCurrentBranch(cwd string) string {
	args := []string{"symbolic-ref", "--short", "HEAD"}

	out, err := execGitQuiet(cwd, args...)
	if err != nil {
		return ""
	}

	return strings.Trim(out.String(), " \n")
}

func ResetHard(cwd, reference string) error {
	_, err := execGitQuiet(cwd, "reset", "--hard", reference)

	return err
}
