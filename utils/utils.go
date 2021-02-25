package utils


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

func RuneIs(r rune, list []rune) bool {
	for _, el := range list {
		if el == r {
			return true
		}
	}
	return false
}