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
|===
|Параметр |Описание
a|
Строка подключения.
[[conn-string]]
.Для подключения к SQL Server с использованием Windows аутентификации:
[source, xml, subs="macros+", role=small]
----
  "ConnectionStrings": {
        "default": "Server=pass:quotes[#.\\SQLEXPRESS#]; Database=pass:quotes[#tessa#]; Integrated Security=true; Connect Timeout=200; pooling='true'; Max Pool Size=200; MultipleActiveResultSets=true;"
    }
----
.Для подключения с использованием пользователя SQL Server:
[source, xml, subs="macros+", role=small]
----
  "ConnectionStrings": {
        "default": "Server=pass:quotes[#.\\SQLEXPRESS#]; Database=pass:quotes[#tessa#]; Integrated Security=false; User ID=pass:quotes[#sa#]; Password=pass:quotes[#master#]; Connect Timeout=200; pooling='true'; Max Pool Size=200; MultipleActiveResultSets=true;"
    }
----
.Для подключения с использованием пользователя SQL Server и указанием номера порта (1433 - номера порта по умолчанию для протокола TCP/IP):
[source, xml, subs="macros+", role=small]
----
  "ConnectionStrings": {
        "default": "Server=pass:quotes[#.\\SQLEXPRESS,1433#]; Database=pass:quotes[#tessa#]; Integrated Security=false; User ID=pass:quotes[#sa#]; Password=pass:quotes[#master#]; Connect Timeout=200; pooling='true'; Max Pool Size=200; MultipleActiveResultSets=true;"
    }
----
.Для подключения с использованием пользователя PostgreSQL:
[source, xml, subs="macros+", role=small]
----
  "ConnectionStrings": {
        "default": [ "Host=pass:quotes[#localhost#]; Database=pass:quotes[#tessa#]; Integrated Security=false; User ID=pass:quotes[#postgres#]; Password=pass:quotes[#Master1234#]; Pooling=true; MaxPoolSize=100", "Npgsql" ]
  },
----
|

Строка подключения к базе данных Tessa в формате http://msdn.microsoft.com/ru-ru/library/system.data.sqlclient.sqlconnection.connectionstring.aspx[Sql Server Connection string]/PostgreSQL connection strings. 

Не забывайте, что подключение к MS SQL Server в случае использования Windows аутентификации (Integrated Security=true) будет происходить от учетной записи, от которой запущен пул приложений, обычно это 

a|
[[server-code]]
[source, xml, subs="macros+", role=small]
----
"ServerCode": "pass:quotes[#tessa#]"
----
|
Код сервера. Для разных инсталляций Tessa указывайте разные коды приложений, например, "prod" или "qa". Код сервера используется для формирования ссылок tessa:// для desktop-клиента, при этом код сервера в Tessa Applications и на сервере должны совпадать. Также код сервера используется для разделения глобального кэша метаинформации между процессами, поэтому при использовании на сервере приложения нескольких экземпляров системы, укажите для каждого из них отличающийся код сервера. Подробнее по установке второго сервиса на одном сервере приложений см. в разделе <<secondinstance, Установка второго экземпляра Tessa на этом же сервере приложений>>.

|===
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