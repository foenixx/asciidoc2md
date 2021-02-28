package lexer

import (
	"asciidoc2md/token"
	"asciidoc2md/utils"
	"strings"
	"unicode/utf8"
)

type Lexer struct {
	input        string
	position     int // current position in input (points to current char)
	readPosition int // current reading position in input (after current char)
	prevToken    *token.Token
	ch			 rune
	receiver	 func(tok *token.Token)
	line 		 uint //current line
	tableFlag 		bool //we've started parsing table, this flag is set after "|===" token occurred
}

//lexer position
type Position struct {
	position     int
	readPosition int
	ch			 rune
	line 		 uint
	tableFlag 	bool
}

func New(input string, receiver func(token2 *token.Token)) *Lexer {
	l := &Lexer{input: input}
	l.receiver = receiver
	l.line = 1
	l.readRune()
	l.prevToken = &token.Token{Type: token.NEWLINE}
	return l
}

func (l *Lexer) LastToken() *token.Token {
	return l.prevToken
}

func (l *Lexer) Position() *Position {
	return &Position{
		position:     l.position,
		readPosition: l.readPosition,
		ch:           l.ch,
		line:         l.line,
		tableFlag:    l.tableFlag,
	}
}
//used for debugging
func (l *Lexer) forceFinish() {
	l.ch = 0
}

func (l *Lexer) readRune() {
	var width int
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch,  width = utf8.DecodeRuneInString(l.input[l.readPosition:])
	}
	l.position = l.readPosition
	l.readPosition += width
}

func (l *Lexer) peekRune() rune {
	if l.readPosition >= len(l.input) {
		return 0
	} else {
		ch, _ := utf8.DecodeRuneInString(l.input[l.readPosition:])
		return ch
	}
}

func (l *Lexer) Rewind(pos *Position) {
	l.position = pos.position
	l.readPosition = pos.readPosition
	l.ch = pos.ch
	l.line = pos.line
	l.tableFlag = pos.tableFlag
}

func (l *Lexer) setNewToken(typ token.TokenType, line uint, literal string) *token.Token {
	l.prevToken = &token.Token{Type: typ, Line: line, Literal: literal}
	l.receiver(l.prevToken)
	return l.prevToken
}

func (l *Lexer) setToken(tok *token.Token) *token.Token {
	l.prevToken = tok
	l.receiver(tok)
	return l.prevToken
}

func (l *Lexer) ReadAll() {
	for l.NextToken() {}
}

func (l *Lexer) NextToken() bool {

	switch  {
	case l.ch == '.' && l.prevToken.Type == token.NEWLINE && !utils.RuneIs(l.peekRune(), []rune{'.','*',' ','\t'}):
		//block title ".title"
		l.readRune() // move to a next char
		l.setNewToken(token.BLOCK_TITLE, l.line, l.readLine())
	case l.ch == '+' && l.prevToken.Type == token.NEWLINE && isNewLine(l.peekRune()):
		// paragraph concatenation
		ch := l.ch
		l.readRune() // move to a next char
		l.readWhitespace() //skip whitespace after
		l.setNewToken(token.CONCAT_PAR, l.line, string(ch))
	case isNewLine(l.ch):
		//new line
		l.setNewToken(token.NEWLINE, l.line, l.readNewLine())
	case isWhitespace(l.ch) && l.prevToken.Type == token.NEWLINE:
		//leading spaces: indentation
		l.setNewToken(token.INDENT, l.line, l.readWhitespace())
	case l.ch == '=' && l.prevToken.Type == token.NEWLINE:
		t := l.readHeaderOrExample()
		l.setToken(t)

	case isListMarker(l.ch) && utils.RuneIs(l.peekRune(), []rune{'.','*',' ','\t'}):
		// "* list" is a list
		// "** list" is a list
		// "*text" is not a list
		// ".text" is not a list - this case is captured above in the block title case
		ch := l.ch
		m := l.readListMarker()
		l.readWhitespace() //skip whitespace after
		if ch == '*' {
			l.setNewToken(token.L_MARK, l.line, m)
		} else {
			l.setNewToken(token.NL_MARK, l.line, m)
		}
	case l.ch == '[' && l.prevToken.Type == token.NEWLINE:
		l.readRune()
		if l.ch == '[' {
			//bookmark
			l.setToken(l.readBookmark())
		} else {
			//block options "[source, json]"
			l.setToken(l.readBlockOptions())
		}
	case l.ch == ':' && l.prevToken.Type == token.NEWLINE:
		//ignore asciidoc options ":keyword: text"
		l.readLine()
	case l.ch == 0:
		l.setNewToken(token.EOF, l.line, "")
		return false
	default:
		tokens := l.readString() //read til EOL

		switch {
		case tokens[0].Type == token.BLOCK_DELIM:
			// "----" syntax block
			l.setNewToken(token.SYNTAX_BLOCK, l.line, l.readSyntaxBlock(tokens[0]))
		case tokens[0].Type == token.TABLE:
			//invert flag
			l.tableFlag = !l.tableFlag
			l.setToken(tokens[0])
		default:
			for _, tok := range tokens {
				l.setToken(tok)
			}
		}

	}
	return true
}

