// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package producer

// extra are the tools that a package-level variable states.
var extra = Extra{Format: "format@1.1.0"}

// unset is a variable without a value.
var unset Extra

// first and second are two variables of one declaration.
var first, second = Extra{Format: "first@1.0.0"}, Extra{Format: "second@1.2.0"}

// received is the value of a receive.
var received = "received@1.0.0"
