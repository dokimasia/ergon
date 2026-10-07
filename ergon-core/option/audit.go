// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package option

import (
	"fmt"
	"regexp"
	"slices"
)

// advisory matches the identifier of an advisory or of a check, such as PYSEC-2024-1,
// RUSTSEC-2024-0001, GHSA-xxxx-xxxx-xxxx or CKV_AWS_20.
var advisory = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]*$`)

// Severity is the lowest severity of a known vulnerability that fails a scan, on the scale that npm
// audit and the audit of NuGet share.
type Severity string

// The severities of a known vulnerability, from the lowest.
const (
	// SeverityLow fails a scan on every known vulnerability.
	SeverityLow Severity = "low"

	// SeverityModerate fails a scan on a known vulnerability of moderate, high or critical
	// severity.
	SeverityModerate Severity = "moderate"

	// SeverityHigh fails a scan on a known vulnerability of high or critical severity.
	SeverityHigh Severity = "high"

	// SeverityCritical fails a scan on a known vulnerability of critical severity alone.
	SeverityCritical Severity = "critical"
)

// severities are the severities, from the lowest.
var severities = []Severity{SeverityLow, SeverityModerate, SeverityHigh, SeverityCritical}

// Validate returns an error that wraps [ErrInvalid] for an s other than low, moderate, high and
// critical.
func (s Severity) Validate() error {
	if !slices.Contains(severities, s) {
		return fmt.Errorf("%w: severity %q, which is none of low, moderate, high and critical", ErrInvalid, s)
	}
	return nil
}

// Audit is the options of the step audit of a scan that accepts advisories by their identifier,
// such as pip-audit, cargo-audit and checkov.
type Audit struct {
	// Ignore are the identifiers of the advisories, or of the checks, that the scan accepts.
	Ignore []string `yaml:"ignore"`
}

// Validate returns an error that wraps [ErrInvalid] for an identifier that is not a letter or a
// digit followed by letters, digits, '.', '_', ':' and '-', and for an identifier that Ignore names
// twice.
func (a Audit) Validate() error {
	for i, id := range a.Ignore {
		if !advisory.MatchString(id) || slices.Contains(a.Ignore[:i], id) {
			return fmt.Errorf("%w: ignore %q, which is not the identifier of an advisory or is named twice",
				ErrInvalid, id)
		}
	}
	return nil
}

// Threshold is the options of the step audit of a scan that fails at a severity, such as npm
// audit and the audit of NuGet.
type Threshold struct {
	// Severity is the lowest severity of a known vulnerability that fails the scan.
	Severity Severity `yaml:"severity"`
}

// Validate returns the error of [Severity.Validate] for the severity of t.
func (t Threshold) Validate() error {
	return t.Severity.Validate()
}
