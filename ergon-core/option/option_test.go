// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package option_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
)

// The spellings of the keys of the struct tags and of the value of the tag run, which the struct
// tags of every producer write literally.
const (
	docTag        = "doc"
	answerTag     = "answer"
	programTag    = "program"
	runTag        = "run"
	classifierTag = "classifier"
	sourceTag     = "source"
	project       = "project"
)

// prefix is the start of the text of every error of the package.
const prefix = "option: "

func TestOption(t *testing.T) {
	t.Parallel()

	t.Run("ErrInvalid", func(t *testing.T) {
		t.Parallel()

		t.Run("starts its text with the name of the package", func(t *testing.T) {
			t.Parallel()
			assert.HasPrefix(t, option.ErrInvalid.Error(), prefix, "the text of ErrInvalid")
		})
	})

	t.Run("ErrNoAsset", func(t *testing.T) {
		t.Parallel()

		t.Run("starts its text with the name of the package", func(t *testing.T) {
			t.Parallel()
			assert.HasPrefix(t, option.ErrNoAsset.Error(), prefix, "the text of ErrNoAsset")
		})
	})

	t.Run("DocTag", func(t *testing.T) {
		t.Parallel()

		t.Run("is the key doc", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, option.DocTag, docTag, "DocTag")
		})
	})

	t.Run("AnswerTag", func(t *testing.T) {
		t.Parallel()

		t.Run("is the key answer", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, option.AnswerTag, answerTag, "AnswerTag")
		})
	})

	t.Run("ProgramTag", func(t *testing.T) {
		t.Parallel()

		t.Run("is the key program", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, option.ProgramTag, programTag, "ProgramTag")
		})
	})

	t.Run("RunTag", func(t *testing.T) {
		t.Parallel()

		t.Run("is the key run", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, option.RunTag, runTag, "RunTag")
		})
	})

	t.Run("ClassifierTag", func(t *testing.T) {
		t.Parallel()

		t.Run("is the key classifier", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, option.ClassifierTag, classifierTag, "ClassifierTag")
		})
	})

	t.Run("SourceTag", func(t *testing.T) {
		t.Parallel()

		t.Run("is the key source", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, option.SourceTag, sourceTag, "SourceTag")
		})
	})

	t.Run("Project", func(t *testing.T) {
		t.Parallel()

		t.Run("is the value project", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, option.Project, project, "Project")
		})
	})
}
