package parser

import (
	"bufio"
	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/slogtest"
	"context"
	"errors"
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
		incFile string
		incContent string
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
            container block:
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
* li1
* li2
====
`,
		expected: `
document:
  block title: title
  example block:
    paragraph:
      text: any text
    list begin: (0/false/*)
    item:
      container block:
        paragraph:
          text: li1
    item:
      container block:
        paragraph:
          text: li2
    list end`,
	},
	{
		name: "include",
		input:
		`
= Header 1

== Header 1.1

[[include_ref]]
include::inc.adoc[leveloffset=+1]

== Header 1.2
`,
		incFile: "inc.adoc",
		incContent:
		`
= Header i1

== Header i1.1

== Header i1.2
`,
		expected:
		`
document:
  header: 1, Header 1
  header: 2, Header 1.1
  bookmark: include_ref
  document:
    header: 2, Header i1
    header: 3, Header i1.1
    header: 3, Header i1.2
  header: 2, Header 1.2`,
	},
	{
		name: "paragraph",
		input:	`В появившемся окне нажимаем кнопку *Открыть* image:image031.png[] и указываем окне импорта появятся карточки из выбранной библиотеки.`,
		expected:
		`
document:
  paragraph:
    text: В появившемся окне нажимаем кнопку *Открыть* 
    inline image: image031.png
    text:  и указываем окне импорта появ...точки из выбранной библиотеки.`,
	},
	{
		name: "definition list",
		input:	`def list 1::
+
text 1
+
def list 2::
+
text 2
+
def list 3::
+
text 3`,
		expected:
		`
document:
  list begin: (0/false/::)
  item:
    container block:
      paragraph:
        text: def list 1
      paragraph:
        text: text 1
  item:
    container block:
      paragraph:
        text: def list 2
      paragraph:
        text: text 2
  item:
    container block:
      paragraph:
        text: def list 3
      paragraph:
        text: text 3
  list end`,
	},
	{
		name: "fenced block",
		input: "``` sql\n  line1  \nline2\n```",
		expected: `
document:
  syntax block: 
  line1  
line2
`,
	},
	{
		name: "block admonition",
		input: `
[NOTE]
====
Примеры выполняются на карточке типа "Дополнительное соглашение".
====
`,
		expected: `
document:
  admonition block: NOTE
    paragraph:
      text: Примеры выполняются на карточк...а "Дополнительное соглашение".`,
	},
}

func testACase(t *testing.T, tc *parserTestCase, log slog.Logger) {
	p := New(tc.input, func(s string) ([]byte, error) {
		if s != tc.incFile {
			return nil, errors.New("invalid include file name")
		}
		return []byte(tc.incContent), nil
	}, log)
	doc, err := p.Parse("test.adoc")

	if assert.NoError(t, err) {
		t.Log(doc.StringWithIndent(""))
		assert.Equal(t, tc.expected, doc.StringWithIndent(""))
	}
}

func testAFile(t *testing.T, fIn string, fOut string, log slog.Logger) {
	input, err := ioutil.ReadFile(fIn)
	if !assert.NoError(t, err) {
		return
	}

	p := New(string(input), nil, log)
	log.Info(context.Background(), "test message")
	doc, err := p.Parse("test.adoc")
	if !assert.NoError(t, err) {
		return
	}
	log.Debug(context.Background(), doc.StringWithIndent(""))
	//os.Stdout.WriteString(doc.StringWithIndent(""))
	if fOut != "" {
		fo, err := os.Create(fOut)
		if !assert.NoError(t, err) {
			return
		}
		defer fo.Close()
		w := bufio.NewWriter(fo)
		w.WriteString(doc.StringWithIndent(""))
		err = w.Flush()
		if !assert.NoError(t, err) {
			return
		}
	}
}

func Test1(t *testing.T) {
	logger := slogtest.Make(t, nil).Leveled(slog.LevelDebug)
	testAFile(t, "../docs/beginners/BeginnersGuide.adoc", "test.out", logger)
}

func TestAllCases(t *testing.T) {
	logger := slogtest.Make(t, nil)
	logger.Info(context.Background(), "log message")

	for _, tc := range cases {
		testACase(t, &tc, logger)
	}
}

var case1 = parserTestCase{
name: "debug",
input:
`
В области предпросмотра есть кнопки для управления областью (кнопки доступны только когда в области предпросмотра не открыт файл):

image:image190.png[] - скрыть область предпросмотра файлов. Снова отобразить ее можно будет с помощью контекстного меню в списке файлов;

image:image191.png[] - поменять местами область карточки и область предпросмотра файлов;

image:image192.png[] - разделяет в равных долях область карточки и область предпросмотра файлов (актуально, если пользователем была перемещена вертикальная граница области карточки/области предпросмотра).
`,
expected:
``,
}

func TestParser_DebugCase(t *testing.T) {
	logger := slogtest.Make(t, nil)
	logger.Info(context.Background(), "log message")

	testACase(t, &case1, logger)
}

