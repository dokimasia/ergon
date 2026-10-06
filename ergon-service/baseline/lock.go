// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"

	"go.dokimi.dev/ergon/core/language"
)

// ErrInvalidLock is the error for a lock that does not parse as one JSON object of the lock's
// fields, or whose file entries break the rules of [lockFile]. Its text states the remedy: the
// command new writes the lock again once the invalid lock is removed.
var ErrInvalidLock = errors.New("baseline: invalid .ergon/init.lock, so remove it and run new")

// lockPath is the path of the lock in the repository.
const lockPath = ".ergon/init.lock"

// lock is the content of .ergon/init.lock: the version of ergon that wrote it, the managed files
// with their digests, sorted by path, and the answers. The fields are in the order that packs
// their pointers first, which is also the order of their keys in the lock.
type lock struct {
	// Ergon is the version of ergon that wrote the lock.
	Ergon string `json:"ergon"`

	// Files are the managed files, sorted by path.
	Files []lockFile `json:"files"`

	// Answers are the answers that the managed files render.
	Answers language.Answers `json:"answers"`
}

// lockFile is the record of one managed file. Its path is valid as [fs.ValidPath] states, its
// producer is not empty, and each digest is 64 lowercase hexadecimal digits.
type lockFile struct {
	// Path is the path of the file in the repository.
	Path string `json:"path"`

	// Producer is the producer of the file, or of its first fragment.
	Producer string `json:"producer"`

	// Local is the digest of the local file that the file contains, or empty.
	Local string `json:"local,omitempty"`

	// SHA256 is the digest of the file as the command wrote it.
	SHA256 string `json:"sha256"`
}

// decodeLock parses data as a lock. It returns an error that wraps [ErrInvalidLock] for data that
// is not one JSON object of the lock's fields, and for a file entry that breaks the rules of
// [lockFile].
func decodeLock(data []byte) (lock, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var l lock
	if err := dec.Decode(&l); err != nil {
		return lock{}, fmt.Errorf("%w: %w", ErrInvalidLock, err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return lock{}, fmt.Errorf("%w: data after the object", ErrInvalidLock)
	}
	for _, f := range l.Files {
		if !fs.ValidPath(f.Path) || f.Producer == "" || !isDigest(f.SHA256) || (f.Local != "" && !isDigest(f.Local)) {
			return lock{}, fmt.Errorf("%w: entry %q", ErrInvalidLock, f.Path)
		}
	}
	return l, nil
}

// encode returns l as JSON indented by two spaces, with a final newline. A lock of strings,
// integers and lists encodes without an error.
func (l *lock) encode() []byte {
	b, _ := json.MarshalIndent(l, "", "  ")
	return append(b, '\n')
}

// digest returns the SHA-256 digest of b as 64 lowercase hexadecimal digits.
func digest(b []byte) string {
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
