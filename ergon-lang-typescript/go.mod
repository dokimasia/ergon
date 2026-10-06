module go.dokimi.dev/ergon/lang/typescript

go 1.27.1

require (
	go.dokimi.dev/assert v0.0.0-20261006203413-f36a960fd058
	go.dokimi.dev/ergon/core v0.0.0
	go.dokimi.dev/ergon/lang/javascript v0.0.0
)

replace (
	go.dokimi.dev/ergon/core => ../ergon-core
	go.dokimi.dev/ergon/lang/javascript => ../ergon-lang-javascript
)
