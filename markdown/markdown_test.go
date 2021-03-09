package markdown

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
	p := parser.New(tc.input, log)
	doc, err := p.Parse()
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

func Test1(t *testing.T) {
	logger := slogtest.Make(t, nil).Leveled(slog.LevelDebug)
	testAFile(t, "../data/installation.adoc", "../test.md", logger)
}

func TestConverter(t *testing.T) {
	logger := slogtest.Make(t, nil).Leveled(slog.LevelInfo)
	input :=
`

.Установите .NET Runtime & Windows Server Hosting bundle
* Установка описана https://docs.microsoft.com/en-us/aspnet/core/host-and-deploy/iis/?view=aspnetcore-5.0#install-the-net-core-hosting-bundle[в документации на сайте Microsoft]:
** Перейдите https://dotnet.microsoft.com/download/dotnet/5.0[на страницу загрузки]. В разделе Runtime 5.0.x найдите последнюю доступную версию (обычно сверху), скачайте и установите Runtime & Hosting Bundle для Windows по ссылке "Hosting Bundle" (на картинке ниже).
+
[.text-center]
image::image088.png[]
+
IMPORTANT: На странице со списком версий не выбирайте preview-версию.
+
IMPORTANT: Устанавливайте .NET Hosting bundle ТОЛЬКО после установки и настройки IIS. Исправление описано в разделе <<repair-hosting, Возможных проблем>>.
`
	p := parser.New(input, logger)
	doc, err := p.Parse()
	assert.Nil(t, err)
	logger.Info(context.Background(), doc.String(""))
	var builder = strings.Builder{}
	conv := Converter{imageFolder: "data/images/", log: logger}
	conv.log.Debug(context.Background(), "message")
	conv.RenderMarkdown(doc, &builder)
	res := builder.String()
	logger.Info(context.Background(), res)
	assert.Equal(t, "text", res)
}