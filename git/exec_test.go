package git

import (
	"bytes"
	"testing"

	. "gopkg.in/check.v1"
)

type GitSuite struct{}

var _ = Suite(&GitSuite{})

func TestGit(t *testing.T) { TestingT(t) }

var expectedOutput = `
  Enumerating objects: 7, done.
  Counting objects: 0% (0/7), done.` + "\r" + `  Counting objects: 57% (4/7), done.` + "\r" + `  Counting objects: 100% (7/7), done.
  Delta compression using up to 4 threads
  Compressing objects: 100% (4/4), done.
  Writing objects: 100% (4/4), 1.05 KiB | 1.05 MiB/s, done.
  Total 4 (delta 2), reused 1 (delta 0)

  Validating submodules

  Validating configuration files

  Processing activity: Tugdual Saunier pushed to test-deployment-failing
      Found 1 new commit

      Building application 'app' (runtime type: golang:1.11, tree: 5e8278e)
        Generating runtime configuration.

        Executing build hook...
          W: + curl -s https://get.symfony.com/cloud/configurator
          W: + bash
          W: + mkdir -p /app/.global/bin/
          W: + tar -C /app/.global/bin/ -jxpf -
          W: + curl -s https://get.symfony.com/cloud/tools.tar.bz2
          W: + echo -e ''
          W: + echo 'export PATH=/app/bin:/app/vendor/bin:$PATH $(scenv)'
          W: + echo eyJzaXplIjogIkFVVE8iLCAiZGlzayI6IDEwMjQsICJhY2Nlc3MiOiB7InNzaCI6ICJjb250cmlidXRvciJ9LCAicmVsYXRpb25zaGlwcyI6IHt9LCAibW91bnRzIjogeyIvdmFyIjogeyJzb3VyY2UiOiAibG9jYWwiLCAic291cmNlX3BhdGgiOiAidmFyIn19LCAidGltZXpvbmUiOiBudWxsLCAidmFyaWFibGVzIjoge30sICJuYW1lIjogImFwcCIsICJ0eXBlIjogImdvbGFuZzoxLjExIiwgInJ1bnRpbWUiOiB7fSwgInByZWZsaWdodCI6IHsiZW5hYmxlZCI6IHRydWUsICJpZ25vcmVkX3J1bGVzIjogW119LCAiZGVwZW5kZW5jaWVzIjoge30sICJidWlsZCI6IHsiZmxhdm9yIjogIm5vbmUifSwgIndlYiI6IHsibG9jYXRpb25zIjogeyIvIjogeyJyb290IjogbnVsbCwgImV4cGlyZXMiOiAiLTFzIiwgInBhc3N0aHJ1IjogdHJ1ZSwgInNjcmlwdHMiOiB0cnVlLCAiYWxsb3ciOiBmYWxzZSwgImhlYWRlcnMiOiB7fSwgInJ1bGVzIjoge319fSwgImNvbW1hbmRzIjogeyJzdGFydCI6ICJzY2VudiAvYXBwL3N0cmlwZS1ub3RpZmljYXRpb25zIiwgInN0b3AiOiBudWxsfSwgInVwc3RyZWFtIjogeyJzb2NrZXRfZmFtaWx5IjogInRjcCIsICJwcm90b2NvbCI6ICJodHRwIn0sICJtb3ZlX3RvX3Jvb3QiOiBmYWxzZX0sICJob29rcyI6IHsiYnVpbGQiOiAic2V0IC1lIC14XG5cblxuXG5cblxuXG5cblxuXG5jdXJsIC1zIGh0dHBzOi8vZ2V0LnN5bWZvbnkuY29tL2Nsb3VkL2NvbmZpZ3VyYXRvciB8ICg+JjIgYmFzaClcbmdvIGJ1aWxkXG4iLCAiZGVwbG95IjogbnVsbCwgInBvc3RfZGVwbG95IjogbnVsbH0sICJjcm9ucyI6IHt9LCAid29ya2VycyI6IHt9fQ==
          W: + base64 --decode
          W: + json_pp
          W: + grep '"type" : "php'
          W: + base64 --decode
          W: + exit 0
          W: + go build
          W: go: finding github.com/nlopes/slack v0.4.0
          W: [...]
          W: go: downloading github.com/gorilla/websocket v1.4.0
          W: go: downloading github.com/pkg/errors v0.8.0

        Executing pre-flight checks...

        Compressing application.
        Beaming package to its final destination.

      Provisioning certificates
        Environment certificates
        - certificate d22187d: expiring on 2019-01-28 07:13:00+00:00, covering test-deployment-failing-gbppxsi-4xfrp6lcgobc4.eu.s5y.io


      Re-deploying environment 4xfrp6lcgobc4-test-deployment-failing-gbppxsi
        Environment configuration
          app (type: golang:1.11, size: S, disk: 1024)

        Environment routes
          http://test-deployment-failing-gbppxsi-4xfrp6lcgobc4.eu.s5y.io/ redirects to https://test-deployment-failing-gbppxsi-4xfrp6lcgobc4.eu.s5y.io/
          https://test-deployment-failing-gbppxsi-4xfrp6lcgobc4.eu.s5y.io/ is served by application 'app'


  To git.eu.s5y.io:4xfrp6lcgobc4.git
     72daff6..2c02b16  HEAD -> test-deployment-failing
`

