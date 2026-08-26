package shellutil

import "strings"

// Quote wraps a string in single quotes for safe shell interpolation.
// It escapes any embedded single quotes using the standard '\'' pattern.
func Quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
