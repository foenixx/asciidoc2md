package lexer

import (
	"asciidoc2md/token"
	"asciidoc2md/utils"
	"regexp"
	"strings"
	"unicode/utf8"
)

type Lexer struct {
	input        string
	position     int // current position in input (points to current char)
	readPosition int // current reading position in input (after current char)
	prevToken    *token.Token
	ch			 rune
	line 		 uint //current line
	tableFlag 		bool //we've started parsing table, this flag is set after "|===" token occurred
}

//lexer position
type State struct {
	position     int
	readPosition int
	ch			 rune
	line 		 uint
	tableFlag 	bool
	prevToken   *token.Token
}

func New(input string) *Lexer {
	l := &Lexer{input: input}
	l.line = 1
	l.readRune()
	l.prevToken = &token.Token{Type: token.NEWLINE}
	return l
}

func (l *Lexer) LastToken() *token.Token {
	return l.prevToken
}

func (l *Lexer) GetState() *State {
	return &State{
		position:     l.position,
		readPosition: l.readPosition,
		ch:           l.ch,
		line:         l.line,
		tableFlag:    l.tableFlag,
		prevToken:    l.prevToken,
	}
}
//used for debugging
func (l *Lexer) forceFinish() {
	l.ch = 0
}

//Shift increments l.readPosition on bts bytes (not runes!) and reads symbol there.
// For example, the current lexer state is:
//
//    (readPosition)
//     ↓
//   абвгд
//    ↑
//   (l.ch='б')
//
// Now l.Shift(2) gets us at:
//
//       (readPosition)
//       ↓
//   абвгд
//      ↑
//     (l.ch='г')
//
func (l *Lexer) Shift(bts int) {
	l.readPosition += bts
	l.readRune()
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

func (l *Lexer) Rewind(pos *State) {
	l.position = pos.position
	l.readPosition = pos.readPosition
	l.ch = pos.ch
	l.line = pos.line
	l.tableFlag = pos.tableFlag
	l.prevToken = pos.prevToken
}

func (l *Lexer) setNewToken(typ token.TokenType, line uint, literal string) *token.Token {
	l.prevToken = &token.Token{Type: typ, Line: line, Literal: literal}
	return l.prevToken
}

func (l *Lexer) setToken(tok *token.Token) *token.Token {
	l.prevToken = tok
	return l.prevToken
}

func (l *Lexer) NextToken() *token.Token {
	if l.prevToken.Type == token.EOF {
		return nil
	}

	for {
		pos := l.position
		tok := l.next()
		pos2 := l.GetState()
		if tok != nil {

			//merge all subsequent STRs
			if tok.Type == token.STR {
				for {
					// break if token type is not STR or we got stuck.
					// next() can be null if it has eaten some whitespace,
					// otherwise (if no advance) we got stuck
					if (tok != nil && tok.Type != token.STR) ||
						(tok == nil && l.position == pos2.position) {
						break //for
					}
					pos2 = l.GetState()
					tok = l.next()
				}
				if tok != nil && tok.Type != token.STR {
					l.Rewind(pos2)
				}
				return &token.Token{Type: token.STR, Line: l.line, Literal: l.input[pos:l.position]}
			}


			return tok
		}
		if pos == l.position {
			return &token.Token{Type: token.ILLEGAL, Literal: "reader got stuck", Line: l.line}
		}
	}
}

func (l *Lexer) next() *token.Token {

	switch  {
	case l.ch == '.' && l.prevToken.Type == token.NEWLINE && !utils.RuneIs(l.peekRune(), '.','*',' ','\t'):
		//block title ".title"
		l.readRune() // move to a next char
		return l.setNewToken(token.BLOCK_TITLE, l.line, l.readLine())
	case l.ch == '+' && l.prevToken.Type == token.NEWLINE && isNewLine(l.peekRune()):
		// paragraph concatenation
		ch := l.ch
		l.readRune() // move to a next char
		l.readWhitespace() //skip whitespace after
		return l.setNewToken(token.CONCAT_PAR, l.line, string(ch))
	case isNewLine(l.ch):
		//new line
		return l.setNewToken(token.NEWLINE, l.line, l.readNewLine())
	case isWhitespace(l.ch) && l.prevToken.Type == token.NEWLINE:
		//leading spaces: indentation
		return l.setNewToken(token.INDENT, l.line, l.readWhitespace())
	case l.ch == '=' && l.prevToken.Type == token.NEWLINE:
		t := l.readHeaderOrExample()
		return l.setToken(t)

	case isListMarker(l.ch) && l.prevToken.Type == token.NEWLINE && utils.RuneIs(l.peekRune(), '.','*',' ','\t'):

		// "* list" is a list
		// "** list" is a list
		// "*text" is not a list
		// ".text" is not a list - this case is captured above in the block title case
		ch := l.ch
		m := l.readListMarker()
		l.readWhitespace() //skip whitespace after
		if ch == '*' {
			return l.setNewToken(token.L_MARK, l.line, m)
		} else {
			return l.setNewToken(token.NL_MARK, l.line, m)
		}
	case l.ch == '[' && l.peekRune() == '[':
		//bookmark
		l.readRune() //second opening bracket
		l.readRune() //jump to the bookmark text
		return l.setToken(l.readBookmark())
	case l.ch == '[' && l.prevToken.Type == token.NEWLINE:
		//block options "[source, json]"
		return l.setToken(l.readBlockOptions())
	case l.ch == ':' && l.prevToken.Type == token.NEWLINE:
		//ignore asciidoc options ":keyword: text"
		l.readLine()
	case l.ch == 'a' && l.tableFlag && l.peekRune() == '|':
		l.readRune()
		l.readRune()
		return l.setNewToken(token.A_COLUMN, l.line, "a|")
	case l.ch == '|' && l.tableFlag && l.peekRune() != '=':
		l.readRune()
		return l.setNewToken(token.COLUMN, l.line, "|")
	case isWhitespace(l.ch):
		l.readWhitespace()
		return nil
	case l.ch == 0:
		return l.setNewToken(token.EOF, l.line, "")
	default:
		pos1 := l.GetState()
		tok := l.readString()
		//we got stuck without new tokens
		if l.position == pos1.position {
			return l.setNewToken(token.ILLEGAL, l.line, "got stuck")
		}

		switch {
		case tok.Type == token.BLOCK_DELIM:
			// "----" syntax block
			return l.setNewToken(token.SYNTAX_BLOCK, l.line, l.readSyntaxBlock(tok))
		case tok.Type == token.TABLE:
			//invert flag
			l.tableFlag = !l.tableFlag
			return l.setToken(tok)
		default:
			return l.setToken(tok)
		}

	}
	return nil
}


func (l *Lexer) readString() *token.Token {
	//we are either at where the line begins or at the start of the word
	//tokens := make([]*token.Token, 0)
	//pos := l.position
	//end := l.position // end of the last processed word
	var w string
	var tok *token.Token
	var bts int

		pos := l.GetState()
		w = l.readLine()
		if l.prevToken.Type == token.NEWLINE {
			tok, bts = l.lookupLineKeyword(w)
		}
		//no newline before or no line keywords found
		if tok == nil {
			tok, bts = l.lookupInlineKeyword(w)
		}
		if tok != nil {
			if bts < len(w) {
				//only part of the string is consumed, return the rest to processing
				l.Shift(bts - len(w))
			}
			return tok
		}
		//no token found, start from the beginning
		l.Rewind(pos)

		//try to find next possible non-STR token in the current line
		w = l.readWord()
		if w == "" {
			return nil
		}
		return &token.Token{Type: token.STR, Literal: l.input[pos.position:l.position], Line: l.line}

}

func (l *Lexer) lookupInlineKeyword(w string) (*token.Token, int) {
	switch {
	case strings.HasPrefix(w,"image:"): //inline image
		//find closing bracket
		br := strings.Index(w, "]")
		//cannot find closing bracket
		if br == - 1 {
			return &token.Token{Type: token.ILLEGAL, Literal: w, Line: l.line}, len(w)
		}
		return &token.Token{Type: token.INLINE_IMAGE, Line: l.line, Literal: w[:br+1]}, br + 1
//	case l.tableFlag && strings.HasPrefix(w, "|"): //column
//		return &token.Token{Type: token.COLUMN, Line: l.line, Literal: w}, len(w)
//	case l.tableFlag && w == "a" && l.ch == '|': // 'a|' column
//		return &token.Token{Type: token.A_COLUMN, Line: l.line, Literal: "a|"}, 2
	}
	return nil, 0
}


var admonitionRE = regexp.MustCompile(`^\s*((?:NOTE)|(?:TIP)|(?:IMPORTANT)|(?:WARNING)|(?:CAUTION)):\s(.*)$`)
/*
lookupLineKeyword is used only for starting from newline keywords.
Returns found token and count of consumed bytes.
*/
func (l *Lexer) lookupLineKeyword(w string) (*token.Token, int) {
	switch {
	case strings.HasPrefix(w, "|==="):  //table
		return &token.Token{Type: token.TABLE, Line: l.line, Literal: w}, len(w)
	//case l.tableFlag && w == "|": //column
	//	return &token.Token{Type: token.COLUMN, Line: l.line, Literal: w}
	case strings.HasPrefix(w, "____"): //quotation block
		return &token.Token{Type: token.QUOTE_BLOCK, Line: l.line, Literal: "____"}, len(w)
	case strings.HasPrefix(w, "----"): //block delimiter
		// actual literal could have trailing spaces, let's don't bother trimming them
		return &token.Token{Type: token.BLOCK_DELIM, Line: l.line, Literal: "----"}, len(w)
	case strings.HasPrefix(w, "image::"): //block image
		return &token.Token{Type: token.BLOCK_IMAGE, Line: l.line, Literal: w}, len(w)
	//case strings.HasPrefix(w,"image:"): //inline image
	//	return &token.Token{Type: token.INLINE_IMAGE, Line: l.line, Literal: w}
	case w == "'''" && l.prevToken.Type == token.NEWLINE:
		return &token.Token{Type: token.HOR_LINE, Line: l.line, Literal: w}, len(w)
	case w == "//EOF" && l.prevToken.Type == token.NEWLINE:
		//interrupt parsing here, for debugging sake
		l.forceFinish()
	case strings.HasPrefix(w,"image:"): //inline image
		//find closing bracket
		br := strings.Index(w, "]")
		//cannot find closing bracket
		if br == - 1 {
			return &token.Token{Type: token.ILLEGAL, Literal: w, Line: l.line}, len(w)
		}
		return &token.Token{Type: token.INLINE_IMAGE, Line: l.line, Literal: w[:br+1]}, br + 1
	//case l.tableFlag && w == "|": //column
	//	return &token.Token{Type: token.COLUMN, Line: l.line, Literal: w}, len(w)
	//case l.tableFlag && w == "a" && l.ch == '|': // 'a|' column
	//	return &token.Token{Type: token.A_COLUMN, Line: l.line, Literal: "a|"}, 2
	default:
		//case w == "NOTE:" || w == "TIP:" || w == "IMPORTANT:" || w == "WARNING:" || w == "CAUTION:":
		//admonition
		matches := admonitionRE.FindStringSubmatch(w)
		// full string match + 2 capturing groups
		if len(matches) != 3 {
			return nil, 0
		}
		return &token.Token{Type: token.ADMONITION, Line: l.line, Literal: matches[1]}, len(matches[1]) + 2 /* name and ": " */

	}
	return nil, 0
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

// reads "bookmark_text]]"
func (l *Lexer) readBookmark() *token.Token {
	pos := l.position
	//read until closing bracket
	l.readUntil(true, true, ']')
	if l.ch != ']' {
		return &token.Token{Type: token.ILLEGAL, Line: l.line, Literal: l.input[pos:l.position]}
	}
	l.readRune()
	if l.ch != ']' {
		return &token.Token{Type: token.ILLEGAL, Line: l.line, Literal: l.input[pos:l.position]}
	}
	// return bookmark text
	defer l.readRune() //jump to the text after bookmark
	return &token.Token{Type: token.BOOKMARK, Line: l.line, Literal: l.input[pos:l.position-1]}

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
	//read until word delimiter
	for !isWordDelimiter(l.ch) {
		l.readRune()
	}
	//nothing to read because first symbol is a delimiter, but not newline of eof
	if l.position == pos && !(isNewLine(l.ch) || l.ch == 0) {
		//last chance for parsing, move on by 1 rune
		l.readRune()
	}
	return l.input[pos:l.position]
}

func (l *Lexer) readUntil(eol bool, eof bool, runes ...rune) string {
	pos := l.position
	for ! (utils.RuneIs(l.ch, runes...) || (eol && isNewLine(l.ch)) || (eof && l.ch == 0)) {
		l.readRune()
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
	return isWhitespace(ch) || isNewLine(ch) || ch == 0 || ch == '[' || isColumn(ch)
}


func isListMarker(ch rune) bool {
	return ch == '*' || ch == '.'
}