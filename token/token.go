package token

import "fmt"

type TokenType string

type Token struct {
	Type TokenType
	Literal string
	Line uint
	//Position int
}

func (t *Token) String() string {
	return fmt.Sprintf("[ type:%v, line:%v, literal:%s ]", t.Type, t.Line, t.Literal)
}

const (
	ILLEGAL = "ILLEGAL"
	EOF = "EOF"
	// Delimiters
	NEWLINE = "NEWLINE"
	// Values
	STR         = "STR"
	SYNTAX_BLOCK = "SYNTAX_BLOCK"
	// Keywords
	BLOCK_DELIM  = "BLOCK_DELIM" // "----" block delimiter
	BLOCK_OPTS   = "BLOCK_OPTS" // "[source,json]" code block options
	INDENT       = "INDENT"
	HEADER       = "HEADER"
	HOR_LINE     = "HOR_LINE"
	L_MARK       = "L_MARK"   //not-numbered list marker
	NL_MARK      = "NL_MARK" //numbered list  marker
	BLOCK_IMAGE  = "BLOCK_IMAGE"
	INLINE_IMAGE = "INLINE_IMAGE"
	ADMONITION   = "ADMONITION"
	CONCAT_PAR   = "CONCAT_PAR" // "+" symbol between paragraphs
	BOOKMARK     = "BOOKMARK" // "[[bookmark_text]]"
	BLOCK_TITLE  = "BLOCK_TITLE"
	EX_BLOCK     = "EX_BLOCK" //open example block "===="
	QUOTE_BLOCK  = "QUOTE_BLOCK" //quotation block "____"
	//EX_BLOCK_R   = "EX_BLOCK_R" //close example block "===="
)