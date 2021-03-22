package utils

import (
	"fmt"
	"reflect"
	"strings"
	"unicode/utf8"
)



func FixFormatting(s string) string {
	var inside = false
	var prev int
	b := strings.Builder{}
	//loop over all runes in s
	for pos, r := range s {
		if r == '`' {
			inside = !inside
		}
		if !inside && (r == '<' || r == '>' || r == '#') {
			b.WriteString(s[prev:pos])
			switch r {
			case '<':
				b.WriteString("&lt;")
			case '>':
				b.WriteString("&gt;")
			case '#':
				b.WriteString(`\#`)
			}
			prev = pos + 1
		}
	}
	b.WriteString(s[prev:])
	return b.String()
}

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

func IsNil(i interface{}) bool {
	if i == nil {
		return true
	}
	switch reflect.TypeOf(i).Kind() {
	case reflect.Ptr, reflect.Map, reflect.Array, reflect.Chan, reflect.Slice:
		return reflect.ValueOf(i).IsNil()
	}
	return false
}

func CountDirs(path string) int {
	parts := strings.Split(path, "/")
	cnt := 0
	for _, d := range parts {
		if strings.TrimSpace(d) != "" {
			cnt++
		}
	}
	return cnt
}