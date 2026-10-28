// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"fmt"

	"go.dokimi.dev/ergon/core/option"
)

// The repositories on GitHub whose releases publish the release binaries of the section.
const (
	tflintRepository        = "terraform-linters/tflint"
	terraformDocsRepository = "terraform-docs/terraform-docs"
)

// rulesetRepository is the start of the repository on GitHub whose releases publish a ruleset of
// tflint, before the name of the ruleset, such as aws.
const rulesetRepository = "terraform-linters/tflint-ruleset-"

// rulesetProgram is the start of the program of a ruleset of tflint, before the name of the
// ruleset, which tflint loads from its plugin directory.
const rulesetProgram = "tflint-ruleset-"

// windows is the system of Windows, whose programs end in .exe.
const windows = "windows"

// TFLint is the release binary of tflint, terraform-linters/tflint on GitHub, whose release
// publishes a .zip for every platform.
type TFLint struct {
	option.Binary `yaml:",inline"`
}

var _ option.Release = TFLint{}

// Asset returns the archive of tflint for p from the release of the version of t on GitHub:
// tflint_<os>_<arch>.zip, with the program at its root, and tflint.exe on Windows. It returns an
// error that wraps [option.ErrNoAsset] for a platform that is not valid.
func (t TFLint) Asset(p option.Platform) (option.Asset, error) {
	if p.Validate() != nil {
		return option.Asset{}, fmt.Errorf("%w: tflint has no asset for %q", option.ErrNoAsset, p)
	}
	program := "tflint"
	if p.OS() == windows {
		program += ".exe"
	}
	return option.Asset{
		URL: "https://github.com/" + tflintRepository + "/releases/download/v" + t.Version + "/tflint_" + p.OS() +
			"_" + p.Arch() + ".zip",
		Program: program,
	}, nil
}

// Repository returns terraform-linters/tflint, the repository on GitHub whose releases publish
// tflint.
func (TFLint) Repository() string {
	return tflintRepository
}

// AWSRuleset is the release binary of tflint-ruleset-aws, the ruleset of tflint for the provider of
// AWS. Its release publishes a .zip for each platform but Windows on 64-bit ARM.
type AWSRuleset struct {
	option.Binary `yaml:",inline"`
}

var _ option.Release = AWSRuleset{}

// Asset returns the archive of the ruleset for p, as [rulesetAsset] states it for aws.
func (r AWSRuleset) Asset(p option.Platform) (option.Asset, error) {
	return rulesetAsset(RulesetAWS, r.Version, p)
}

// Repository returns terraform-linters/tflint-ruleset-aws, the repository on GitHub whose releases
// publish the ruleset.
func (AWSRuleset) Repository() string {
	return rulesetRepository + string(RulesetAWS)
}

// AzureRMRuleset is the release binary of tflint-ruleset-azurerm, the ruleset of tflint for the
// provider of Azure. Its release publishes a .zip for each platform but Windows on 64-bit ARM.
type AzureRMRuleset struct {
	option.Binary `yaml:",inline"`
}

var _ option.Release = AzureRMRuleset{}

// Asset returns the archive of the ruleset for p, as [rulesetAsset] states it for azurerm.
func (r AzureRMRuleset) Asset(p option.Platform) (option.Asset, error) {
	return rulesetAsset(RulesetAzureRM, r.Version, p)
}

// Repository returns terraform-linters/tflint-ruleset-azurerm, the repository on GitHub whose
// releases publish the ruleset.
func (AzureRMRuleset) Repository() string {
	return rulesetRepository + string(RulesetAzureRM)
}

// GoogleRuleset is the release binary of tflint-ruleset-google, the ruleset of tflint for the
// provider of Google Cloud. Its release publishes a .zip for each platform but Windows on 64-bit
// ARM.
type GoogleRuleset struct {
	option.Binary `yaml:",inline"`
}

var _ option.Release = GoogleRuleset{}

// Asset returns the archive of the ruleset for p, as [rulesetAsset] states it for google.
func (r GoogleRuleset) Asset(p option.Platform) (option.Asset, error) {
	return rulesetAsset(RulesetGoogle, r.Version, p)
}

// Repository returns terraform-linters/tflint-ruleset-google, the repository on GitHub whose
// releases publish the ruleset.
func (GoogleRuleset) Repository() string {
	return rulesetRepository + string(RulesetGoogle)
}

// TerraformDocs is the release binary of terraform-docs, terraform-docs/terraform-docs on GitHub,
// whose release publishes a .tar.gz for each platform, and a .zip for Windows.
type TerraformDocs struct {
	option.Binary `yaml:",inline"`
}

var _ option.Release = TerraformDocs{}

// Asset returns the archive of terraform-docs for p from the release of the version of d on GitHub:
// terraform-docs-v<version>-<os>-<arch>.tar.gz with the program at its root, and a .zip with
// terraform-docs.exe on Windows. It returns an error that wraps [option.ErrNoAsset] for a platform
// that is not valid.
func (d TerraformDocs) Asset(p option.Platform) (option.Asset, error) {
	if p.Validate() != nil {
		return option.Asset{}, fmt.Errorf("%w: terraform-docs has no asset for %q", option.ErrNoAsset, p)
	}
	release := "https://github.com/" + terraformDocsRepository + "/releases/download/v" + d.Version +
		"/terraform-docs-v" + d.Version + "-" + p.OS() + "-" + p.Arch()
	if p.OS() == windows {
		return option.Asset{URL: release + ".zip", Program: "terraform-docs.exe"}, nil
	}
	return option.Asset{URL: release + ".tar.gz", Program: "terraform-docs"}, nil
}

// Repository returns terraform-docs/terraform-docs, the repository on GitHub whose releases publish
// terraform-docs.
func (TerraformDocs) Repository() string {
	return terraformDocsRepository
}

// rulesetAsset returns the archive of the ruleset r of tflint for p from the release of version on
// GitHub: tflint-ruleset-<name>_<os>_<arch>.zip, with the program tflint-ruleset-<name> at its root,
// and tflint-ruleset-<name>.exe on Windows. It returns an error that wraps [option.ErrNoAsset] for a
// platform that is not valid, and for Windows on 64-bit ARM, which the releases of the rulesets do
// not publish.
func rulesetAsset(r Ruleset, version string, p option.Platform) (option.Asset, error) {
	if p.Validate() != nil || p == option.WindowsARM64 {
		return option.Asset{}, fmt.Errorf("%w: tflint-ruleset-%s has no asset for %q", option.ErrNoAsset, r, p)
	}
	program := rulesetProgram + string(r)
	if p.OS() == windows {
		program += ".exe"
	}
	return option.Asset{
		URL: "https://github.com/" + rulesetRepository + string(r) + "/releases/download/v" + version + "/" +
			rulesetProgram + string(r) + "_" + p.OS() + "_" + p.Arch() + ".zip",
		Program: program,
	}, nil
}
