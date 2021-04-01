package lexer

import (
	"asciidoc2md/token"
	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/slogtest"
	"context"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"regexp"
	"testing"
)

func TestReadRune(t *testing.T) {
	input := "日本語 олле"
	tests := []struct {
		expectedRune rune
		expectedPosition int
	}{
		{'日',0},
		{'本',3},
		{'語',6},
		{' ',9},
		{'о',10},
		{'л',12},
		{'л',14},
		{'е',16},
		{0,18},
	}
	l := New(input)
	for _, tt := range tests {
		assert.Equal(t, tt.expectedRune, l.ch)
		assert.Equal(t, tt.expectedPosition, l.position)
		l.readRune()
	}
}

func TestShifts(t *testing.T) {
	input := `абвгд`
	l := New(input)
	assert.Equal(t, 'а', l.ch)
	l.readRune()
	assert.Equal(t, 'б', l.ch)
	l.readRune()
	assert.Equal(t, 'в', l.ch)
	l.Shift(2) // 'г' takes 2 bytes, 'д' expected
	assert.Equal(t, 'д', l.ch)
	l.Shift(-2) // back to 'г'
	assert.Equal(t, 'г', l.ch)
	l.Shift(-2*3) // back to 'а'
	assert.Equal(t, 'а', l.ch)
}

func TestShifts2(t *testing.T) {
	input := "<.> ab\n"
	lex := New(input)
	//l := lex.readLine()
	tok := lex.tryReadToken()
	assert.Equal(t, &token.Token{token.AL_MARK,"<.>",1}, tok)
	tok = lex.tryReadToken()
	assert.Equal(t, &token.Token{token.STR,"ab",1}, tok)
}

func TestNextToken(t *testing.T) {
	input := "строка1 ==="
	l := New(input)
	var tok *token.Token
	tok = l.NextToken()
 	assert.Equal(t, token.STR, int(tok.Type))
	assert.Equal(t, "строка1 ===", tok.Literal)
}

type (
	lexerTestCase struct {
		name     string
		input    string
		expected []lt
	}
	lexerTest struct {
		typ token.TokenType
		lit string
	}
	lt lexerTest //short alias
)
var (
	nl = lt{token.NEWLINE, "\n"}
	eof = lt{token.EOF, ""}
)

