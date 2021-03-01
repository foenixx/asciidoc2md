package utils

import (
	"fmt"
	"unicode/utf8"
)

// https://stackoverflow.com/questions/41602230/return-first-n-chars-of-a-string
func FirstN(s string, n int) string {
	i := 0
	//iterate over runes
	for j := range s {
		if i == n {
			return s[:j]
		}
		i++
	}
	return s
}

func LastN(s string, len int, n int) string {

	skipFirstN :=  len - n
	i := 0
	//iterate over runes
	for j := range s {
		if i == skipFirstN {
			return s[j:]
		}
		i++
	}
	return ""
}

func ShortenString(s string, first int, last int) string {
	ln := utf8.RuneCountInString(s)
	if ln <= first + last + 3 {
		//nothing to shor...ten
		return s
	}
	l := LastN(s, ln, last)
	if l == "" {
		return FirstN(s, first)
	} else {
		return fmt.Sprintf("%s...%s", FirstN(s, first), l)
	}
}

func RuneIs(r rune, runes ...rune) bool {
	for _, el := range runes {
		if el == r {
			return true
		}
	}
	return false
}