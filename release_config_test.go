package main

import (
	"os"
	"slices"
	"testing"

	"gopkg.in/yaml.v2"
)

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