var cases = []lexerTestCase{
	{
		name:  "test 1",
		input: "\r текст с отступом\nтекст с пробелом после \r\n\n\rкакой-то текст",
		expected: []lt{
			{token.NEWLINE, "\r"},
			{token.INDENT, " "},
			{token.STR, "текст с отступом"},
			{token.NEWLINE, "\n"},
			{token.STR, "текст с пробелом после "},
			{token.NEWLINE, "\r\n"},
			{token.NEWLINE, "\n"},
			{token.NEWLINE, "\r"},
			{token.STR, "какой-то текст"},
			{token.EOF, ""},
		},
	},
	{
		name: "test 2",
		input: `= Заголовок 1
*bold*
* list 1
* list 2
** nested list
* list 3
строка1 ===
строка2
== Заголовок 2
строка 3
`,
		expected: []lt{
			{token.HEADER, "="},
			{token.STR, "Заголовок 1"},
			{token.NEWLINE, "\n"},
			{token.STR, "*bold*"},
			{token.NEWLINE, "\n"},
			{token.L_MARK, "*"},
			{token.STR, "list 1"},
			{token.NEWLINE, "\n"},
			{token.L_MARK, "*"},
			{token.STR, "list 2"},
			{token.NEWLINE, "\n"},
			{token.L_MARK, "**"},
			{token.STR, "nested list"},
			{token.NEWLINE, "\n"},
			{token.L_MARK, "*"},
			{token.STR, "list 3"},
			{token.NEWLINE, "\n"},
			{token.STR, "строка1 ==="},
			{token.NEWLINE, "\n"},
			{token.STR, "строка2"},
			{token.NEWLINE, "\n"},
			{token.HEADER, "=="},
			{token.STR, "Заголовок 2"},
			{token.NEWLINE, "\n"},
			{token.STR, "строка 3"},
			{token.NEWLINE, "\n"},
			{token.EOF, ""},
		},
	},
	{
		name: "images",
		input: `image::image15_3.png[]

После внесения изменений image:image15_3.png[] схему данных необходимо сохранить.

image::image15_4.png[]`,
		expected: []lt{
			{token.BLOCK_IMAGE, "image::image15_3.png[]"},
			{token.NEWLINE, "\n"},
			{token.NEWLINE, "\n"},
			{token.STR, "После внесения изменений "},
			{token.INLINE_IMAGE, "image:image15_3.png[]"},
			{token.STR, " схему данных необходимо сохранить."},
			{token.NEWLINE, "\n"},
			{token.NEWLINE, "\n"},
			{token.BLOCK_IMAGE, "image::image15_4.png[]"},
			{token.EOF, ""},
		},
	},
	{
		name: "images 2",
		input: `image:image1.png[] text1`,
		expected: []lt{
			{token.INLINE_IMAGE, "image:image1.png[]"}, {token.STR, " text1"}, {token.EOF, ""},
		},
	},
	{
		name: "mixed lists",
		input:
		`. list1
.. list2
.not a list
*not a list
*** list3
* list4`,
		expected: []lt{
			{token.NL_MARK, "."}, {token.STR, "list1"}, {token.NEWLINE, "\n"},
			{token.NL_MARK, ".."}, {token.STR, "list2"}, {token.NEWLINE, "\n"},
			{token.BLOCK_TITLE, `not a list`}, {token.NEWLINE, "\n"},
			{token.STR, "*not a list"}, {token.NEWLINE, "\n"},
			{token.L_MARK, "***"}, {token.STR, "list3"}, {token.NEWLINE, "\n"},
			{token.L_MARK, "*"}, {token.STR, "list4"}, {token.EOF, ""},
		},
	},
	{
		name: "list boundary",
		input:
		`* list1
+
--
text 1

** list 11

text 2
--`,
		expected: []lt{
			{token.L_MARK, "*"}, {token.STR, "list1"}, nl,
			{token.CONCAT_PAR, "+"}, nl,
			{token.L_BOUNDARY, "--"}, nl,
			{token.STR, "text 1"}, nl, nl,
			{token.L_MARK, "**"}, {token.STR, "list 11"}, nl, nl,
			{token.STR, "text 2"}, nl,
			{token.L_BOUNDARY, "--"}, eof,
		},
	},
	{
		name: "syntax block",
		input:
		`----
"DocLoad.OutputFolderFormat": "yyyy-MM-dd_HH-mm-ss"
---- `,
		expected: []lt{
			{token.SYNTAX_BLOCK, `"DocLoad.OutputFolderFormat": "yyyy-MM-dd_HH-mm-ss"` + "\n"},
			{token.EOF, ""},
		},
	},
	{
		name: "block title",
		input:
		`.title 1
some text`,
		expected: []lt{
			{token.BLOCK_TITLE, `title 1`},
			{token.NEWLINE, "\n"},
			{token.STR, `some text`},
			{token.EOF, ""},
		},
	},
	{
		name: "inline keywords",
		input: `====
Если в правиле доступа... //EOF то
в противном ---- случае ==== никаких____
----`,
		expected: []lt{
			{token.EX_BLOCK, `====`},{token.NEWLINE, "\n"},
			{token.STR, `Если в правиле доступа... //EOF то`},{token.NEWLINE, "\n"},
			{token.STR, `в противном ---- случае ==== никаких____`},{token.NEWLINE, "\n"},
			{token.SYNTAX_BLOCK, ``},
			{token.EOF, ""},
		},
	},
	{
		name: "table",
		input: `|===
| text1 | text2|
 a|text3 | text4|
	|text5|text6
|===
| text7 | text8|`,
		expected: []lt{
			{token.TABLE, `|===`},{token.NEWLINE, "\n"},
			{token.COLUMN, `|`}, {token.STR, " text1 "},
				{token.COLUMN, `|`}, {token.STR, " text2"},{token.COLUMN, `|`},
					{token.NEWLINE, "\n"},
			{token.INDENT, " "},{token.A_COLUMN, "a|"}, {token.STR, "text3 "},
				{token.COLUMN, `|`}, {token.STR, " text4"},{token.COLUMN, `|`},
					{token.NEWLINE, "\n"},
			{token.INDENT, "\t"},{token.COLUMN, `|`}, {token.STR, "text5"},
				{token.COLUMN, `|`}, {token.STR, "text6"},{token.NEWLINE, "\n"},
			{token.TABLE, `|===`},{token.NEWLINE, "\n"},
			{token.STR, `| text7 | text8|`},	{token.EOF, ""},
		},
	},
	{
		name: "bookmark",
		input: "[[bookmark1]]**Структура `json` с опциями слияния, [[bookmark2]]описание свойств, их типы и значения по умолчанию:**",
		expected: []lt{
			{token.BOOKMARK, "bookmark1"},{token.STR, "**Структура `json` с опциями слияния, "},
			{token.BOOKMARK, "bookmark2"},{token.STR, "описание свойств, их типы и значения по умолчанию:**"},
			{token.EOF, ""},
		},
	},
	{
		name: "admonition",
		input: "NOTE: Admonition text",
		expected: []lt{
			{token.ADMONITION, "NOTE"},{token.STR, "Admonition text"},
			{token.EOF, ""},
		},
	},
	{
		name: "links",
		input: "text1 https://olle[text2] \ntext3",
		expected: []lt{
			{token.STR, "text1 "}, {token.URL, "https://olle"},
				{token.LINK_NAME, "text2"},{token.STR, " "}, {token.NEWLINE, "\n"},
			{token.STR, "text3"}, {token.EOF, ""},
		},
	},
	{
		name: "comments",
		input: "text1\n// text2\ntext3",
		expected: []lt{
			{token.STR, "text1"}, {token.NEWLINE, "\n"},
			{token.COMMENT, "// text2"}, {token.NEWLINE, "\n"},
			{token.STR, "text3"}, {token.EOF, ""},
		},

	},
	{
		name: "definition list",
		input:
`def list 1::
+
text 1
+
def list 2::
+
text 2`,
		expected: []lt{
			{token.DEFL_MARK, "def list 1"}, {token.NEWLINE, "\n"},
			{token.CONCAT_PAR, "+"}, {token.NEWLINE, "\n"},
			{token.STR, "text 1"}, {token.NEWLINE, "\n"},
			{token.CONCAT_PAR, "+"}, {token.NEWLINE, "\n"},
			{token.DEFL_MARK, "def list 2"}, {token.NEWLINE, "\n"},
			{token.CONCAT_PAR, "+"}, {token.NEWLINE, "\n"},
			{token.STR, "text 2"}, {token.EOF, ""},
		},

	},
	{
		name: "fenced block",
		input: "``` sql\n  line1  \nline2\n```",
		expected: []lt{
			{token.FENCED_SYNTAX_BLOCK, "sql\n  line1  \nline2\n"}, {token.EOF, ""},
		},

	},
	{
		name: "code annotations",
		input: `<.> text1
<.> text2`,
		expected: []lt{
			{token.AL_MARK, "<.>"},{token.STR, "text1"}, nl,
			{token.AL_MARK, "<.>"},{token.STR, "text2"}, eof,
		},
	},

}

