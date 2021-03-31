package main

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

func writeNav(input string, fileName string, nav []string) (string, error) {
	s1 := regexp.QuoteMeta(fileName)
	var re = regexp.MustCompile(fmt.Sprintf(`(?ms)^(\s*)(# %s {\s*?\r?\n).*(^\s*# %s })`, s1, s1))
	matches := re.FindStringSubmatch(input)
	if matches == nil || len(matches) != 4 {
		return "", errors.New("cannot find expected pattern to replace")
	}
	indent := matches[1]
	navStr := indent + strings.Join(nav, "\n" + indent) + "\n"
	return re.ReplaceAllString(input, "${1}${2}" + navStr + "$3"), nil
}