func (l *Lexer) readSyntaxBlock(delim *token.Token) string {
	l.readRune() //skip newline
	pos := l.position
	var to int
	for {
		to = l.position
		line := l.readLine()
		// read without tokenizing till the same delimiter or ...
		if strings.TrimSpace(line) == delim.Literal {
			break
		}
		// ... or EOF
		if l.ch == 0 {
			to = l.position
			break
		}
		l.readNewLine() //skip newline
	}
	return l.input[pos:to]
}

// reads "[source,json]" like lines
func (l *Lexer) readBlockOptions() *token.Token {
	pos := l.position
	opts := l.readLine()
	//should be enclosed in brackets, opening "[" is skipped by the calling code
	if opts[len(opts) - 1] == ']' {
		// return options without brackets
		return &token.Token{Type: token.BLOCK_OPTS, Line: l.line, Literal: opts[: len(opts) - 1]}
	}
	return &token.Token{Type: token.ILLEGAL, Line: l.line, Literal: l.input[pos:l.position]}
}

// reads "[[bookmark_text]]]"
func (l *Lexer) readBookmark() *token.Token {
	pos := l.position
	b := l.readLine()
	//should be enclosed in double brackets
	if strings.HasSuffix(b, "]]") {
		// return bookmark text
		return &token.Token{Type: token.BOOKMARK, Line: l.line, Literal: b[1: len(b) - 2]}
	}
	return &token.Token{Type: token.ILLEGAL, Line: l.line, Literal: l.input[pos:l.position]}
}

func (l *Lexer) readHeaderOrExample() *token.Token {
	from := l.position
	for l.ch == '=' {
		l.readRune()
	}
	literal := l.input[from:l.position]
	l.readWhitespace() //skip whitespace before header text
	if isNewLine(l.ch) {
		//example block, not a header:
		//  ====
		//  text
		//  ====
		if literal == "====" {
			return &token.Token{Type: token.EX_BLOCK, Line: l.line, Literal: literal}
		}
		return &token.Token{Type: token.ILLEGAL, Line: l.line, Literal: literal}
	}
	return &token.Token{Type: token.HEADER, Line: l.line, Literal: literal}
}

func (l *Lexer) readNewLine() string {
	ch := l.ch
	literal := string(ch)
	l.readRune()
	if ch == '\r' && l.ch == '\n' {
		literal += string(l.ch)
		l.readRune()
	}
	l.line++
	return literal
}

func (l *Lexer) readWhitespace() string {
	pos := l.position
	for isWhitespace(l.ch) {
		l.readRune()
	}
	return l.input[pos:l.position]
}

func (l *Lexer) readWord() string {
	pos := l.position
	//read until word delimiter of column separator (in table mode)
	for !isWordDelimiter(l.ch) && !(l.tableFlag && isColumn(l.ch)) {
		l.readRune()
	}
	if l.position == pos && (l.tableFlag && isColumn(l.ch)) {
		//word starts from "|"
		l.readRune() //skip to the next rune
		return "|"
	}
	return l.input[pos:l.position]
}

func (l *Lexer) readLine() string {
	pos := l.position
	for ! (isNewLine(l.ch) || l.ch == 0) {
		l.readRune()
	}
	return l.input[pos:l.position]
}

func (l *Lexer) readListMarker() string {
	pos := l.position
	for isListMarker(l.ch) {
		l.readRune()
	}
	return l.input[pos:l.position]
}

