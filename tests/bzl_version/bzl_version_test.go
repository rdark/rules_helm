package main

import (
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/bazelbuild/rules_go/go/runfiles"
)

func readRunfile(rlocation string) (string, error) {
	path, err := runfiles.Rlocation(rlocation)
	if err != nil {
		return "", err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// parseModuleVersion returns the version in the `module()` call of a MODULE.bazel file.
func parseModuleVersion(content string) (string, error) {
	var moduleFound bool = false
	for _, line := range strings.Split(content, "\n") {
		text := strings.TrimRight(line, "\r")
		if moduleFound {
			if strings.HasSuffix(text, ")") {
				return "", errors.New("Failed to parse version from module section")
			}
			split := strings.SplitN(text, " = ", 2)
			if len(split) < 2 {
				continue
			}
			param, val := split[0], split[1]
			if strings.Trim(param, " ") == "version" {
				return strings.Trim(val, "\" ,"), nil
			}

		} else if strings.HasPrefix(text, "module(") {
			moduleFound = true
			continue
		}
	}
	return "", errors.New("Failed to parse version from MODULE.bazel")
}

// parseBazelDepVersion returns the version of a `bazel_dep()` call on a single line.
func parseBazelDepVersion(content string, name string) (string, error) {
	re := regexp.MustCompile(`(?m)^bazel_dep\(name = "` + regexp.QuoteMeta(name) + `", version = "([^"]*)"`)
	match := re.FindStringSubmatch(content)
	if match == nil {
		return "", errors.New("Failed to find a bazel_dep on " + name)
	}
	return match[1], nil
}

func TestVersionFiles(t *testing.T) {
	version, found := os.LookupEnv("VERSION")
	if !found {
		t.Fatal("Couldn't parse VERSION from environment.")
	}

	rulesHelm, err := readRunfile("rules_helm/MODULE.bazel")
	if err != nil {
		t.Fatalf("Error reading file MODULE.bazel: %v", err)
	}
	// `rules_helm_img` is released from the same tag, so it has the same version
	// and requires exactly this version of `rules_helm`.
	rulesHelmImg, err := readRunfile("rules_helm_img/MODULE.bazel")
	if err != nil {
		t.Fatalf("Error reading file modules/rules_helm_img/MODULE.bazel: %v", err)
	}

	for _, check := range []struct {
		description string
		parse       func() (string, error)
	}{
		{"MODULE.bazel", func() (string, error) { return parseModuleVersion(rulesHelm) }},
		{"modules/rules_helm_img/MODULE.bazel", func() (string, error) { return parseModuleVersion(rulesHelmImg) }},
		{"the rules_helm dependency of modules/rules_helm_img/MODULE.bazel", func() (string, error) { return parseBazelDepVersion(rulesHelmImg, "rules_helm") }},
	} {
		got, err := check.parse()
		if err != nil {
			t.Errorf("Error reading the version of %s: %v", check.description, err)
		} else if got != version {
			t.Errorf("version.bzl (%s) != %s (%s)", version, check.description, got)
		}
	}
}
