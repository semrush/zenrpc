package str

import (
	"strings"
	"unicode"
)

var (
	ToSnakeCase    = toSomeCaseWithSep('_', unicode.ToLower)
	ToURLSnakeCase = toSomeCaseWithSep('-', unicode.ToLower)
	ToDotSnakeCase = toSomeCaseWithSep('.', unicode.ToLower)
	ToLowerCase    = strings.ToLower
)

func ToNoCase(s string) string {
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