func (ts *GitSuite) TestGitOutputWriter(t *C) {
	var buf bytes.Buffer
	writer := gitOutputWriter{output: &buf}

	if _, err := writer.Write([]byte("\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("Enumerating objects: 7, done.\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("Counting objects: 0% (0/7), done.\r")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("Counting objects: 57% (4/7), done.\r")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("Counting objects: 100% (7/7), done.\nDelta compression using up to 4 threads\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("Compressing objects: 100% (4/4), done.\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("Writing objects: 100% (4/4), 1.05 KiB | 1.05 MiB/s, done.\nTotal 4 (delta 2), reused 1 (delta 0)\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("Validating sub")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("modules\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("Validating configuration files\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("Processing activity: Tugdual Saunier pushed to test-deployment-failing\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("    Found 1 new commit\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("    Building application 'app' (runtime type: golang:1.11, tree: 5e8278e)\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("      Generating runtime configuration.\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("      Executing build hook...\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("        W: + curl -s https://get.symfony.com/cloud/configurator\n        W: + bash\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("        W: + mkdir -p /app/.global/bin/\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("        W: + tar -C /app/.global/bin/ -jxpf -\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("        W: + curl -s https://get.symfony.com/cloud/tools.tar.bz2\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("        W: + echo -e ''\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("        W: + echo 'export PATH=/app/bin:/app/vendor/bin:$PATH $(scenv)'\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if n, err := writer.Write([]byte("        W: + echo eyJzaXplIjogIkFVVE8iLCAiZGlzayI6IDEwMjQsICJhY2Nlc3MiOiB7InNzaCI6ICJjb250cmlidXRvciJ9LCAicmVsYXRpb25zaGlwcyI6IHt9LCAibW91bnRzIjogeyIvdmFyIjogeyJzb3VyY2UiOiAibG9jYWwiLCAic291cmNlX3BhdGgiOiAidmFyIn19LCAidGltZXpvbmUiOiBudWxsLCAidmFyaWFibGVzIjoge30sICJuYW1lIjogImFwcCIsICJ0eXBlIjogImdvbGFuZzoxLjExIiwgInJ1bnRpbWUiOiB7fSwgInByZWZsaWdodCI6IHsiZW5hYmxlZCI6IHRydWUsICJpZ25vcmVkX3J1bGVzIjogW119LCAiZGVwZW5kZW5jaWVzIjoge30sICJidWlsZC")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	} else if n != 440 {
		t.Fatalf("wrong char count returned by gitOutputWriter.Write: got %v, 440 expected", n)
	}

	if n, err := writer.Write([]byte("I6IHsiZmxhdm9yIjogIm5vbmUifSwgIndlYiI6IHsibG9jYXRpb25zIjogeyIvIjogeyJyb290IjogbnVsbCwgImV4cGlyZXMiOiAiLTFzIiwgInBhc3N0aHJ1IjogdHJ1ZSwgInNjcmlwdHMiOiB0cnVlLCAiYWxsb3ciOiBmYWxzZSwgImhlYWRlcnMiOiB7fSwgInJ1bGVzIjoge319fSwgImNvbW1hbmRzIjogeyJzdGFydCI6ICJzY2VudiAvYXBwL3N0cmlwZS1ub3RpZmljYXRpb25zIiwgInN0b3AiOiBudWxsfSwgInVwc3RyZWFtIjogeyJzb2NrZXRfZmFtaWx5IjogInRjcCIsICJwcm90b2NvbCI6ICJodHRwIn0sICJtb3ZlX3RvX3Jvb3QiOiBmYWxzZX0sICJob29rcyI6IHsiYnVpbGQiOiAic2V0IC1lIC14XG5cblxuXG5cblxuXG5cblxuXG5jdXJsIC1zIGh0dHBzOi8vZ2V0LnN5bWZvbnkuY29tL2Nsb3VkL2NvbmZpZ3VyYXRvciB8ICg+JjIgYmFzaClcbmdvIGJ1aWxkXG4iLCAiZGVwbG95IjogbnVsbCwgInBvc3RfZGVwbG95IjogbnVsbH0sICJjcm9ucyI6IHt9LCAid29ya2VycyI6IHt9fQ==\n        W: + base64 --decode\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	} else if n != 712 {
		t.Fatalf("wrong char count returned by gitOutputWriter.Write: got %v, 712 expected", n)
	}

	if _, err := writer.Write([]byte("        W: + json_pp\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("        W: + grep '\"type\" : \"php'\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("        W: + base64 --decode\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("        W: + exit 0\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("        W: + go build\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("        W: go: finding github.com/nlopes/slack v0.4.0\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("        W: [...]\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("        W: go: downloading github.com/gorilla/websocket v1.4.0\n        W: go: downloading github.com/pkg/errors v0.8.0\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("\n      Executing pre-flight checks...\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("\n      Compressing application.\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("      Beaming package to its final destination.\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("\n    Provisioning certificates\n      Environment certificates\n      - certificate d22187d: expiring on 2019-01-28 07:13:00+00:00, covering test-deployment-failing-gbppxsi-4xfrp6lcgobc4.eu.s5y.io\n\n\n    Re-deploying environment 4xfrp6lcgobc4-test-deployment-failing-gbppxsi\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("      Environment configuration\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("        app (type: golang:1.11, size: S, disk: 1024)\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("      Environment routes\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("        http://test-deployment-failing-gbppxsi-4xfrp6lcgobc4.eu.s5y.io/ redirects to https://test-deployment-failing-gbppxsi-4xfrp6lcgobc4.eu.s5y.io/\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("        https://test-deployment-failing-gbppxsi-4xfrp6lcgobc4.eu.s5y.io/ is served by application 'app'\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	if _, err := writer.Write([]byte("To git.eu.s5y.io:4xfrp6lcgobc4.git\n   72daff6..2c02b16  HEAD -> test-deployment-failing\n")); err != nil {
		t.Fatalf("gitOutputWriter.Write returned an unexcepted error: %v", err)
	}

	output := buf.String()
	t.Assert(output, Equals, expectedOutput)
}
