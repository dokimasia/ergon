// Package other has a function New that is not errors.New.
package other

import "errors"

// New returns an error with the text s.
func New(s string) error {
	return errors.New(s)
}
