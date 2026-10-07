package skips

import "testing"

// expired is a constant message of a skip that expired.
const expired = "flaky, expires 2026-03-01"

func TestExpired(t *testing.T) {
	t.Skip("TODO: fix the order, expires 2026-01-01") // want `the skip expired on 2026-01-01, so fix the test and remove the skip`
}

func TestSkipf(t *testing.T) {
	t.Skipf("expires 2025-12-31, attempt %d", 1) // want `the skip expired on 2025-12-31, so fix the test and remove the skip`
}

func TestConstant(t *testing.T) {
	t.Skip(expired) // want `the skip expired on 2026-03-01, so fix the test and remove the skip`
}

func TestToday(t *testing.T) {
	t.Skip("expires 2026-06-01")
}

func TestFuture(t *testing.T) {
	t.Skip("expires 2099-12-31")
}

func TestUndated(t *testing.T) {
	t.Skip("flaky")
}

func TestDynamic(t *testing.T) {
	message := "expires 2020-01-01"
	t.Skip(message)
}

func TestWithout(t *testing.T) {
	t.Skip()
}

func TestNow(t *testing.T) {
	t.SkipNow()
}

func TestLog(t *testing.T) {
	t.Log("expires 2020-01-01")
}

func TestFunction(t *testing.T) {
	skip(t, "expires 2020-01-01")
}

// skip skips t with message, through a function that is no method Skip.
func skip(t *testing.T, message string) {
	t.Helper()
	t.Skip(message)
}
