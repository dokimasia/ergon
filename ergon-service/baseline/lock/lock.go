// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package lock

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"slices"

	"go.dokimi.dev/ergon/core/language"
)

// Path is the path of the lock in a repository.
const Path = ".ergon/init.lock"

// ErrInvalid is the error for a lock that does not parse as one JSON object of the lock's fields,
// or whose entries of files break the rules of [File]. Its text states the remedy: ergon init new
// writes the lock again once the invalid lock is removed.
var ErrInvalid = errors.New("lock: invalid .ergon/init.lock, so remove it and run ergon init new")

// Lock is the content of .ergon/init.lock. The fields are in the order of their keys in the file.
type Lock struct {
	// Ergon is the release of ergon that wrote the lock.
	Ergon string `json:"ergon"`

	// Files are the managed files, sorted by path.
	Files []File `json:"files"`

	// Settings are the baseline value of each option that ergon init wrote into .ergon.yaml, by its
	// key, such as go.fuzz.time. A lock without the field records no option.
	Settings map[string]any `json:"settings"`

	// Answers are the answers that the managed files render.
	Answers language.Answers `json:"answers"`
}

// File is the record of one managed file. Its path is valid as [fs.ValidPath] states, its producer
// is not empty, and each digest is 64 lowercase hexadecimal digits.
type File struct {
	// Path is the path of the file in the repository.
	Path string `json:"path"`

	// Producer is the producer of the file, or of its first fragment.
	Producer string `json:"producer"`

	// Local is the digest of the local file that the file contains, or empty.
	Local string `json:"local,omitempty"`

	// SHA256 is the digest of the file as ergon init wrote it.
	SHA256 string `json:"sha256"`
}

// Decode parses data as a lock. It returns an error that wraps [ErrInvalid] for data that is not
// one JSON object of the lock's fields, and for an entry of a file that breaks the rules of [File].
func Decode(data []byte) (Lock, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var l Lock
	if err := dec.Decode(&l); err != nil {
		return Lock{}, fmt.Errorf("%w: %w", ErrInvalid, err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return Lock{}, fmt.Errorf("%w: data after the object", ErrInvalid)
	}
	for _, f := range l.Files {
		if !fs.ValidPath(f.Path) || f.Producer == "" || !isDigest(f.SHA256) || (f.Local != "" && !isDigest(f.Local)) {
			return Lock{}, fmt.Errorf("%w: the entry of %q", ErrInvalid, f.Path)
		}
	}
	return l, nil
}

// Encode returns l as JSON indented by two spaces, with a final newline. A lock of strings,
// integers, lists, and the values that a scalar, a list or a mapping of YAML decodes into encodes
// without an error.
func (l *Lock) Encode() []byte {
	b, _ := json.MarshalIndent(l, "", "  ")
	return append(b, '\n')
}

// File returns the entry of the file path, and reports whether l has one.
func (l *Lock) File(path string) (File, bool) {
	i := slices.IndexFunc(l.Files, func(f File) bool { return f.Path == path })
	if i < 0 {
		return File{}, false
	}
	return l.Files[i], true
}

// Digest returns the SHA-256 digest of b as 64 lowercase hexadecimal digits.
func Digest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// isDigest reports whether s is 64 lowercase hexadecimal digits.
func isDigest(s string) bool {
	if len(s) != sha256.Size*2 {
		return false
	}
	for i := range len(s) {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
