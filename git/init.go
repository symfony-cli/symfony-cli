package git

import "bytes"

func Init(dir string, debug bool) (*bytes.Buffer, error) {
	return doExecGit(dir, []string{"init"}, !debug)
}

func AddAndCommit(dir, msg string, debug bool) (*bytes.Buffer, error) {
	cmds := [][]string{
		{"add", "."},
		{"commit", "-a", "-m", msg},
	}
	for _, cmd := range cmds {
		if content, err := doExecGit(dir, cmd, !debug); err != nil {
			return content, err
		}
	}
	return nil, nil
}
