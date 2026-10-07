// Package legacy breaks both rules, so a skip that excludes it changes what the analyzers report.
package legacy

import "errors"

// ErrBare has no name of the package.
var ErrBare = errors.New("no name of the package")
