// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package producer

import (
	"io/fs"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
)

// lint is a tool that a constant states.
const lint = "lint@1.0.0"

// Producer is the producer of the cases.
type Producer struct{}

// Templates returns no templates.
func (Producer) Templates() fs.FS {
	return nil
}

// Options returns the baseline of the cases.
func (Producer) Options() language.Options {
	return (&Settings{
		Tools: Tools{
			Lint:  "example.com/lint/cmd/lint@v1.10.0", // the linter
			Audit: "audit@2.1.0",                       // the audit
			UV: option.UV{Binary: option.Binary{
				SHA256: map[option.Platform]string{
					option.LinuxAMD64:  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					option.DarwinARM64: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				},
				Version: "0.11.0",
			}},
			Check: Check{
				SHA256: map[option.Platform]string{
					option.LinuxAMD64: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
					option.LinuxARM64: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
				},
				Version: "1.1.0",
			},
		},
		Hooks:  "v6.1.0",
		Extra:  extra,
		Second: second,
		CI: option.CI[Actions]{
			Actions: Actions{Checkout: workflow.Action{
				Uses:    "actions/checkout",
				Commit:  "2222222222222222222222222222222222222222",
				Release: "v7.1.0",
			}},
			Timeout: 10,
		},
		Faults: Faults{
			Computed:   version(),
			Constant:   lint,
			Number:     3,
			Received:   <-received,
			Tools:      tools(),
			Mapped:     map[string]string{"lint": "lint@1.0.0"},
			Positional: Check{nil, "1.0.0"},
			Action:     workflow.Action{Uses: "actions/cache", Release: "v6.1.0"},
			Bare:       Check{Version: "1.0.0"},
			Digests:    Check{Version: "1.0.0", SHA256: digests()},
			Keyless: Check{
				SHA256:  map[option.Platform]string{"4444444444444444444444444444444444444444444444444444444444444444"},
				Version: "1.0.0",
			},
			Called: Check{SHA256: map[option.Platform]string{option.LinuxAMD64: digest()}, Version: "1.0.0"},
		},
	})
}
