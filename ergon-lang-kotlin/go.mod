module go.dokimi.dev/ergon/lang/kotlin

go 1.27.1

require (
	go.dokimi.dev/assert v0.0.0-20261005091421-c73aed48ae5c
	go.dokimi.dev/ergon/core v0.0.0
	go.dokimi.dev/ergon/lang/java v0.0.0
)

replace (
	go.dokimi.dev/ergon/core => ../ergon-core
	go.dokimi.dev/ergon/lang/java => ../ergon-lang-java
)
