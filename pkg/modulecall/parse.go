package modulecall

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/keilerkonzept/terraform-module-versions/pkg/source"
)

type Parsed struct {
	Source            *source.Source
	Version           *semver.Version
	VersionString     string
	Constraints       *semver.Constraints
	ConstraintsString string
	Raw               tfconfig.ModuleCall
}

func Parse(raw tfconfig.ModuleCall) (*Parsed, error) {
	src, err := source.Parse(raw.Source)
	if err != nil {
		return nil, fmt.Errorf("parse module call source: %w", err)
	}
	out := Parsed{Source: src, Raw: raw}
	switch {
	case src.Git != nil:
		if ref := src.Git.RefValue; ref != nil {
			version, err := semver.NewVersion(*ref)
			if err == nil {
				out.Version = version
			}
			out.VersionString = *ref
		}
		if raw.Version == "" {
			return &out, nil
		}
		// this adds (non-terraform-standard..) support for version constraints to Git sources
		constraints, err := semver.NewConstraint(raw.Version)
		if err != nil {
			return nil, fmt.Errorf("parse constraint %q: %w", raw.Version, err)
		}
		out.Constraints = constraints
		out.ConstraintsString = raw.Version
	case src.Registry != nil:
		if raw.Version == "" {
			return &out, nil
		}

		terraformConstrainRegex := regexp.MustCompile("^(?:=|!=|>|<|>=|<=|~>)?(.+)$")
		if terraformConstrainRegex.Match([]byte(raw.Version)) {
			extractedVersion := terraformConstrainRegex.FindStringSubmatch(raw.Version)[1]
			version, err := semver.NewVersion(extractedVersion)
			if err == nil { // interpret a single-version constraint as a pinned version
				out.Version = version
				out.VersionString = version.String()
			}
		} else {
			version, err := semver.NewVersion(raw.Version)
			if err == nil { // interpret a single-version constraint as a pinned version
				out.Version = version
				out.VersionString = version.String()
			}
		}
		// constraints, err := semver.NewConstraint(raw.Version)
		var constraints *semver.Constraints
		if strings.Contains(raw.Version, "~>") { // handle pessimistic contraint
			pessimisticVersion := strings.ReplaceAll(raw.Version, "~>", "^")
			constraints, err = semver.NewConstraint(pessimisticVersion)
		} else {
			constraints, err = semver.NewConstraint(raw.Version)
		}
		if err != nil {
			return nil, fmt.Errorf("parse constraint %q: %w", raw.Version, err)
		}
		out.Constraints = constraints
		out.ConstraintsString = constraints.String()
	}
	return &out, nil
}
