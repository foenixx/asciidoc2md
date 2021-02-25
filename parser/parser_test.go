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
container block:
  list begin: (0/false/*)
  item:
    container block:
      container block:
        text: Item 1
      list begin: (1/false/**)
      item:
        container block:
          container block:
            text: Item 1.1
          image: image1.png
          container block:
            text: More text.
          admonition: NOTE
            paragraph:
              text: Admonition text.
      item:
        container block:
          container block:
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
container block:
  list begin: (0/true/.)
  item 1:
    container block:
      container block:
        text: Item 1
      list begin: (1/false/*)
      item:
        container block:
          container block:
            text: Item 1.1
      list end
  item 2:
    container block:
      container block:
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
		expected: ``,
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
		expected: ``,
	},
	{
		name: "debug",
		input:
`
Допустим, вы используете нумератор с именем вида Входящие-{YYYY}-{f:MainInfo.FolderName}.
В карточке у вас есть поле "Папка" (MainInfo.FolderId, FolderName), где вы выбираете папку из номенклатуры дел.
Если поле не заполняется автоматически сразу при создании карточки (специальным расширением), а должно выбираться регистратором, то для корректной работы в данном случае необходимо настроить:
`	,
		expected: "",
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

func TestParser_DebugCase(t *testing.T) {
	logger := slogtest.Make(t, nil)
	logger.Info(context.Background(), "log message")

	testACase(t, &cases[len(cases)-1], logger)
}

