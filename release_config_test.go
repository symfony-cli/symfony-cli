package main

import (
	"os"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v2"
)

type releaseWorkflow struct {
	Jobs map[string]releaseWorkflowJob `yaml:"jobs"`
}

type releaseWorkflowJob struct {
	Steps []releaseWorkflowStep `yaml:"steps"`
}

type releaseWorkflowStep struct {
	Name string            `yaml:"name"`
	Uses string            `yaml:"uses"`
	Run  string            `yaml:"run"`
	With map[string]string `yaml:"with"`
	Env  map[string]string `yaml:"env"`
}

func readReleaseWorkflow(t *testing.T, path string) releaseWorkflow {
	t.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var workflow releaseWorkflow
	if err := yaml.Unmarshal(contents, &workflow); err != nil {
		t.Fatal(err)
	}

	return workflow
}

func TestReleaseWorkflowUsesDockerContainerBuildx(t *testing.T) {
	workflow := readReleaseWorkflow(t, ".github/workflows/releaser.yml")
	releaser, ok := workflow.Jobs["releaser"]
	if !ok {
		t.Fatal("releaser job not found")
	}

	qemuStep := -1
	buildxStep := -1
	goReleaserStep := -1
	for index, step := range releaser.Steps {
		switch step.Uses {
		case "docker/setup-qemu-action@v4":
			qemuStep = index
		case "docker/setup-buildx-action@v4":
			buildxStep = index
			if step.With["driver"] != "docker-container" {
				t.Errorf("unexpected Docker Buildx driver %q", step.With["driver"])
			}
		case "goreleaser/goreleaser-action@v7":
			if goReleaserStep == -1 {
				goReleaserStep = index
			}
		}
	}

	if qemuStep == -1 {
		t.Error("QEMU setup step not found")
	}
	if buildxStep == -1 {
		t.Error("Docker Buildx setup step not found")
	}
	if goReleaserStep == -1 {
		t.Error("GoReleaser step not found")
	}
	if qemuStep >= buildxStep || buildxStep >= goReleaserStep {
		t.Errorf("unexpected release setup order: QEMU=%d, Docker Buildx=%d, GoReleaser=%d", qemuStep, buildxStep, goReleaserStep)
	}
}

func TestReleaseWorkflowChecksCredentialsBeforePublishing(t *testing.T) {
	workflow := readReleaseWorkflow(t, ".github/workflows/releaser.yml")
	releaser, ok := workflow.Jobs["releaser"]
	if !ok {
		t.Fatal("releaser job not found")
	}

	credentialStep := -1
	goReleaserStep := -1
	for index, step := range releaser.Steps {
		switch step.Name {
		case "Check release credentials":
			credentialStep = index
			if step.Env["GH_TOKEN"] != "${{ secrets.GH_PAT }}" {
				t.Errorf("unexpected Homebrew token %q", step.Env["GH_TOKEN"])
			}
			if step.Env["CLOUDSMITH_API_KEY"] != "${{ secrets.CLOUDSMITH_API_KEY }}" {
				t.Errorf("unexpected Cloudsmith token %q", step.Env["CLOUDSMITH_API_KEY"])
			}
			if !strings.Contains(step.Run, "permissions.push") || !strings.Contains(step.Run, "cloudsmith whoami") {
				t.Errorf("credential checks are incomplete: %q", step.Run)
			}
		case "Run GoReleaser":
			goReleaserStep = index
		}
	}

	if credentialStep == -1 {
		t.Error("release credential check not found")
	}
	if goReleaserStep == -1 {
		t.Error("GoReleaser release step not found")
	}
	if credentialStep >= goReleaserStep {
		t.Errorf("release credentials are checked after publishing: credentials=%d, GoReleaser=%d", credentialStep, goReleaserStep)
	}
}

func TestCloudsmithRecoveryWorkflowUsesPublishedPackages(t *testing.T) {
	const path = ".github/workflows/cloudsmith_recovery.yml"

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contents), "workflow_dispatch:") {
		t.Error("workflow dispatch trigger not found")
	}

	workflow := readReleaseWorkflow(t, path)
	publish, ok := workflow.Jobs["publish"]
	if !ok {
		t.Fatal("publish job not found")
	}

	downloadStep := -1
	credentialStep := -1
	uploadStep := -1
	for index, step := range publish.Steps {
		switch step.Name {
		case "Download release packages":
			downloadStep = index
			if !strings.Contains(step.Run, "gh release download") {
				t.Errorf("release download command not found: %q", step.Run)
			}
		case "Check Cloudsmith credentials":
			credentialStep = index
			if step.Env["CLOUDSMITH_API_KEY"] != "${{ secrets.CLOUDSMITH_API_KEY }}" {
				t.Errorf("unexpected Cloudsmith token %q", step.Env["CLOUDSMITH_API_KEY"])
			}
		case "Upload packages":
			uploadStep = index
			for _, format := range []string{"push deb", "push rpm", "push alpine"} {
				if !strings.Contains(step.Run, format) {
					t.Errorf("Cloudsmith %s command not found", format)
				}
			}
		}
	}

	if downloadStep == -1 || credentialStep == -1 || uploadStep == -1 {
		t.Fatalf("incomplete recovery workflow: download=%d, credentials=%d, upload=%d", downloadStep, credentialStep, uploadStep)
	}
	if downloadStep >= credentialStep || credentialStep >= uploadStep {
		t.Errorf("unexpected recovery order: download=%d, credentials=%d, upload=%d", downloadStep, credentialStep, uploadStep)
	}
}

func TestGoReleaserCosignConfiguration(t *testing.T) {
	contents, err := os.ReadFile(".goreleaser.yml")
	if err != nil {
		t.Fatal(err)
	}

	var config struct {
		Signs []struct {
			Command     string   `yaml:"cmd"`
			Signature   string   `yaml:"signature"`
			Certificate string   `yaml:"certificate"`
			Args        []string `yaml:"args"`
			Artifacts   string   `yaml:"artifacts"`
		} `yaml:"signs"`
	}
	if err := yaml.Unmarshal(contents, &config); err != nil {
		t.Fatal(err)
	}

	var cosignConfigurations int
	for _, sign := range config.Signs {
		if sign.Command != "cosign" {
			continue
		}

		cosignConfigurations++
		if sign.Signature != "${artifact}.sigstore.json" {
			t.Errorf("unexpected signature template %q", sign.Signature)
		}
		if sign.Certificate != "" {
			t.Errorf("unexpected separate certificate %q", sign.Certificate)
		}
		if want := []string{"sign-blob", "--bundle=${signature}", "${artifact}", "--yes"}; !slices.Equal(sign.Args, want) {
			t.Errorf("unexpected cosign arguments %q", sign.Args)
		}
		if sign.Artifacts != "all" {
			t.Errorf("unexpected signed artifacts %q", sign.Artifacts)
		}
	}

	if cosignConfigurations != 1 {
		t.Errorf("expected one cosign configuration, got %d", cosignConfigurations)
	}
}
