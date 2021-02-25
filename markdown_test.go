package main

import (
	"asciidoc2md/parser"
	"bufio"
	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/slogtest"
	"context"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

type convtc struct{
	input string
	output string
}

var cases = []convtc {
	{
		input: `
* item0
* item1
+
.Example title
[caption=""]
====
example text
====`,
		output: `
* item0
* item1`,
	}}

var input2 = `
* item0
* item1
`

var input = `
* item 1
`

func testACase(t *testing.T, tc *convtc, log slog.Logger) {
	p := parser.New(tc.input, log)
	doc, err := p.Parse()
	if !assert.NoError(t, err) {
		return
	}
	w := strings.Builder{}
	conv := Converter{ImageFolder: "data/images/", log: log}
	conv.RenderMarkdown(doc, &w)
	assert.Equal(t, tc.output, w.String())
}

func testAFile(t *testing.T, fIn string, fOut string, log slog.Logger) {
	input, err := ioutil.ReadFile(fIn)
	if !assert.NoError(t, err) {
		return
	}

	p := parser.New(string(input), log)
	doc, err := p.Parse()
	if !assert.NoError(t, err) {
		return
	}
	log.Debug(context.Background(), doc.String(""))
	//os.Stdout.WriteString(doc.String(""))
	if fOut != "" {
		fo, err := os.Create(fOut)
		if !assert.NoError(t, err) {
			return
		}
		defer fo.Close()
		w := bufio.NewWriter(fo)
		conv := Converter{ImageFolder: "data/images/", log: log}
		conv.RenderMarkdown(doc, w)
		err = w.Flush()
		if !assert.NoError(t, err) {
			return
		}
	}
}

func TestAll(t *testing.T) {
	logger := slogtest.Make(t, nil).Leveled(slog.LevelDebug)
	for _, tc := range cases {
		testACase(t, &tc, logger)
	}
}

func Test1(t *testing.T) {
	logger := slogtest.Make(t, nil).Leveled(slog.LevelDebug)
	testAFile(t, "data/test.adoc", "test.md", logger)
}

func TestConverter(t *testing.T) {
	logger := slogtest.Make(t, nil).Leveled(slog.LevelInfo)
	p := parser.New(input2, logger)
	doc, err := p.Parse()
	assert.Nil(t, err)
	logger.Info(context.Background(), doc.String(""))
	var builder = strings.Builder{}
	conv := Converter{ImageFolder: "data/images/", log: logger}
	conv.log.Debug(context.Background(), "message")
	conv.RenderMarkdown(doc, &builder)
	assert.Equal(t, "text", builder.String())
}