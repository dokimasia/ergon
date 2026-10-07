// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package version_test

import (
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
	"go.dokimi.dev/ergon/core/version"
)

// identifier matches a pre-release identifier that Parse accepts: a word that starts with a letter,
// or a number without a leading zero.
const identifier = `([a-z][a-z0-9-]{0,4}|0|[1-9][0-9]{0,4})`

// versions generates the versions that Parse accepts, with and without a pre-release and build
// metadata.
var versions = prop.Composite(func(c *prop.Case) version.Version {
	component := prop.Integer[uint64](0, version.MaxComponent)
	v := version.Version{
		Major: c.Draw(component, "major"),
		Minor: c.Draw(component, "minor"),
		Patch: c.Draw(component, "patch"),
	}
	if c.Draw(prop.Boolean(), "has a pre-release") {
		v.Pre = c.Draw(prop.StringMatching(identifier+`(\.`+identifier+`){0,2}`), "pre-release")
	}
	if c.Draw(prop.Boolean(), "has build metadata") {
		v.Build = c.Draw(prop.StringMatching(`[0-9A-Za-z-]{1,6}(\.[0-9A-Za-z-]{1,6}){0,2}`), "build")
	}
	return v
})

// precedence are versions in ascending order of precedence: the example of Semantic Versioning
// 2.0.0, and the numeric components in order.
var precedence = []string{
	"1.0.0-1",
	"1.0.0-alpha",
	"1.0.0-alpha.1",
	"1.0.0-alpha.beta",
	"1.0.0-beta",
	"1.0.0-beta.2",
	"1.0.0-beta.11",
	"1.0.0-rc.1",
	"1.0.0-rc.2",
	"1.0.0",
	"1.0.1",
	"1.1.0",
	"2.0.0",
}

