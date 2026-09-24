package policy_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	yaml "go.yaml.in/yaml/v3"
)

// The release configuration, checked as configuration.
//
// Two rules here used to live in a shell script that inspected images
// AFTER pushing them. That check could only fail once a release had
// happened, and it could not see the thing most likely to go wrong: a
// platform quietly dropped from the list. These run in the gate instead,
// before anything is built.
//
// docs/contracts/release.md §5 is what they defend.

type releaseConfig struct {
	Kos []struct {
		ID        string   `yaml:"id"`
		Platforms []string `yaml:"platforms"`
	} `yaml:"kos"`
	DockersV2 []struct {
		ID         string   `yaml:"id"`
		Dockerfile string   `yaml:"dockerfile"`
		Platforms  []string `yaml:"platforms"`
	} `yaml:"dockers_v2"`
}

// Every architecture a consumer might run, for every image.
//
// A RELEASE IS ALWAYS MULTI-ARCHITECTURE — not because of what any cluster
// runs this week, but because a published artifact is consumed by machines
// whose architecture the publisher does not know and should not have to
// ask about. An image carrying one architecture fails for half its
// consumers, and it fails in somebody else's cluster.
var everyArchitecture = []string{"linux/amd64", "linux/arm64"}

func loadReleaseConfig(t *testing.T) releaseConfig {
	t.Helper()

	raw, err := os.ReadFile(".goreleaser.yaml")
	if err != nil {
		t.Fatal(err)
	}

	var config releaseConfig
	if err := yaml.Unmarshal(raw, &config); err != nil {
		t.Fatal(err)
	}

	if len(config.Kos)+len(config.DockersV2) == 0 {
		t.Fatal("the release configuration builds no images; this test would pass over nothing")
	}

	return config
}

func TestEveryImageIsBuiltForEveryArchitecture(t *testing.T) {
	config := loadReleaseConfig(t)

	check := func(id string, platforms []string) {
		t.Helper()
		for _, want := range everyArchitecture {
			if !slices.Contains(platforms, want) {
				t.Errorf("image %q does not declare %s — a release that ships it would fail for every consumer on that architecture, in their cluster rather than in this one", id, want)
			}
		}
	}

	for _, image := range config.Kos {
		check(image.ID, image.Platforms)
	}

	for _, image := range config.DockersV2 {
		check(image.ID, image.Platforms)
	}
}

// Nothing executes while an image is assembled.
//
// This is what makes building for another architecture a FILE COPY rather
// than emulation: no second runner, no qemu, and a multi-architecture
// release that costs almost nothing. It is load-bearing for the release
// configuration above — which is exactly why it is asserted rather than
// remembered.
//
// A RUN line would also move part of the build inside the image, where it
// is neither reviewed nor reproducible.
func TestNoDockerfileRunsAnythingWhileBuilding(t *testing.T) {
	config := loadReleaseConfig(t)

	if len(config.DockersV2) == 0 {
		t.Skip("no Dockerfile-built images")
	}

	for _, image := range config.DockersV2 {
		t.Run(image.ID, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Clean(image.Dockerfile))
			if err != nil {
				t.Fatal(err)
			}

			for number, line := range strings.Split(string(raw), "\n") {
				if strings.HasPrefix(strings.TrimSpace(line), "RUN ") {
					t.Errorf("%s:%d executes while the image is assembled: %q", image.Dockerfile, number+1, strings.TrimSpace(line))
				}
			}
		})
	}
}

// The destination is not written down here.
//
// One repository publishes to ONE registry — public to ghcr, private to
// ECR — and which one is the caller's to say, so that a local loop, a CI
// loop and a release differ in an environment variable rather than in
// what they build. A literal registry in this file is a release that can
// only go one place and a local loop that has to be a different file.
func TestTheImageDestinationIsNotHardCoded(t *testing.T) {
	raw, err := os.ReadFile(".goreleaser.yaml")
	if err != nil {
		t.Fatal(err)
	}

	for number, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "repositories:") && !strings.HasPrefix(trimmed, "images:") {
			continue
		}
		if !strings.Contains(trimmed, "{{ .Env.IMAGE_REPO }}") {
			t.Errorf(".goreleaser.yaml:%d names a registry instead of taking one: %q", number+1, trimmed)
		}
	}
}
