// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/service/baseline"
)

// zeros is a digest of 64 zeros, valid in form.
var zeros = strings.Repeat("0", 64)

func TestLock(t *testing.T) {
	t.Parallel()

	t.Run("Repository", func(t *testing.T) {
		t.Parallel()

		t.Run("New", func(t *testing.T) {
			t.Parallel()

			t.Run("writes the version, the answers and the digest of each managed file", func(t *testing.T) {
				t.Parallel()
				_, root := initialized(t)
				want := `{
  "ergon": "1.2.3",
  "files": [
    {
      "path": ".github/ci.yml",
      "producer": "common",
      "sha256": "` + sum(workflowContent) + `"
    },
    {
      "path": ".gitignore",
      "producer": "common",
      "sha256": "` + sum("# common\nalpha/\n") + `"
    },
    {
      "path": "LICENSE",
      "producer": "common",
      "sha256": "` + sum("Copyright Dokimasia B.V.\n") + `"
    },
    {
      "path": "alpha/settings.txt",
      "producer": "alpha",
      "sha256": "` + sum("alpha\n") + `"
    }
  ],
  "answers": {
    "name": "demo",
    "owner": "Dokimasia B.V.",
    "license": "MIT",
    "repository": "dokimasia/demo",
    "security-contact": "security@example.com",
    "languages": [
      "alpha"
    ],
    "year": 2026
  }
}
`
				assert.Equal(t, content(t, root, lockPath), want, "the lock")
			})

			t.Run("writes an empty list of files for a rendering without managed files", func(t *testing.T) {
				t.Parallel()
				root := directory(t)
				r, err := baseline.Open(root, catalog(t), version)
				assert.NoError(t, err, "Open")
				a := answers()
				a.Languages = nil
				_, err = r.New(a, baseline.Options{})
				assert.NoError(t, err, "New")
				assert.Contains(t, content(t, root, lockPath), `"files": []`, "the lock")
			})
		})

		t.Run("Check", func(t *testing.T) {
			t.Parallel()

			tests := []struct {
				name string
				lock string
			}{
				{name: "returns ErrInvalidLock for a lock that is not JSON", lock: "not JSON"},
				{name: "returns ErrInvalidLock for an unknown field", lock: `{"ergon": "1.2.3", "extra": 1}`},
				{name: "returns ErrInvalidLock for data after the object", lock: lockWith("") + " {}"},
				{
					name: "returns ErrInvalidLock for an entry with an invalid path",
					lock: lockWith(`{"path": "../x", "producer": "common", "sha256": "` + zeros + `"}`),
				},
				{
					name: "returns ErrInvalidLock for an entry without a producer",
					lock: lockWith(`{"path": "x", "producer": "", "sha256": "` + zeros + `"}`),
				},
				{
					name: "returns ErrInvalidLock for a digest of 63 digits",
					lock: lockWith(`{"path": "x", "producer": "common", "sha256": "` + zeros[1:] + `"}`),
				},
				{
					name: "returns ErrInvalidLock for a digest with the byte before 0",
					lock: lockWith(`{"path": "x", "producer": "common", "sha256": "/` + zeros[1:] + `"}`),
				},
				{
					name: "returns ErrInvalidLock for a digest with the byte after 9",
					lock: lockWith(`{"path": "x", "producer": "common", "sha256": ":` + zeros[1:] + `"}`),
				},
				{
					name: "returns ErrInvalidLock for a digest with the byte before a",
					lock: lockWith(`{"path": "x", "producer": "common", "sha256": "` + "`" + zeros[1:] + `"}`),
				},
				{
					name: "returns ErrInvalidLock for a digest with the byte after f",
					lock: lockWith(`{"path": "x", "producer": "common", "sha256": "g` + zeros[1:] + `"}`),
				},
				{
					name: "returns ErrInvalidLock for a digest with an uppercase digit",
					lock: lockWith(`{"path": "x", "producer": "common", "sha256": "A` + zeros[1:] + `"}`),
				},
				{
					name: "returns ErrInvalidLock for an invalid digest of the local file",
					lock: lockWith(`{"path": "x", "producer": "common", "local": "x", "sha256": "` + zeros + `"}`),
				},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					root := directory(t)
					put(t, root, lockPath, tt.lock)
					_, err := repository(t, root).Check()
					assert.ErrorIs(t, err, baseline.ErrInvalidLock, "Check")
				})
			}

			t.Run("returns no error for the digits 0 to 9 and a to f in a digest", func(t *testing.T) {
				t.Parallel()
				root := directory(t)
				digits := strings.Repeat("09af", 16)
				put(t, root, lockPath, lockWith(`{"path": "x", "producer": "common", "local": "`+digits+
					`", "sha256": "`+digits+`"}`))
				_, err := repository(t, root).Check()
				assert.NoError(t, err, "Check")
			})
		})
	})
}

// sum returns the SHA-256 digest of s as 64 lowercase hexadecimal digits.
func sum(s string) string {
	digest := sha256.Sum256([]byte(s))
	return hex.EncodeToString(digest[:])
}

// lockWith returns a lock without answers whose only file entry is entry, a JSON object.
func lockWith(entry string) string {
	return `{"ergon": "1.2.3", "answers": {"languages": []}, "files": [` + entry + `]}`
}
