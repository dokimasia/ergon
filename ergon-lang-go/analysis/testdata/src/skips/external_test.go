package skips_test

import "testing"

func TestExternal(t *testing.T) {
	t.Skip("expires 2026-05-31") // want `the skip expired on 2026-05-31, so fix the test and remove the skip`
}
