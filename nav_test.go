package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"regexp"
	"strings"
	"testing"
)

func testDebug(t *testing.T) {
	input := `
nav:
	- Пользователю:
	- Руководство пользователя:
		# UserGuide.adoc {
		text
		# UserGuide.adoc }
	- Администратору:`
	input = strings.ReplaceAll(input, "\n", "\r\n")
	s := "UserGuide.adoc"
	s = regexp.QuoteMeta(s)
	//var re = regexp.MustCompile(fmt.Sprintf(`(?m)^(\s*)(# %s {.*)$.*(^\s*# %s })$`, s, s))
	var re = regexp.MustCompile(fmt.Sprintf(`(?ms)^(\s*)(# %s {\s*?\r?\n).*(^\s*# %s })`, s, s))
	matches := re.FindStringSubmatch(input)
	for _, m := range matches {
		fmt.Sprintf("%q\n", m)
	}

}

func TestNavWriter(t *testing.T) {
	input := `
nav:
	- Пользователю:
	- Руководство пользователя:
		# UserGuide.adoc {
		text
		# UserGuide.adoc }
	- Администратору:`
	exp := `
nav:
	- Пользователю:
	- Руководство пользователя:
		# UserGuide.adoc {
		- nav1
		- nav2
		# UserGuide.adoc }
	- Администратору:`
	nav :=[]string{"- nav1", "- nav2"}
	res, err := writeNav(input, "UserGuide.adoc", nav)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, exp,res)
}
