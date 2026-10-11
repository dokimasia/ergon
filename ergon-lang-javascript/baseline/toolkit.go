// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"fmt"
	"slices"

	"go.dokimi.dev/ergon/core/option"
)

// Toolkit is the toolkit that formats, lints and tests the JavaScript and TypeScript sources of a
// repository.
type Toolkit string

// The toolkits of the section js.
const (
	// ToolkitBiome formats and lints the sources with Biome and the managed biome.json, checks the
	// types of TypeScript with tsc, and runs the test script of package.json.
	ToolkitBiome Toolkit = "biome"

	// ToolkitVitePlus formats, lints and checks the types of the sources with vp check and the
	// configuration of vite.config.ts, and runs the tests with vp test. ergon init renders
	// .prettierignore in place of biome.json.
	ToolkitVitePlus Toolkit = "vite-plus"
)

// toolkits are the toolkits.
var toolkits = []Toolkit{ToolkitBiome, ToolkitVitePlus}

// Validate returns an error that wraps [option.ErrInvalid] for a t other than biome and vite-plus.
func (t Toolkit) Validate() error {
	if !slices.Contains(toolkits, t) {
		return fmt.Errorf("%w: toolkit %q, which is neither biome nor vite-plus", option.ErrInvalid, t)
	}
	return nil
}
