// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"slices"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/service/baseline/options"
	"go.dokimi.dev/ergon/service/pin"
)

// Override is a pin that .ergon.yaml sets to another version than the baseline of the running
// ergon.
type Override struct {
	// Key is the key of the pin in .ergon.yaml, such as go.tools.golangci-lint.
	Key string

	// Version is the version that .ergon.yaml sets.
	Version string

	// Baseline is the version of the baseline of the running ergon.
	Baseline string
}

// Overrides returns the pins of .ergon.yaml whose version differs from the baseline of the running
// ergon, in the order of the producers and of their keys. An option that equals the value that the
// lock records follows the baseline, so it is no override. Overrides writes nothing.
//
// It returns an error that wraps [ErrNotInitialized] for a repository without a lock, the errors
// that [Repository.Options] returns for the lock and .ergon.yaml, and the error of [pin.Find] for a
// baseline with an invalid source tag.
func (r *Repository) Overrides() ([]Override, error) {
	l, err := r.readLock()
	if err != nil {
		return nil, err
	}
	units, next, err := r.units(&l.Answers)
	if err != nil {
		return nil, err
	}
	_, res, err := r.resolve(units, &next.Answers, &l)
	if err != nil {
		return nil, err
	}
	var overrides []Override
	for _, u := range units {
		c, ok := u.Producer.(language.Configurable)
		if !ok {
			continue
		}
		baseline, err := pin.Find(u.Name, c.Options())
		if err != nil {
			return nil, err
		}
		i := slices.IndexFunc(res.Sections, func(s options.Section) bool { return s.Name == u.Name })
		// The options of the section have the type of the baseline, so Find returns the same pins
		// without an error.
		stated, _ := pin.Find(u.Name, res.Sections[i].Options)
		for k, p := range stated {
			if p.Version != baseline[k].Version {
				overrides = append(overrides, Override{Key: p.Key, Version: p.Version, Baseline: baseline[k].Version})
			}
		}
	}
	return overrides, nil
}
