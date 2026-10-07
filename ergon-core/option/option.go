// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package option

import "errors"

// ErrInvalid is the error of a Validate method for a value that its option does not accept. The
// text names the value and the rule that it breaks. The caller adds the key of the option.
var ErrInvalid = errors.New("option: invalid value")

// ErrNoAsset is the error of [Release.Asset] for a platform whose asset a release does not
// publish.
var ErrNoAsset = errors.New("option: no asset for the platform")

// The keys of the struct tags of an options struct. Each states a fact about one field.
const (
	// DocTag is the key of the meaning of the field. ergon init writes it as the comment above the
	// key of the field in .ergon.yaml.
	DocTag = "doc"

	// AnswerTag is the key of the answer of ergon init that the field states, by the key of the
	// answer in the lock, such as owner. The field takes the answer's value: ergon init writes it,
	// and records no baseline value for it.
	AnswerTag = "answer"

	// ProgramTag is the key of the name of the program of a tool, where it differs from the last
	// element of the tool's package: tsc of the npm package typescript.
	ProgramTag = "program"

	// RunTag is the key of the environment in which a tool of the kind [PyPI] runs. Its one value
	// is [Project].
	RunTag = "run"

	// ClassifierTag is the key of the classifier of the jar of a tool of the kind [Maven], such as
	// all for the jar that contains its dependencies.
	ClassifierTag = "classifier"
)

// Project is the value of [RunTag] for a tool that runs in the environment of the project, as a
// type checker or a test runner does. A tool without it runs in an environment of its own.
const Project = "project"
