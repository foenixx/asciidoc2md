package main

import (
	"asciidoc/parser"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

var input2 = `
* item 1
** item 1.1
** item 1.2
*** item 1.2.1
* item 2
** item 2.1

`

var input = `
* item 1
** item 1.1
* item 2
`


func TestConverter(t *testing.T) {
	p := parser.New(input)
	doc, errs := p.Parse()
	assert.Len(t, errs, 0, "errors: %v", errs)
	var builder = strings.Builder{}
	conv := Converter{ImageFolder: "data/images/"}
	conv.RenderMarkdown(doc, &builder)
	assert.Equal(t, "text", builder.String())
}