// Package skips is the package of the cases of skipexpiry.
package skips

// Runner has a method Skip outside a test file.
type Runner struct{}

// Skip does nothing.
func (Runner) Skip(...any) {}

// Use calls Skip outside a test file, which skipexpiry does not read.
func Use() {
	Runner{}.Skip("expires 2020-01-01")
}
