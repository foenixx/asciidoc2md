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

type convtc struct{
	input string
	output string
}

var cases = []convtc{
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


func testACase(t *testing.T, tc *convtc, log slog.Logger) {
	p := parser.New(tc.input, nil, log)
	doc, err := p.Parse("test.adoc")
	if !assert.NoError(t, err) {
		return
	}
	w := strings.Builder{}
	conv := Converter{imageFolder: "data/images/", log: log}
	conv.RenderMarkdown(doc, &w)
	assert.Equal(t, tc.output, w.String())
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
	assert.Equal(t, "`это` ка&lt;кие&gt;-то `неправильные` пчелы `и они`&lt;&gt;", utils.ReplaceHtmlOutsideBackticks("`это` ка<кие>-то `неправильные` пчелы `и они`<>"))
}


func Test1(t *testing.T) {
	logger := slogtest.Make(t, nil).Leveled(slog.LevelDebug)
	testAFile(t, "../test.adoc", "../test.md", logger)
}

func TestConverter(t *testing.T) {
	logger := slogtest.Make(t, nil).Leveled(slog.LevelInfo)
	input :=
`
В появившемся окне нажимаем кнопку *Открыть* image:image031.png[] и указываем путь к библиотеке карточек из сборки Configuration\Cards\Tessa.ms.cardlib (или Tessa.pg.cardlib, если установка выполняется для СУБД PostgreSQL). В окне импорта появятся карточки из выбранной библиотеки.
`
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
	assert.Equal(t, "text", res)
}