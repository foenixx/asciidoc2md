package parser

import (
	"bufio"
	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/slogtest"
	"context"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

type (
	parserTestCase struct {
		name string
		input string
		expected string
	}
)

var cases = []parserTestCase{
	{
		name: "lists 1",
		input: `* Item 1
** Item 1.1
+
image::image1.png[]
+
More text.
+
NOTE: Admonition text.
+
** Item 1.2`,
		expected: `
document:
  list begin: (0/false/*)
  item:
    container block:
      paragraph:
        text: Item 1
      list begin: (1/false/**)
      item:
        container block:
          paragraph:
            text: Item 1.1
          image: image1.png
          paragraph:
            text: More text.
          admonition: NOTE
            paragraph:
              text: Admonition text.
      item:
        container block:
          paragraph:
            text: Item 1.2
      list end
  list end`,
	},
	{
		name: "lists 2",
		input: `. Item 1
* Item 1.1
. Item 2`,
		expected: `
document:
  list begin: (0/true/.)
  item 1:
    container block:
      paragraph:
        text: Item 1
      list begin: (1/false/*)
      item:
        container block:
          paragraph:
            text: Item 1.1
      list end
  item 2:
    container block:
      paragraph:
        text: Item 2
  list end`,
	},
	{
		name: "block title",
		input:
		`.title text
* list

. list
+
.title
paragraph text
`,
		expected: `
document:
  block title: title text
  list begin: (0/false/*)
  item:
    container block:
      paragraph:
        text: list
  list end
  list begin: (0/true/.)
  item 1:
    container block:
      paragraph:
        text: list
      block title: title
      paragraph:
        text: paragraph text
  list end`,
	},
	{
		name: "example block",
		input:
		`.title
[options]
====
any text
====
`,
		expected: `
document:
  block title: title
  example block:
    paragraph:
      text: any text`,
	},
}

func testACase(t *testing.T, tc *parserTestCase, log slog.Logger) {
	p := New(tc.input, log)
	doc, err := p.Parse()

	if assert.NoError(t, err) {
		t.Log(doc.String(""))
		assert.Equal(t, tc.expected, doc.String(""))
	}
}

func testAFile(t *testing.T, fIn string, fOut string, log slog.Logger) {
	input, err := ioutil.ReadFile(fIn)
	if !assert.NoError(t, err) {
		return
	}

	p := New(string(input), log)
	log.Info(context.Background(), "test message")
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
		w.WriteString(doc.String(""))
		err = w.Flush()
		if !assert.NoError(t, err) {
			return
		}
	}
}

func Test1(t *testing.T) {
	logger := slogtest.Make(t, nil).Leveled(slog.LevelDebug)
	testAFile(t, "../data/test.adoc", "test.out", logger)
}

func TestParser(t *testing.T) {
	logger := slogtest.Make(t, nil)
	logger.Info(context.Background(), "log message")

	for _, tc := range cases {
		testACase(t, &tc, logger)
	}
}

var case1 = parserTestCase{
name: "debug",
input:
"[[cardmergeoptionsdetails]]**Структура `json` с опциями слияния, описание свойств, их типы и значения по умолчанию:**",
expected: "",
}

func TestParser_DebugCase(t *testing.T) {
	logger := slogtest.Make(t, nil)
	logger.Info(context.Background(), "log message")

	testACase(t, &cases[0], logger)
}

