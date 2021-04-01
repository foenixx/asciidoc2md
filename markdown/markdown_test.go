package markdown

import (
	"asciidoc2md/parser"
	"asciidoc2md/utils"
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

type convCase struct{
	name  string
	input string
	exp   string
}

var cases = []convCase{
	{
		name: "complex list item",
		input: `
* item0
* item1
+
.Example title
[caption=""]
====
example text
====`,
		exp: `
* item0

* item1

  _Example title_

  !!! example
      example text
`,
	},
	{
		name:  "header with custom id",
		input: `
[[v3.6]]
== Версия 3.6
`,
		exp:
`## Версия 3.6 { #v3.6 }
`,
	},
}

var input2 = `
* item0
* item1
`


func testACase(t *testing.T, tc *convCase, log slog.Logger) {
	p := parser.New(tc.input, nil, log)
	doc, err := p.Parse("test.adoc")
	if !assert.NoError(t, err) {
		return
	}
	w := strings.Builder{}
	conv := Converter{imageFolder: "data/images/", log: log}
	conv.RenderMarkdown(doc, &w)
	assert.Equal(t, tc.exp, w.String())
}

func testAFile(t *testing.T, fIn string, fOut string, log slog.Logger) {
	input, err := ioutil.ReadFile(fIn)
	if !assert.NoError(t, err) {
		return
	}

	p := parser.New(string(input), nil, log)
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
		conv := Converter{imageFolder: "data/images/", log: log}
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

func TestEscapeHtml(t *testing.T) {
	assert.Equal(t, "`это` ка&lt;кие&gt;-то `неправильные` пчелы `и они`&lt;&gt;", utils.FixFormatting("`это` ка<кие>-то `неправильные` пчелы `и они`<>"))
}


func Test1(t *testing.T) {
	logger := slogtest.Make(t, nil).Leveled(slog.LevelDebug)
	testAFile(t, "../test.adoc", "../mkdocs_test/docs/index.md", logger)
}

func testConverter(t *testing.T) {
	logger := slogtest.Make(t, nil).Leveled(slog.LevelInfo)
	input := "NOTE: В параметре `source` можно также задавать маскированные пути к файлам, например `Types/Cards/Kr*.jtype` импортирует типы карточек (файлы с расширением .jtype), которые начинаются с букв \"Kr\". Маски задаются символами `*` и `?` таким же образом, как и для других команд в командной строке Windows."

	inc := ``
	p := parser.New(input, func(name string) ([]byte, error) {
		return []byte(inc), nil
	}, logger)
	doc, err := p.Parse("test.adoc")
	assert.Nil(t, err)
	logger.Info(context.Background(), doc.StringWithIndent(""))
	var builder = strings.Builder{}
	conv := Converter{imageFolder: "data/images/", log: logger}
	conv.log.Debug(context.Background(), "message")
	conv.RenderMarkdown(doc, &builder)
	res := builder.String()
	logger.Info(context.Background(), res)
	assert.Equal(t, "", res)
}

func TestFixString(t *testing.T) {
	assert.Equal(t, "bc de", fixString("*bc de*", true))
	assert.Equal(t, "[x] abcd", fixString("[*] abcd", false))
	assert.Equal(t, "some **bold** text and mi**dd**le and *)", fixString("some *bold* text and mi**dd**le and *)", false))
	assert.Equal(t, "`#abc_id` \\# de", fixString("#abc_id # de", false))
	assert.Equal(t, `&lt;&gt;`, fixString("<>", false))
	assert.Equal(t, "**SQL условие** - условие", fixText(`*SQL условие* - условие`))
	inp := `+++\ ` + "`" + ` * _ { } [ ] ( ) # + - . ! |+++`
	exp :=  `\\ ` + "\\` " + `\* \_ \{ \} \[ \] \( \) \# \+ \- \. \! \|`
	assert.Equal(t, exp, fixText(inp))
	assert.Equal(t, "` some text `", fixText("`+++ some text +++`"))
	assert.Equal(t, "символами `*` и `?` таким", fixText("символами `*` и `?` таким"))
	assert.Equal(t, "Через `#view` представление", fixText("Через #view представление"))
	assert.Equal(t, "оператора `#if`, который ", fixText("оператора `*#if*`, который "))
	assert.Equal(t, "Маппинг полей (тип объекта **person** / **user**):", fixText("Маппинг полей (тип объекта *person* / *user*):"))
}