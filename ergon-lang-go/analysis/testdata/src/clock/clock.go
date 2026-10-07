// Package clock is the package of the cases of errorprefix.
package clock

import (
	"errors"
	stderrors "errors"

	"other"
)

// text is a constant text without the name of the package.
const text = "constant text"

// ErrZero has the name of the package and a colon.
var ErrZero = errors.New("clock: instant is zero")

// ErrPatch has the name of the package and the qualifier patch.
var ErrPatch = errors.New("clock.patch: malformed")

// ErrBare has no name of the package.
var ErrBare = errors.New("instant is zero") // want `the text "instant is zero" of errors.New does not start with "clock:", the name of its package`

// ErrAliased is a call of errors.New under another name.
var ErrAliased = stderrors.New("aliased") // want `the text "aliased" of errors.New does not start with "clock:", the name of its package`

// ErrConstant is a call of errors.New with a constant.
var ErrConstant = errors.New(text) // want `the text "constant text" of errors.New does not start with "clock:", the name of its package`

// ErrOther is a call of a function New of another package.
var ErrOther = other.New("other text")

// Dynamic returns an error whose text is no constant.
func Dynamic(s string) error {
	return errors.New(s)
}
