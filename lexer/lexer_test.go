package lexer

import (
	"asciidoc2md/token"
	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/slogtest"
	"context"
	"io/ioutil"
	"log"
	"testing"
	"github.com/stretchr/testify/assert"
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
	l := New(input, func(tok *token.Token) {
		log.Print("i am a receiver")
	})
	for _, tt := range tests {
		assert.Equal(t, tt.expectedRune, l.ch)
		assert.Equal(t, tt.expectedPosition, l.position)
		l.readRune()
	}
}

func TestShifts(t *testing.T) {
	input := `абвгд`
	l := New(input, func(tok2 *token.Token) {
		//t.Logf("type: %v, literal: %v", tok2.Type, tok2.Literal)
	})
	assert.Equal(t, 'а', l.ch)
	l.readRune()
	assert.Equal(t, 'б', l.ch)
	l.readRune()
	assert.Equal(t, 'в', l.ch)
	l.Shift(2) // 'г' takes 2 bytes, 'д' expected
	assert.Equal(t, 'д', l.ch)
	l.Shift(-2) // back to 'д'
	assert.Equal(t, 'д', l.ch)
	l.Shift(-2*3) // back to 'д'
	assert.Equal(t, 'в', l.ch)

}

type (
	lexerTestCase struct {
		name string
		input string
		tests []lt
	}
	lexerTest struct {
		expectedType token.TokenType
		expectedLiteral string
	}
	lt lexerTest //short alias
)

var cases = []lexerTestCase{
	{
		name:  "test 1",
		input: "\r текст с отступом\nтекст с пробелом после \r\n\n\rкакой-то текст",
		tests: []lt{
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
		tests: []lt{
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
		name: "case 3: images",
		input: `image::image15_3.png[]

После внесения изменений image:image15_3.png[] схему данных необходимо сохранить.

image::image15_4.png[]`,
		tests: []lt{
			{token.BLOCK_IMAGE, "image::image15_3.png[]"},
			{token.NEWLINE, "\n"},
			{token.NEWLINE, "\n"},
			{token.STR, "После внесения изменений"},
			{token.INLINE_IMAGE, "image:image15_3.png[]"},
			{token.STR, "схему данных необходимо сохранить."},
			{token.NEWLINE, "\n"},
			{token.NEWLINE, "\n"},
			{token.BLOCK_IMAGE, "image::image15_4.png[]"},
			{token.EOF, ""},
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
		tests: []lt{
			{token.NL_MARK, "."}, {token.STR, "list1"}, {token.NEWLINE, "\n"},
			{token.NL_MARK, ".."}, {token.STR, "list2"}, {token.NEWLINE, "\n"},
			{token.BLOCK_TITLE, `not a list`}, {token.NEWLINE, "\n"},
			{token.STR, "*not a list"}, {token.NEWLINE, "\n"},
			{token.L_MARK, "***"}, {token.STR, "list3"}, {token.NEWLINE, "\n"},
			{token.L_MARK, "*"}, {token.STR, "list4"}, {token.EOF, ""},
		},
	},
	{
		name: "syntax block",
		input:
		`----
"DocLoad.OutputFolderFormat": "yyyy-MM-dd_HH-mm-ss"
---- `,
		tests: []lt{
			{token.SYNTAX_BLOCK, `"DocLoad.OutputFolderFormat": "yyyy-MM-dd_HH-mm-ss"` + "\n"},
			{token.EOF, ""},
		},
	},
	{
		name: "block title",
		input:
		`.title 1
some text`,
		tests: []lt{
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
		tests: []lt{
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
| text1 | text2|`,
		tests: []lt{
			{token.TABLE, `|===`},{token.NEWLINE, "\n"},
			{token.COLUMN, `|`}, {token.STR, "text1"},
				{token.COLUMN, `|`}, {token.STR, "text2"},{token.COLUMN, `|`},
					{token.NEWLINE, "\n"},
			{token.INDENT, " "},{token.A_COLUMN, "a|"}, {token.STR, "text3"},
				{token.COLUMN, `|`}, {token.STR, "text4"},{token.COLUMN, `|`},
					{token.NEWLINE, "\n"},
			{token.INDENT, "\t"},{token.COLUMN, `|`}, {token.STR, "text5"},
				{token.COLUMN, `|`}, {token.STR, "text6"},{token.NEWLINE, "\n"},
			{token.TABLE, `|===`},{token.NEWLINE, "\n"},
			{token.STR, `| text1 | text2|`},	{token.EOF, ""},
		},
	},
	{
		name: "bookmark",
		input: "[[cardmergeoptionsdetails]]**Структура `json` с опциями слияния, [[bookmark]]описание свойств, их типы и значения по умолчанию:**",
		tests: []lt{
			{token.BOOKMARK, "cardmergeoptionsdetails"},{token.STR, "**Структура `json` с опциями слияния, описание свойств, их типы и значения по умолчанию:**"},
			{token.EOF, ""},
		},
	},

}


func testACase(t *testing.T, tc *lexerTestCase, logger slog.Logger) {
	lex := []*token.Token{}

	l := New(tc.input, func(tok2 *token.Token) {
		lex = append(lex, tok2)
		t.Logf("type: %v, literal: %v", tok2.Type, tok2.Literal)
	})
	l.ReadAll()
	if !assert.Len(t, lex, len(tc.tests)) {
		return
	}

	for i, tt := range tc.tests {
		var tok *token.Token
		if i >= len(lex) {
			tok = lex[len(lex)-1]
		} else {
			tok = lex[i]
		}
		assert.Equal(t, tt.expectedType, tok.Type, "invalid type! case: %v, step: %v", tc.name, i + 1)
		assert.Equal(t, tt.expectedLiteral, tok.Literal, "invalid literal! case: %v, step: %v", tc.name, i + 1)
	}

}

func TestAllCases(t *testing.T) {
	logger := slogtest.Make(t, nil).Leveled(slog.LevelInfo)

	for _, tc := range cases {
		logger.Info(context.Background(), "-------- " + tc.name + "-----------")
		testACase(t, &tc, logger)
	}
}


var cases1 = []lexerTestCase {
	{
		name: "debug",
		input: `
=== Системные требования

[[book1]]*Требования к аппаратной конфигу[[book2]]рации*

NOTE: Указаны примерные в расчёте на среднюю активность пользователей и могут отличаться в зависимости от нагрузки и развернутой конфигурации.
`,
		tests: []lt{},
	},
}

func TestCases1(t *testing.T) {
	logger := slogtest.Make(t, nil).Leveled(slog.LevelInfo)
	testACase(t, &cases1[0], logger)
}

func TestFile1(t *testing.T) {
	logger := slogtest.Make(t, nil).Leveled(slog.LevelInfo)
	input, err := ioutil.ReadFile("../data/test.adoc")
	if !assert.NoError(t, err) {
		return
	}
	var tc lexerTestCase
	tc.name = "test.adoc"
	tc.input = string(input)

	testACase(t, &tc, logger)
}