func (l *Lexer) readString() []*token.Token {
	//we are either at where the line begins or at the start of the word
	tokens := make([]*token.Token, 0)
	pos := l.position
	end := l.position // end of the last processed word
	var w string
	var tok *token.Token

	//process full-line keywords first
	if l.prevToken.Type == token.NEWLINE {
		lexerState := l.Position()
		w = l.readLine()
		tok = l.lookupLineKeyword(w)
		if tok != nil {
			l.readLine() //skip the line
			tokens = append(tokens, tok)
			return tokens
		}
		//didn't found a keyword, rewind to previous position
		l.Rewind(lexerState)
	}
	// not let's process inline keywords
	// read word by word til EOL or EOF
	for ! (isNewLine(l.ch) || l.ch == 0) {
		// read a word without embracing whitespaces
		w = l.readWord()
		tok = l.lookupInlineKeyword(w)

		if tok != nil {
			// we've found a keyword, lets check if there were some text before it
			if end > pos {
				// yes we've got text before the keyword, let's produce it as first token
				tokens = append(tokens, &token.Token{Type: token.STR, Line: l.line, Literal: l.input[pos:end]})
			}
			// producing found keyword
			tokens = append(tokens, tok)
			l.readWhitespace() // skip whitespace after the keyword
			pos = l.position
		}
		end = l.position // end of the last processed word
		l.readWhitespace()
	}
	if l.position > pos {
		tokens = append(tokens, &token.Token{Type: token.STR, Line: l.line, Literal: l.input[pos:l.position]})
	}
	return tokens
}

func (l *Lexer) lookupInlineKeyword(w string) *token.Token {
	switch {
	case strings.HasPrefix(w,"image:"): //inline image
		return &token.Token{Type: token.INLINE_IMAGE, Line: l.line, Literal: w}
	case l.tableFlag && w == "|": //column
		return &token.Token{Type: token.COLUMN, Line: l.line, Literal: w}
	case l.tableFlag && w == "a" && l.ch == '|':
		l.readRune()
		return &token.Token{Type: token.A_COLUMN, Line: l.line, Literal: "a|"}
	}
	return nil
}

/*
lookupLineKeyword is used only for starting from newline keywords
*/
func (l *Lexer) lookupLineKeyword(w string) *token.Token {
	switch {
	case strings.HasPrefix(w, "|==="):  //table
		return &token.Token{Type: token.TABLE, Line: l.line, Literal: w}
	//case l.tableFlag && w == "|": //column
	//	return &token.Token{Type: token.COLUMN, Line: l.line, Literal: w}
	case strings.HasPrefix(w, "____"): //quotation block
		return &token.Token{Type: token.QUOTE_BLOCK, Line: l.line, Literal: "____"}
	case strings.HasPrefix(w, "----"): //block delimiter
		// actual literal could have trailing spaces, let's don't bother trimming them
		return &token.Token{Type: token.BLOCK_DELIM, Line: l.line, Literal: "----"}
	case strings.HasPrefix(w, "image::"): //block image
		return &token.Token{Type: token.BLOCK_IMAGE, Line: l.line, Literal: w}
	case strings.HasPrefix(w,"image:"): //inline image
	//	return &token.Token{Type: token.INLINE_IMAGE, Line: l.line, Literal: w}
	//case w == "NOTE:" || w == "TIP:" || w == "IMPORTANT:" || w == "WARNING:" || w == "CAUTION:":
		//admonition
		return &token.Token{Type: token.ADMONITION, Line: l.line, Literal: w[0:len(w)-1] /*name without trailing ":"*/}
	case w == "'''" && l.prevToken.Type == token.NEWLINE:
		return &token.Token{Type: token.HOR_LINE, Line: l.line, Literal: w}
	case w == "//EOF" && l.prevToken.Type == token.NEWLINE:
		//interrupt parsing here, simplify debugging
		l.forceFinish()
	}
	return nil
}

func isColumn(ch rune) bool {
	return ch == '|'
}

func isWhitespace(ch rune) bool {
	return ch == ' ' || ch == '\t'
}

func isNewLine(ch rune) bool {
	return ch == '\n' || ch == '\r'
}


func isWordDelimiter(ch rune) bool {
	return isWhitespace(ch) || isNewLine(ch) ||
			//ch == '.' || ch == ',' || ch == '!' || ch == '?' || //punctuation
			ch == 0
}


func isListMarker(ch rune) bool {
	return ch == '*' || ch == '.'
}