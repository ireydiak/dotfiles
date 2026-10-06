package exec

import "strings"

// Quote wraps s in single quotes for /bin/zsh -c, escaping embedded single quotes.
func Quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
