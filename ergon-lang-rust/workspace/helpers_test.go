// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workspace_test

import (
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/lang/rust/workspace"
)

// spanAfter returns the range of the bytes of the first literal in data after the marker, which
// data has once, and stops the test when data does not have either.
func spanAfter(t *testing.T, data, marker, literal string) workspace.Span {
	t.Helper()
	assert.Equal(t, strings.Count(data, marker), 1, "the occurrences of the marker "+marker)
	from := strings.Index(data, marker) + len(marker)
	assert.Contains(t, data[from:], literal, "the text after the marker "+marker)
	at := from + strings.Index(data[from:], literal)
	return workspace.Span{Start: at, End: at + len(literal)}
}

// parse returns the manifest of data in the file, and stops the test at an error of ParseManifest.
func parse(t *testing.T, file, data string) *workspace.Manifest {
	t.Helper()
	m, err := workspace.ParseManifest(file, []byte(data))
	assert.NoError(t, err, "ParseManifest of "+file)
	return m
}
