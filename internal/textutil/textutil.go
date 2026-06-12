// Package textutil provides small, dependency-free string utilities
// that need to be unit-testable without pulling in CGO-heavy packages
// (e.g. Fyne).
package textutil

import "unicode/utf8"

// Truncate returns s shortened to at most max visible characters (runes),
// with an ellipsis appended when truncation occurred.
//
// The naive byte-based version of this function sliced s in the middle of
// a multi-byte UTF-8 sequence when given Chinese (or any other non-ASCII)
// input, producing invalid UTF-8 strings that displayed as garbled "tofu"
// boxes in Fyne labels. Counting characters via utf8.RuneCountInString
// avoids that and always returns a valid UTF-8 string.
//
// If max <= 3, no ellipsis is appended because there is no room for it.
func Truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}
