package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNavWriter(t *testing.T) {
	input := `
nav:
	- Пользователю:
	- Руководство пользователя:
		# UserGuide.adoc {
		text
		# UserGuide.adoc }
	- Администратору:
`
	exp := `
nav:
	- Пользователю:
	- Руководство пользователя:
		# UserGuide.adoc {
		- nav1
		- nav2
		# UserGuide.adoc }
	- Администратору:
`
	nav := []string{"- nav1", "- nav2"}
	res, err := writeNav(input, "UserGuide.adoc", nav)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, exp,res)
}