func logLexems(t *testing.T, input string, logger slog.Logger) {
	l := New(input)

	//t.Logf("type: %v, literal: %v", tok2.Type, tok2.Literal)
	tok := l.NextToken()

	for tok != nil  {
		logger.Debug(context.Background(), tok.String())
		tok = l.NextToken()
	}
}

func testACase(t *testing.T, tc *lexerTestCase, logger slog.Logger) {

	l := New(tc.input)

	//t.Logf("type: %v, literal: %v", tok2.Type, tok2.Literal)
	tok := l.NextToken()
	i := 0
	for tok != nil && i < len(tc.expected) {
		logger.Debug(context.Background(), tok.String())

		if !assert.Equal(t, tc.expected[i].typ, tok.Type, "invalid type! case: %v, step: %v, token: %v", tc.name, i+1, tok) {
				return
			}
		if !assert.Equal(t, tc.expected[i].lit, tok.Literal, "invalid literal! case: %v, step: %v", tc.name, i+1) {
			return
		}

		tok = l.NextToken()
		i++
	}

	//no more expected tokens
	if !assert.Equal(t, i, len(tc.expected)) {
			return
		}
	assert.Nil(t, tok)

}

func TestAllCases(t *testing.T) {
	logger := slogtest.Make(t, nil).Leveled(slog.LevelInfo)

	for _, tc := range cases {
		logger.Info(context.Background(), "-------- " + tc.name + "-----------")
		testACase(t, &tc, logger)
	}
}