func TestVersion(t *testing.T) {
	t.Parallel()

	t.Run("Parse", func(t *testing.T) {
		t.Parallel()

		valid := []struct {
			name string
			give string
			want version.Version
		}{
			{name: "returns the components of a release", give: "1.4.0", want: version.Version{Major: 1, Minor: 4}},
			{name: "returns the zero version for 0.0.0", give: "0.0.0", want: version.Version{}},
			{
				name: "returns the pre-release without its hyphen",
				give: "2.0.0-rc.1",
				want: version.Version{Major: 2, Pre: "rc.1"},
			},
			{
				name: "returns the build metadata without its plus sign",
				give: "1.0.0+20261007",
				want: version.Version{Major: 1, Build: "20261007"},
			},
			{
				name: "returns the hyphens inside the identifiers",
				give: "1.0.0-alpha-1.x-y+meta-data.007",
				want: version.Version{Major: 1, Pre: "alpha-1.x-y", Build: "meta-data.007"},
			},
			{
				name: "returns the letters and digits at the ends of their ranges",
				give: "1.0.0-09azAZ+09azAZ",
				want: version.Version{Major: 1, Pre: "09azAZ", Build: "09azAZ"},
			},
			{
				name: "returns a component at MaxComponent",
				give: "9007199254740991.0.0",
				want: version.Version{Major: version.MaxComponent},
			},
		}
		for _, tt := range valid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := version.Parse(tt.give)
				assert.NoError(t, err, "Parse")
				assert.Equal(t, got, tt.want, "the version")
			})
		}

		invalid := []struct {
			name string
			give string
			want string
		}{
			{name: "returns ErrInvalid for two components", give: "1.4", want: "does not have the form"},
			{name: "returns ErrInvalid for a leading v", give: "v1.4.0", want: "is not a number"},
			{name: "returns ErrInvalid for an empty component", give: "1..0", want: "is not a number"},
			{name: "returns ErrInvalid for a component with a leading zero", give: "1.04.0", want: "leading zero"},
			{
				name: "returns ErrInvalid for a component above MaxComponent",
				give: "9007199254740992.0.0",
				want: "is above",
			},
			{
				name: "returns ErrInvalid for a component above the range of uint64",
				give: "1.0.99999999999999999999",
				want: "is above",
			},
			{name: "returns ErrInvalid for an empty pre-release", give: "1.0.0-", want: "empty identifier"},
			{name: "returns ErrInvalid for an empty identifier", give: "1.0.0-rc..1", want: "empty identifier"},
			{
				name: "returns ErrInvalid for a numeric identifier of the pre-release with a leading zero",
				give: "1.0.0-rc.01",
				want: "leading zero",
			},
			{name: "returns ErrInvalid for empty build metadata", give: "1.0.0+", want: "empty identifier"},
			{name: "returns ErrInvalid for an underscore", give: "1.0.0-rc_1", want: "the character '_'"},
			{name: "returns ErrInvalid for the byte before 0", give: "1.0.0-a/", want: "the character '/'"},
			{name: "returns ErrInvalid for the byte after 9", give: "1.0.0-a:", want: "the character ':'"},
			{name: "returns ErrInvalid for the byte before A", give: "1.0.0+a@", want: "the character '@'"},
			{name: "returns ErrInvalid for the byte after Z", give: "1.0.0+a[", want: "the character '['"},
			{name: "returns ErrInvalid for the byte before a", give: "1.0.0-a`", want: "the character '`'"},
			{name: "returns ErrInvalid for the byte after z", give: "1.0.0-a{", want: "the character '{'"},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := version.Parse(tt.give)
				assert.ErrorIs(t, err, version.ErrInvalid, "Parse")
				assert.Contains(t, err.Error(), tt.want, "the error")
			})
		}
	})

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		t.Run("returns what Parse reads back", func(t *testing.T) {
			t.Parallel()
			prop.RoundTrip(t, func(v version.Version) (string, error) { return v.String(), nil }, version.Parse,
				"Parse undoes String", prop.Using(versions))
		})
	})

	t.Run("MarshalText", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the text that String returns", func(t *testing.T) {
			t.Parallel()
			prop.Equal(t, func(v version.Version) string {
				// MarshalText returns no error, which the case below states.
				text, _ := v.MarshalText()
				return string(text)
			}, version.Version.String, "MarshalText and String", prop.Using(versions))
		})

		t.Run("returns no error", func(t *testing.T) {
			t.Parallel()
			prop.NoError(t, func(v version.Version) error {
				_, err := v.MarshalText()
				return err
			}, "MarshalText", prop.Using(versions))
		})
	})

	t.Run("UnmarshalText", func(t *testing.T) {
		t.Parallel()

		t.Run("reads back what MarshalText writes", func(t *testing.T) {
			t.Parallel()
			prop.RoundTrip(t, version.Version.MarshalText, func(text []byte) (version.Version, error) {
				var v version.Version
				err := v.UnmarshalText(text)
				return v, err
			}, "UnmarshalText undoes MarshalText", prop.Using(versions))
		})

		t.Run("returns ErrInvalid and leaves the version unchanged for text that Parse refuses", func(t *testing.T) {
			t.Parallel()
			v := version.Version{Major: 1}
			assert.ErrorIs(t, v.UnmarshalText([]byte("1.0")), version.ErrInvalid, "UnmarshalText of 1.0")
			assert.Equal(t, v, version.Version{Major: 1}, "the version after the error")
		})
	})

	t.Run("IsZero", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give version.Version
			want bool
		}{
			{name: "reports true for the zero value", give: version.Version{}, want: true},
			{name: "reports false for 0.0.1", give: version.Version{Patch: 1}, want: false},
			{name: "reports false for a pre-release of 0.0.0", give: version.Version{Pre: "rc.1"}, want: false},
			{name: "reports false for build metadata of 0.0.0", give: version.Version{Build: "7"}, want: false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.IsZero(), tt.want, "IsZero")
			})
		}
	})

	t.Run("Compare", func(t *testing.T) {
		t.Parallel()

		ordered := make([]version.Version, 0, len(precedence))
		for _, s := range precedence {
			v, err := version.Parse(s)
			assert.NoError(t, err, "Parse of "+s)
			ordered = append(ordered, v)
		}

		t.Run("returns -1 for each version before the next in precedence", func(t *testing.T) {
			t.Parallel()
			assert.Pairwise(t, ordered, func(earlier, later version.Version) bool {
				return earlier.Compare(later) == -1
			}, "the order of precedence")
		})

		t.Run("returns 1 for each version after the previous in precedence", func(t *testing.T) {
			t.Parallel()
			reversed := slices.Clone(ordered)
			slices.Reverse(reversed)
			assert.Pairwise(t, reversed, func(earlier, later version.Version) bool {
				return earlier.Compare(later) == 1
			}, "the reverse order of precedence")
		})

		t.Run("returns 0 for versions that differ in build metadata alone", func(t *testing.T) {
			t.Parallel()
			a := version.Version{Major: 1, Pre: "rc.1", Build: "a"}
			b := version.Version{Major: 1, Pre: "rc.1", Build: "b"}
			assert.Equal(t, a.Compare(b), 0, "Compare")
		})
	})

	t.Run("Bump", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give string
			bump version.Bump
			want string
		}{
			{name: "returns the version at none", give: "1.2.3+b", bump: version.BumpNone, want: "1.2.3+b"},
			{name: "increments the patch", give: "1.2.3", bump: version.BumpPatch, want: "1.2.4"},
			{name: "increments the minor and resets the patch", give: "1.2.3", bump: version.BumpMinor, want: "1.3.0"},
			{name: "increments the major and resets the rest", give: "1.2.3", bump: version.BumpMajor, want: "2.0.0"},
			{name: "releases 1.0.0 from 0.4.2 at major", give: "0.4.2", bump: version.BumpMajor, want: "1.0.0"},
			{name: "drops the build metadata", give: "1.2.3+b", bump: version.BumpPatch, want: "1.2.4"},
			{name: "releases a pre-release at patch", give: "1.2.3-rc.1", bump: version.BumpPatch, want: "1.2.3"},
			{
				name: "releases a pre-release whose patch is 0 at minor",
				give: "1.3.0-rc.1",
				bump: version.BumpMinor,
				want: "1.3.0",
			},
			{
				name: "increments the minor of a pre-release whose patch is not 0",
				give: "1.3.1-rc.1",
				bump: version.BumpMinor,
				want: "1.4.0",
			},
			{
				name: "releases a pre-release whose minor and patch are 0 at major",
				give: "2.0.0-rc.1",
				bump: version.BumpMajor,
				want: "2.0.0",
			},
			{
				name: "increments the major of a pre-release whose minor is not 0",
				give: "2.1.0-rc.1",
				bump: version.BumpMajor,
				want: "3.0.0",
			},
			{
				name: "increments the major of a pre-release whose patch is not 0",
				give: "2.0.1-rc.1",
				bump: version.BumpMajor,
				want: "3.0.0",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				v, err := version.Parse(tt.give)
				assert.NoError(t, err, "Parse")
				got, err := v.Bump(tt.bump)
				assert.NoError(t, err, "Bump")
				assert.Equal(t, got.String(), tt.want, "the bumped version")
			})
		}

		t.Run("returns ErrInvalid for a level that is not valid", func(t *testing.T) {
			t.Parallel()
			_, err := version.Version{Major: 1}.Bump("huge")
			assert.ErrorIs(t, err, version.ErrInvalid, "Bump")
		})

		overflows := []struct {
			name string
			give version.Version
			bump version.Bump
		}{
			{
				name: "returns ErrOverflow for a major at MaxComponent",
				give: version.Version{Major: version.MaxComponent},
				bump: version.BumpMajor,
			},
			{
				name: "returns ErrOverflow for a minor at MaxComponent",
				give: version.Version{Minor: version.MaxComponent},
				bump: version.BumpMinor,
			},
			{
				name: "returns ErrOverflow for a patch at MaxComponent",
				give: version.Version{Patch: version.MaxComponent},
				bump: version.BumpPatch,
			},
		}
		for _, tt := range overflows {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := tt.give.Bump(tt.bump)
				assert.ErrorIs(t, err, version.ErrOverflow, "Bump")
			})
		}
	})
}
