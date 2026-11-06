package data

import (
	"strings"
)

// joinArgs renders a parenthesised placeholder list for an IN (...) clause and
// returns the matching argument slice. An empty input yields "(NULL)" so the
// surrounding query stays syntactically valid and matches no rows.
func joinArgs[T any](items []T) (string, []any) {
	if len(items) == 0 {
		return "(NULL)", nil
	}
	var b strings.Builder
	args := make([]any, 0, len(items))
	b.WriteString("(")
	for i, it := range items {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString("?")
		args = append(args, it)
	}
	b.WriteString(")")
	return b.String(), args
}
