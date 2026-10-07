// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package option_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
)

func TestRun(t *testing.T) {
	t.Parallel()

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nil for arguments on one line each", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, option.Run{Args: []string{"-count=1", "-p=1"}}.Validate(), "Validate")
		})

		t.Run("returns ErrInvalid for an argument that spans lines", func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, option.Run{Args: []string{"-count=1\n-p=1"}}.Validate(), option.ErrInvalid, "Validate")
		})
	})

	t.Run("Fuzz", func(t *testing.T) {
		t.Parallel()

		t.Run("Validate", func(t *testing.T) {
			t.Parallel()

			valid := []struct {
				name string
				give option.Fuzz
			}{
				{
					name: "returns nil for a duration",
					give: option.Fuzz{Match: ".", Time: "30s", Args: []string{"-fuzzminimizetime=5s"}},
				},
				{name: "returns nil for a number of runs", give: option.Fuzz{Match: "^FuzzDecode$", Time: "1000x"}},
			}
			for _, tt := range valid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.NoError(t, tt.give.Validate(), "Validate")
				})
			}

			invalid := []struct {
				name string
				give option.Fuzz
			}{
				{
					name: "returns ErrInvalid for a match that does not compile",
					give: option.Fuzz{Match: "(", Time: "30s"},
				},
				{
					name: "returns ErrInvalid for a match with a single quote",
					give: option.Fuzz{Match: "a'b", Time: "30s"},
				},
				{
					name: "returns ErrInvalid for a match that spans lines",
					give: option.Fuzz{Match: "a\nb", Time: "30s"},
				},
				{name: "returns ErrInvalid for a time of 0s", give: option.Fuzz{Match: ".", Time: "0s"}},
				{name: "returns ErrInvalid for a time of 0 runs", give: option.Fuzz{Match: ".", Time: "0x"}},
				{name: "returns ErrInvalid for a time without a unit", give: option.Fuzz{Match: ".", Time: "30"}},
				{
					name: "returns ErrInvalid for a count of runs that is not a number",
					give: option.Fuzz{Match: ".", Time: "ax"},
				},
				{
					name: "returns ErrInvalid for an argument that spans lines",
					give: option.Fuzz{Match: ".", Time: "30s", Args: []string{"a\nb"}},
				},
			}
			for _, tt := range invalid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.ErrorIs(t, tt.give.Validate(), option.ErrInvalid, "Validate")
				})
			}
		})
	})

	t.Run("Bench", func(t *testing.T) {
		t.Parallel()

		t.Run("Validate", func(t *testing.T) {
			t.Parallel()

			t.Run("returns nil for the options of the baseline", func(t *testing.T) {
				t.Parallel()
				b := option.Bench{Match: ".", Time: "1s", Args: []string{"-benchmem"}, Count: 6}
				assert.NoError(t, b.Validate(), "Validate")
			})

			invalid := []struct {
				name string
				give option.Bench
			}{
				{
					name: "returns ErrInvalid for a match that does not compile",
					give: option.Bench{Match: "[", Time: "1s", Count: 1},
				},
				{name: "returns ErrInvalid for a negative time", give: option.Bench{Match: ".", Time: "-1s", Count: 1}},
				{
					name: "returns ErrInvalid for an argument that spans lines",
					give: option.Bench{Match: ".", Time: "1s", Args: []string{"\r"}, Count: 1},
				},
				{name: "returns ErrInvalid for a count of 0", give: option.Bench{Match: ".", Time: "1s"}},
			}
			for _, tt := range invalid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.ErrorIs(t, tt.give.Validate(), option.ErrInvalid, "Validate")
				})
			}
		})
	})

	t.Run("Mutate", func(t *testing.T) {
		t.Parallel()

		t.Run("Validate", func(t *testing.T) {
			t.Parallel()

			valid := []struct {
				name string
				give option.Mutate
			}{
				{name: "returns nil for a timeout of 0s", give: option.Mutate{Timeout: "0s", Workers: 1}},
				{
					name: "returns nil for a limit and arguments",
					give: option.Mutate{Timeout: "30m", Args: []string{"-confirm"}, Workers: 2},
				},
			}
			for _, tt := range valid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.NoError(t, tt.give.Validate(), "Validate")
				})
			}

			invalid := []struct {
				name string
				give option.Mutate
			}{
				{
					name: "returns ErrInvalid for a timeout that is not a duration",
					give: option.Mutate{Timeout: "soon", Workers: 1},
				},
				{name: "returns ErrInvalid for a negative timeout", give: option.Mutate{Timeout: "-1m", Workers: 1}},
				{
					name: "returns ErrInvalid for an argument that spans lines",
					give: option.Mutate{Timeout: "0s", Args: []string{"a\nb"}, Workers: 1},
				},
				{name: "returns ErrInvalid for 0 workers", give: option.Mutate{Timeout: "0s"}},
			}
			for _, tt := range invalid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.ErrorIs(t, tt.give.Validate(), option.ErrInvalid, "Validate")
				})
			}
		})
	})
}
