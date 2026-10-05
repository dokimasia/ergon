// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workspace

// Language is the registered name of a language, such as "go" or "kotlin". The zero value is not
// a valid name.
type Language string

// Valid reports whether l is a lowercase ASCII letter followed by lowercase ASCII letters and
// digits.
func (l Language) Valid() bool {
	return valid(string(l))
}

// Toolchain is the registered name of a build toolchain, such as "go", "jvm" or "js". The zero
// value is not a valid name.
type Toolchain string

// Valid reports whether t is a lowercase ASCII letter followed by lowercase ASCII letters and
// digits.
func (t Toolchain) Valid() bool {
	return valid(string(t))
}

// valid reports whether name is a lowercase ASCII letter followed by lowercase ASCII letters and
// digits.
func valid(name string) bool {
	if name == "" {
		return false
	}
	for i := range len(name) {
		c := name[i]
		letter := 'a' <= c && c <= 'z'
		digit := '0' <= c && c <= '9'
		if !letter && (i == 0 || !digit) {
			return false
		}
	}
	return true
}
