// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package option

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Run is the options of a step that runs a command with arguments, such as test, race and
// generate.
type Run struct {
	// Args are the arguments that follow the step's command, each on one line.
	Args []string `yaml:"args"`
}

// Validate returns an error that wraps [ErrInvalid] for an argument that spans lines.
func (r Run) Validate() error {
	return lines("args", r.Args)
}

// Fuzz is the options of the step fuzz: which fuzz targets run, and for how long each.
type Fuzz struct {
	// Match is a regular expression of the names of the fuzz targets that run, on one line and
	// without a single quote, which the Makefile writes in single quotes.
	Match string `yaml:"match"`

	// Time is the time of each fuzz target: a positive duration, such as 30s, or a positive number
	// of runs, such as 1000x.
	Time string `yaml:"time"`

	// Args are the arguments that follow the step's command, each on one line.
	Args []string `yaml:"args"`
}

// Validate returns an error that wraps [ErrInvalid] for a Match that does not compile as a
// regular expression of Go, spans lines or has a single quote, a Time that is neither a positive
// duration nor a positive number of runs, and an argument that spans lines.
func (f Fuzz) Validate() error {
	if err := match(f.Match); err != nil {
		return err
	}
	if err := positiveTime(f.Time); err != nil {
		return err
	}
	return lines("args", f.Args)
}

// Bench is the options of the step bench: which benchmarks run, for how long, and how often.
type Bench struct {
	// Match is a regular expression of the names of the benchmarks that run, on one line and
	// without a single quote, which the Makefile writes in single quotes.
	Match string `yaml:"match"`

	// Time is the time of each run of a benchmark: a positive duration, such as 1s, or a positive
	// number of iterations, such as 100x.
	Time string `yaml:"time"`

	// Args are the arguments that follow the step's command, each on one line.
	Args []string `yaml:"args"`

	// Count is the number of runs of each benchmark, at least 1.
	Count int `yaml:"count"`
}

// Validate returns an error that wraps [ErrInvalid] for a Match that does not compile as a
// regular expression of Go, spans lines or has a single quote, a Time that is neither a positive
// duration nor a positive number of iterations, an argument that spans lines, and a Count below
// 1.
func (b Bench) Validate() error {
	if err := match(b.Match); err != nil {
		return err
	}
	if err := positiveTime(b.Time); err != nil {
		return err
	}
	if err := lines("args", b.Args); err != nil {
		return err
	}
	if b.Count < 1 {
		return fmt.Errorf("%w: count %d, which is less than 1", ErrInvalid, b.Count)
	}
	return nil
}

// Mutate is the options of the step mutate: the limit of the run, and how many mutants run at
// once.
type Mutate struct {
	// Timeout is the limit of the whole run, as a duration of at least 0s. 0s sets no limit.
	Timeout string `yaml:"timeout"`

	// Args are the arguments that follow the step's command, each on one line.
	Args []string `yaml:"args"`

	// Workers is the number of mutants that run at once, at least 1.
	Workers int `yaml:"workers"`
}

// Validate returns an error that wraps [ErrInvalid] for a Timeout that is not a duration of at
// least 0s, an argument that spans lines, and a Workers below 1.
func (m Mutate) Validate() error {
	if d, err := time.ParseDuration(m.Timeout); err != nil || d < 0 {
		return fmt.Errorf("%w: timeout %q, which is not a duration of at least 0s", ErrInvalid, m.Timeout)
	}
	if err := lines("args", m.Args); err != nil {
		return err
	}
	if m.Workers < 1 {
		return fmt.Errorf("%w: workers %d, which is less than 1", ErrInvalid, m.Workers)
	}
	return nil
}

// lines returns an error that wraps [ErrInvalid] for the first of values that spans lines, which
// the error names as an element of key.
func lines(key string, values []string) error {
	for _, v := range values {
		if strings.ContainsAny(v, "\r\n") {
			return fmt.Errorf("%w: %s %q, which spans lines", ErrInvalid, key, v)
		}
	}
	return nil
}

// match returns an error that wraps [ErrInvalid] for a pattern that does not compile as a
// regular expression of Go, spans lines or has a single quote.
func match(pattern string) error {
	if _, err := regexp.Compile(pattern); err != nil || strings.ContainsAny(pattern, "'\r\n") {
		return fmt.Errorf("%w: match %q, which is not a regular expression on one line without a single quote",
			ErrInvalid, pattern)
	}
	return nil
}

// positiveTime returns an error that wraps [ErrInvalid] for a time that go test does not accept
// for -fuzztime and -benchtime: neither a positive duration, such as 30s, nor a positive number
// of iterations, such as 100x.
func positiveTime(t string) error {
	if count, ok := strings.CutSuffix(t, "x"); ok {
		if n, err := strconv.Atoi(count); err == nil && n > 0 {
			return nil
		}
	} else if d, err := time.ParseDuration(t); err == nil && d > 0 {
		return nil
	}
	return fmt.Errorf("%w: time %q, which is neither a positive duration nor a positive count such as 100x",
		ErrInvalid, t)
}
