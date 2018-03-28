package parser

import (
	"strings"
	"unicode"
)

var (
	toSnakeCase = toSomeCaseWithSep('_', unicode.ToLower)
	toLowerCase = strings.ToLower
)

func toNoCase(s string) string {
	return s
}

func toSomeCaseWithSep(sep rune, runeConv func(rune) rune) func(string) string {
	return func(s string) string {
		in := []rune(s)
		n := len(in)
		var runes []rune
		for i, r := range in {
			if unicode.IsSpace(r) {
				runes = append(runes, sep)
				continue
			}
			if unicode.IsUpper(r) {
				if i > 0 && sep != runes[i-1] && ((i+1 < n && unicode.IsLower(in[i+1])) || unicode.IsLower(in[i-1])) {
					runes = append(runes, sep)
				}
				r = runeConv(r)
			}
			runes = append(runes, r)
		}
		return string(runes)
	}
}