var dcase = lexerTestCase {
		name: "bookmark",
		input: "[[cardmergeoptionsdetails]]**Структура `json` с опциями слияния, [[bookmark]]описание свойств, их типы и значения по умолчанию:**",
		expected: []lt{
			{token.BOOKMARK, "cardmergeoptionsdetails"},{token.STR, "**Структура `json` с опциями слияния, описание свойств, их типы и значения по умолчанию:**"},
			{token.EOF, ""},
		},
}

func TestDbg(t *testing.T) {
	input :=
		`
[IMPORTANT]
===============================
Флаг *Полнотекстовый поиск по сообщениям в обсуждениях* может быть не доступен для редактирования, так как при установке TESSA не был установлен необходимый компонент полнотекстового поиска вашей СУБД. Для того чтобы этот флаг был доступен для редактирования, установите этот компонент.

* Если TESSA установлена на систему Windows, в качестве СУБД MS SQL Server, то у Вас этот компонент должен быть установлен, исходя из https://docs.microsoft.com/en-us/sql/relational-databases/search/get-started-with-full-text-search?view=sql-server-ver15[документации MS SQL Server]
* Если вы используете PostgreSQL на любой системе, установка дополнительных компонентов не требуется.
* Если TESSA установлена на Linux и при этом используется MS SQL Server, то по умолчанию пакеты, которые предоставляет Microsoft для https://docs.microsoft.com/en-us/sql/linux/sql-server-linux-overview?view=sql-server-linux-ver15[различных дистрибутивов], не включает в себя этот компонент. Его необходимо установить дополнительно. Руководство по установке для дистрибутивов Linux есть на сайте с https://docs.microsoft.com/en-us/sql/linux/sql-server-linux-setup-full-text-search?view=sql-server-ver15[документацией для MS SQL Server]

После установки это компонента заново импортируйте схему.
===============================
`
	//input := "NOTE: Admonition text"

	logger := slogtest.Make(t, nil).Leveled(slog.LevelDebug)
	logLexems(t, input, logger)
	//testACase(t, &lexerTestCase{input: input}, logger)
}

func TestFile1(t *testing.T) {
	logger := slogtest.Make(t, nil).Leveled(slog.LevelDebug)
	input, err := ioutil.ReadFile("../test.adoc")
	if !assert.NoError(t, err) {
		return
	}

	logLexems(t, string(input), logger)
	//var tc lexerTestCase
	//tc.name = "test.adoc"
	//tc.input = string(input)
	//
	//testACase(t, &tc, logger)
}

func testMethodDbg(t *testing.T) {
	var fencedRE = regexp.MustCompile(`^\x60{3}\s*(\S*)\s*$`)
	input := "d```  "
	m := fencedRE.FindStringSubmatch(input)
	t.Log(m)
	t.Log(len(m))
	t.Fail()
}