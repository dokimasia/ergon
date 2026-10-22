// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/internal/app"
	"go.dokimi.dev/ergon/internal/cli"
	"go.dokimi.dev/ergon/service/baseline/baselinetest"
)

func TestRepository(t *testing.T) {
	t.Parallel()

	t.Run("Producers", func(t *testing.T) {
		t.Parallel()

		t.Run("returns producers whose sections go-cli decodes with every key", func(t *testing.T) {
			t.Parallel()
			producers := cli.Producers()
			names := make([]string, 0, len(producers))
			for _, p := range producers {
				conf, ok := p.Producer.(language.Configurable)
				assert.True(t, ok, "the producer "+p.Name+" has options")
				baselinetest.Keys(t, p.Name, conf)
				names = append(names, p.Name)
			}
			assert.Equal(t, names, []string{"common", "github", "license", "spec"}, "the sections of the producers")
		})
	})

	t.Run("Main", func(t *testing.T) {
		t.Parallel()

		t.Run("license check reads the repository of a parent directory with a lock", func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(licensed(t, nil), "docs", "adr")
			status, stdout, stderr := run(t, dir, "license", "check")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Matches(t, stdout, "^"+counts, "the standard output")
		})

		t.Run("tool run reads the repository of a parent directory with a lock", func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(javascriptRepository(t), "src")
			assert.NoError(t, os.Mkdir(dir, 0o755), "Mkdir of the directory")
			status, stdout, stderr := runWith(t, app.Register, dir, "tool", "run", "js.biome")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, biome+"\n", "the output of the tool")
		})

		t.Run("returns 1 for a working directory without a lock in it or in a parent", func(t *testing.T) {
			t.Parallel()
			status, stdout, stderr := run(t, t.TempDir(), "license", "check")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Empty(t, stdout, "the standard output")
			assert.Equal(t, stderr, "ergon: baseline: the repository has no lock, so run new first\n",
				"the standard error")
		})

		t.Run("returns 1 for a working directory that does not open", func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(t.TempDir(), "absent")
			status, _, stderr := run(t, dir, "tool", "run", "js.biome")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.HasPrefix(t, stderr, "ergon: cli: open the repository: ", "the standard error")
		})
	})
}
