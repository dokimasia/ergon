// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

module go.dokimi.dev/ergon/lang/typescript

go 1.27.3

require (
	go.dokimi.dev/assert v0.3.0
	go.dokimi.dev/ergon/core v0.14.0
	go.dokimi.dev/ergon/lang/javascript v0.4.8
	go.dokimi.dev/ergon/service v0.18.0
)

require (
	github.com/creachadair/tomledit v0.0.29 // indirect
	github.com/pelletier/go-toml/v2 v2.4.3 // indirect
	go.stealthscale.io/cli v0.6.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/mod v0.41.0 // indirect
)

replace (
	go.dokimi.dev/ergon/core => ../ergon-core
	go.dokimi.dev/ergon/lang/javascript => ../ergon-lang-javascript
	go.dokimi.dev/ergon/service => ../ergon-service
)
