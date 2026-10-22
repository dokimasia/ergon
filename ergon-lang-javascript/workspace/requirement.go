// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workspace

import (
	"strings"

	"go.dokimi.dev/ergon/core/version"
)

// Protocol is the prefix of a requirement of pnpm on a package of its workspace, such as
// workspace:^1.2.0.
const Protocol = "workspace:"

// The shorthands of the workspace protocol, which pnpm replaces with a range of the version of the
// required package.
const (
	shorthandAny   = "*"
	shorthandCaret = "^"
	shorthandTilde = "~"
)

// The prefixes of a relative path after the workspace protocol.
const (
	pathCurrent = "./"
	pathParent  = "../"
)

// Shorthand reports whether text is a requirement of the workspace protocol that pnpm replaces with
// the version of the required package when it packs the dependent: workspace: alone,
// workspace:*, workspace:^, workspace:~, and a relative path after workspace:, such as
// workspace:../a.
func Shorthand(text string) bool {
	rest, ok := strings.CutPrefix(text, Protocol)
	if !ok {
		return false
	}
	switch rest {
	case "", shorthandAny, shorthandCaret, shorthandTilde:
		return true
	}
	return strings.HasPrefix(rest, pathCurrent) || strings.HasPrefix(rest, pathParent)
}

// expand returns the requirement that pnpm packs for text on a package at v, after the workspace
// protocol: workspace:<v> for workspace: alone, workspace:* and a relative path, workspace:^<v> for
// workspace:^, and workspace:~<v> for workspace:~. It returns any other text as it is.
func expand(text string, v version.Version) string {
	if !Shorthand(text) {
		return text
	}
	switch rest := strings.TrimPrefix(text, Protocol); rest {
	case shorthandCaret, shorthandTilde:
		return Protocol + rest + v.String()
	}
	return Protocol + v.String()
}
