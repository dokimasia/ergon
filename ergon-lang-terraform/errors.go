// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package terraform

import "errors"

// ErrRepository is the error of [Register] for a [Repository] without one of its functions.
var ErrRepository = errors.New("terraform: incomplete repository")
