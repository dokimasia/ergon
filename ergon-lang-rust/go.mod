// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

module go.dokimi.dev/ergon/lang/rust

go 1.27.3

require (
	github.com/bmatcuk/doublestar/v4 v4.10.4
	github.com/pelletier/go-toml/v2 v2.4.3
	go.dokimi.dev/assert v0.20.0
	go.dokimi.dev/ergon/core v0.19.0
	go.dokimi.dev/ergon/service v0.24.0
)

require (
	github.com/creachadair/tomledit v0.0.29 // indirect
	go.stealthscale.io/cli v0.8.1 // indirect
	go.yaml.in/yaml/v3 v3.0.6 // indirect
	golang.org/x/mod v0.42.0 // indirect
)

replace (
	go.dokimi.dev/ergon/core => ../ergon-core
	go.dokimi.dev/ergon/service => ../ergon-service
)
