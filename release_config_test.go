package main

import (
	"os"
	"slices"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestReleaseWorkflowUsesDockerContainerBuildx(t *testing.T) {
	contents, err := os.ReadFile(".github/workflows/releaser.yml")
	if err != nil {
		t.Fatal(err)
	}

	var workflow struct {
		Jobs map[string]struct {
			Steps []struct {
				Uses string            `yaml:"uses"`
				With map[string]string `yaml:"with"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(contents, &workflow); err != nil {
		t.Fatal(err)
	}

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
