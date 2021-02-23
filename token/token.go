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
	BLOCK_DELIM   = "BLOCK_DELIM" // "----" block delimiter
	BLOCK_OPTS    = "BLOCK_OPTS" // "[source,json]" code block options
	INDENT        = "INDENT"
	HEADER        = "HEADER"
	HOR_LINE      = "HOR_LINE"
	LIST          = "LIST"   //not-numbered list marker
	LIST_N 		  = "LIST_N" //numbered list  marker
	BLOCK_IMAGE   = "BLOCK_IMAGE"
	INLINE_IMAGE  = "INLINE_IMAGE"
	ADMONITION    = "ADMONITION"
	CONCAT_PAR    = "CONCAT_PAR" // "+" symbol between paragraphs
	BOOKMARK      = "BOOKMARK" // "[[bookmark_text]]"
	BLOCK_TITILE  = "BLOCK_TITLE"
)