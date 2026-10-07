// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package lock_test

import (
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/spdx"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/service/baseline/lock"
)

// The digests of the cases: of the empty input, and of "abc", as FIPS 180-2 states them.
const (
	emptyDigest = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	abcDigest   = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
)

// path pins the path of the lock.
const path = ".ergon/init.lock"

// zeros is a digest of 64 zeros, valid in form.
var zeros = strings.Repeat("0", 64)

// encoded is the lock of the cases as Encode writes it.
const encoded = `{
  "ergon": "1.2.3",
  "files": [
    {
      "path": ".gitignore",
      "producer": "common",
      "sha256": "` + emptyDigest + `"
    },
    {
      "path": "go.txt",
      "producer": "go",
      "local": "` + abcDigest + `",
      "sha256": "` + abcDigest + `"
    }
  ],
  "settings": {
    "go.check": [
      "lint",
      "test"
    ],
    "go.fuzz.time": "30s"
  },
  "answers": {
    "name": "demo",
    "owner": "Dokimasia B.V.",
    "license": "MIT",
    "repository": "dokimasia/demo",
    "security-contact": "security@example.com",
    "languages": [
      "go"
    ],
    "year": 2026
  }
}
`

func TestLock(t *testing.T) {
	t.Parallel()

	t.Run("Path", func(t *testing.T) {
		t.Parallel()

		t.Run("is init.lock in the directory .ergon", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, lock.Path, path, "Path")
		})
	})

	t.Run("Decode", func(t *testing.T) {
		t.Parallel()

		t.Run("returns each field of the lock", func(t *testing.T) {
			t.Parallel()
			got, err := lock.Decode([]byte(encoded))
			assert.NoError(t, err, "Decode")
			assert.Equal(t, got, decoded(), "the lock")
		})

		t.Run("returns no error for the digits 0 to 9 and a to f in a digest", func(t *testing.T) {
			t.Parallel()
			digits := strings.Repeat("09af", 16)
			_, err := lock.Decode([]byte(with(`{"path": "x", "producer": "common", "local": "` + digits +
				`", "sha256": "` + digits + `"}`)))
			assert.NoError(t, err, "Decode")
		})

		tests := []struct {
			name string
			give string
		}{
			{name: "returns ErrInvalid for a lock that is not JSON", give: "not JSON"},
			{name: "returns ErrInvalid for an unknown field", give: `{"ergon": "1.2.3", "extra": 1}`},
			{name: "returns ErrInvalid for data after the object", give: with("") + " {}"},
			{
				name: "returns ErrInvalid for an entry with an invalid path",
				give: with(`{"path": "../x", "producer": "common", "sha256": "` + zeros + `"}`),
			},
			{
				name: "returns ErrInvalid for an entry without a producer",
				give: with(`{"path": "x", "producer": "", "sha256": "` + zeros + `"}`),
			},
			{
				name: "returns ErrInvalid for a digest of 63 digits",
				give: with(`{"path": "x", "producer": "common", "sha256": "` + zeros[1:] + `"}`),
			},
			{
				name: "returns ErrInvalid for a digest with the byte before 0",
				give: with(`{"path": "x", "producer": "common", "sha256": "/` + zeros[1:] + `"}`),
			},
			{
				name: "returns ErrInvalid for a digest with the byte after 9",
				give: with(`{"path": "x", "producer": "common", "sha256": ":` + zeros[1:] + `"}`),
			},
			{
				name: "returns ErrInvalid for a digest with the byte before a",
				give: with(`{"path": "x", "producer": "common", "sha256": "` + "`" + zeros[1:] + `"}`),
			},
			{
				name: "returns ErrInvalid for a digest with the byte after f",
				give: with(`{"path": "x", "producer": "common", "sha256": "g` + zeros[1:] + `"}`),
			},
			{
				name: "returns ErrInvalid for a digest with an uppercase digit",
				give: with(`{"path": "x", "producer": "common", "sha256": "A` + zeros[1:] + `"}`),
			},
			{
				name: "returns ErrInvalid for an invalid digest of the local file",
				give: with(`{"path": "x", "producer": "common", "local": "x", "sha256": "` + zeros + `"}`),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := lock.Decode([]byte(tt.give))
				assert.ErrorIs(t, err, lock.ErrInvalid, "Decode")
			})
		}
	})

	t.Run("Encode", func(t *testing.T) {
		t.Parallel()

		t.Run("returns JSON indented by two spaces with a final newline", func(t *testing.T) {
			t.Parallel()
			l := decoded()
			assert.Equal(t, string(l.Encode()), encoded, "the encoded lock")
		})
	})

	t.Run("File", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the entry of a recorded file", func(t *testing.T) {
			t.Parallel()
			l := decoded()
			got, ok := l.File("go.txt")
			assert.True(t, ok, "File of go.txt")
			assert.Equal(t, got, l.Files[1], "the entry of go.txt")
		})

		t.Run("reports false for a file that the lock does not record", func(t *testing.T) {
			t.Parallel()
			l := decoded()
			_, ok := l.File("other.txt")
			assert.False(t, ok, "File of other.txt")
		})
	})

	t.Run("Digest", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the digest of the empty input", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, lock.Digest(nil), emptyDigest, "Digest of nothing")
		})

		t.Run("returns the digest of abc", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, lock.Digest([]byte("abc")), abcDigest, "Digest of abc")
		})
	})
}

// decoded returns the lock of the cases, as Decode returns it for encoded.
func decoded() lock.Lock {
	return lock.Lock{
		Ergon: "1.2.3",
		Files: []lock.File{
			{Path: ".gitignore", Producer: "common", SHA256: emptyDigest},
			{Path: "go.txt", Producer: "go", Local: abcDigest, SHA256: abcDigest},
		},
		Settings: map[string]any{"go.check": []any{"lint", "test"}, "go.fuzz.time": "30s"},
		Answers: language.Answers{
			Name:            "demo",
			Owner:           "Dokimasia B.V.",
			License:         spdx.MIT,
			Repository:      "dokimasia/demo",
			SecurityContact: "security@example.com",
			Languages:       []workspace.Language{"go"},
			Year:            2026,
		},
	}
}

// with returns a lock without answers whose only entry of a file is entry, a JSON object.
func with(entry string) string {
	return `{"ergon": "1.2.3", "answers": {"languages": []}, "files": [` + entry + `]}`
}
