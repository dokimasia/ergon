// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package analysis_test

import (
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/lang/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
)

// today is the day of the cases of skipexpiry, in a location east of UTC, so a skip that expires
// on the day of UTC before it has passed.
var today = time.Date(2026, time.June, 1, 1, 0, 0, 0, time.FixedZone("UTC+2", 2*60*60))

func TestSkipExpiry(t *testing.T) {
	t.Parallel()

	t.Run("SkipExpiry", func(t *testing.T) {
		t.Parallel()

		t.Run("reports each skip of a test whose expiry has passed", func(t *testing.T) {
			t.Parallel()
			analysistest.Run(t, analysistest.TestData(), analysis.SkipExpiry(now, none), "skips")
		})

		t.Run("is named skipexpiry", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, analysis.SkipExpiry(now, none).Name, "skipexpiry", "the name of the analyzer")
		})
	})
}

// now returns today.
func now() time.Time {
	return today
}
