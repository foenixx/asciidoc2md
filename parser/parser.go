package parser

import (
	"asciidoc2md/ast"
	"asciidoc2md/lexer"
	"asciidoc2md/token"
	"asciidoc2md/utils"
	"cdr.dev/slog"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type Parser struct{
	f IncludeFunc
	l       *lexer.Lexer
	tokens  []*token.Token
	next    int          //next token index
	tok     *token.Token //current token
	prevTok *token.Token // previous token
	//nextTok        *token.Token // next token
	//nestedListLevel int
	curBlock ast.Block //block which is being parsed
	log slog.Logger
	tableFlag bool
}

type IncludeFunc func(name string) ([]byte,error)

func New(input string, f IncludeFunc, logger slog.Logger) *Parser {
	var p Parser
	p.log = logger
	p.f = f
	p.l = lexer.New(input)
	return &p
}

func (p *Parser) advance() bool {
	//empty tokens list
	if len(p.tokens) == 0 || p.next == len(p.tokens) {
		return false
	}

	p.tok = p.tokens[p.next]
	if p.next == 0 {
		p.prevTok = &token.Token{Type:token.NEWLINE}
	} else {
		p.prevTok = p.tokens[p.next- 1]
	}
	p.next += 1
	/*
	if p.next == len(p.tokens) {
		// we reached the end of the tokens list
		p.nextTok = nil
	} else {
		p.nextTok = p.tokens[p.next]
	}
	 */
	return true
}

func (p *Parser) advanceMany(count uint) (res bool) {
	var index int
	res = true
	if p.next + int(count) > len(p.tokens) {
		//jump to the end of the tokens list
		index = len(p.tokens) - 1
		res = false
		p.next = len(p.tokens)
	} else {
		index = p.next + int(count) - 1
		p.next += int(count)
	}
	p.tok = p.tokens[index]
	p.prevTok = p.tokens[index - 1]
	return res
}

func (p *Parser) peekToken(shift int) *token.Token {
	if p.next+shift > len(p.tokens) {
		return nil
	}
	return p.tokens[p.next- 1 + shift]
}

func (p *Parser) readAll() {
	tok := p.l.NextToken()
	prev := tok
	for tok != nil {
		if tok.Type == token.EOF && prev.Type != token.NEWLINE {
			//add newline at the end of file to simplify parsing
			p.tokens = append(p.tokens, &token.Token{token.NEWLINE,"\n",tok.Line})
		}
		p.tokens = append(p.tokens, tok)
		prev = tok
		tok = p.l.NextToken()
	}
}

func (p *Parser) Parse(name string) (*ast.Document, error) {
	var doc ast.Document
	//use only file name without directory
	_, doc.Name = filepath.Split(name)
	p.readAll()

forLoop:
	for p.advance() {
		switch {
		case p.tok.Type == token.EOF:
			break forLoop
		case p.isListMarker():
			l, err := p.parseList(nil)
			if err != nil {
				return nil, err
			}
			doc.Add(l)
		case p.tok.Type == token.NEWLINE:
			//do nothing
		default:
			b, err := p.parseBlock()
			if err != nil {
				return nil, err
			}

			if b != nil && !utils.IsNil(b) {
				doc.Add(b)
			}
		}
	}
	return &doc, nil
}

var ErrCannotAdvance = errors.New("cannot advance tokens")

func (p *Parser) parseBlock() (ast.Block, error) {

	var options string

	if p.tok.Type == token.BLOCK_OPTS {
		options = p.tok.Literal
		//p.log.Debug(context.Background(), "parseBlock: BLOCK_OPTS", slog.F("token", p.tok))
		//skip to the token after newline
		if !p.advanceMany(2) {
			return nil, fmt.Errorf("cannot skip newline: unexpected EOF")
		}
	}


	switch {
	case p.isListMarker():
		return p.parseList(nil)
	case p.tok.Type == token.BLOCK_TITLE:
		t := ast.BlockTitle{Title: p.tok.Literal}
		if !p.advance() { return nil, ErrCannotAdvance }
		return &t, nil
	case p.tok.Type == token.HEADER:
		return p.parseHeader("")
	case p.isParagraph(p.tok):
		//paragraph
		return p.parseParagraph()
	case p.tok.Type == token.BLOCK_IMAGE:
		return p.parseImage(options)
	case p.tok.Type == token.INCLUDE:
		return p.parseInclude(options)
	case p.tok.Type == token.HOR_LINE:
		if !p.advance() {
			return nil, fmt.Errorf("cannot advance after HOR_LINE token")
		}
		return &ast.HorLine{}, nil
	case p.tok.Type == token.ADMONITION:
		return p.parseAdmonition()
	case p.tok.Type == token.EX_BLOCK:
		//example block
		return p.parseExampleBlock(options)
	case p.tok.Type == token.TABLE:
		return p.parseTable(options)
	case p.tok.Type == token.SYNTAX_BLOCK:
		sb := &ast.SyntaxBlock{Literal: p.tok.Literal}
		sb.SetOptions(options)
		if !p.advance() {
			return nil, ErrCannotAdvance
		}
		return sb, nil
	case p.tok.Type == token.BOOKMARK:
		return p.parseBookmark()
	case p.tok.Type == token.INDENT || p.tok.Type == token.CONCAT_PAR:
		//skip it for now
		if !p.advance() {
			return nil, ErrCannotAdvance
		}
		return nil, nil
	}
	return nil, fmt.Errorf("parse block: unknown token %v", p.tok)
	//return nil, nil
}

func (p *Parser) isDoubleNewline() bool {
	return p.tok.Type == token.NEWLINE && p.prevTok.Type == token.NEWLINE
}

func (p *Parser) isListMarker() bool {
	return p.tok.Type == token.NL_MARK || p.tok.Type == token.L_MARK
}

func (p *Parser) isColumn() bool {
	return p.tok.Type == token.COLUMN || p.tok.Type == token.A_COLUMN
}

func (p *Parser) isParagraph(tok *token.Token) bool {
	return tok.Type == token.STR || tok.Type == token.INLINE_IMAGE || tok.Type == token.URL || tok.Type == token.INT_LINK
}

func (p *Parser) isParagraphEnd() bool {
	if p.tok.Type == token.NEWLINE && p.isParagraph(p.peekToken(1)) {
		//ignore single NEWLINE
		return false
	}

	return !p.isParagraph(p.tok)

	//return (p.tok.Type == token.NEWLINE && p.prevTok.Type == token.NEWLINE) ||
	//	p.tok.Type == token.EOF ||
	//	p.isListMarker() ||
	//	p.tok.Type == token.CONCAT_PAR ||
	//	p.tok.Type == token.EX_BLOCK ||
	//	p.tok.Type == token.QUOTE_BLOCK ||
	//	//in table mode newline completes the paragraph (in simple mode)
	//	((p.tok.Type == token.COLUMN || p.tok.Type == token.A_COLUMN || p.tok.Type == token.NEWLINE) && p.tableFlag)
}

func (p *Parser) parseBookmark() (ast.Block, error) {
	b := &ast.Bookmark{Literal: p.tok.Literal}
	//check if it is an Id of a header
	if !p.advance() {
		return nil, ErrCannotAdvance
	}
	if p.tok.Type == token.NEWLINE && p.peekToken(1).Type == token.HEADER {
		if !p.advance() {
			return nil, ErrCannotAdvance
		}
		h, err := p.parseHeader(b.Literal)
		return h, err
	}
	return b, nil

}

func (p *Parser) parseInternalLink() (*ast.Link, error) {
	link := ast.Link{Internal: true}

	parts := strings.SplitN(p.tok.Literal, ",", 2)
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid internal link: %v", p.tok.Literal)
	}
	link.Url = parts[0]
	if len(parts) == 2 {
		link.Text = parts[1]
	}
	return &link, nil
}

func (p *Parser) parseLink() (*ast.Link, error) {
	link := ast.Link{Url: p.tok.Literal}

	if !p.advance() {
		return nil, ErrCannotAdvance
	}
	if p.tok.Type == token.LINK_NAME {
		link.Text = p.tok.Literal
		if !p.advance() {
			return nil, ErrCannotAdvance
		}
	}
	return &link, nil
}

func (p *Parser) parseExampleBlock(options string) (*ast.ExampleBlock, error) {
	//skip delimiter + newline tokens
	var ex ast.ExampleBlock
	ex.Options = options
	p.curBlock = &ex
	defer func(old ast.Block) { p.curBlock = old }(p.curBlock)

	if !p.advanceMany(2) {
		return nil, fmt.Errorf("parse example block: cannot advance tokens")
	}
	for p.tok.Type != token.EX_BLOCK && p.tok.Type != token.EOF {

		if p.tok.Type == token.NEWLINE {
			if !p.advance() {
				return nil, fmt.Errorf("parse block: cannot advance tokens")
			}
		} else {
			b, err := p.parseBlock()
			if err != nil {
				return nil, err
			}
			if b != nil {
				ex.Add(b)
			}
		}
	}
	if p.tok.Type == token.EX_BLOCK {
		//skip closing token
		if !p.advance() {
			return nil, fmt.Errorf("parse example block: cannot advance tokens")
		}
	}
	return &ex, nil
}

func (p *Parser) parseAdmonition() (*ast.Admonition, error) {
	var admonition ast.Admonition
	admonition.Kind = p.tok.Literal
	if !p.advance() {
		return nil, fmt.Errorf("parse admonition error: cannot advance tokens")
	}
	b, err := p.parseParagraph()
	if err != nil {
			return nil, err
		}
	admonition.Content = b

	return &admonition, nil
}

func (p *Parser) parseHeader(id string) (*ast.Header, error) {
	var h ast.Header
	h.Id = id
	h.Level = len(p.tok.Literal)
	if !p.advance() {
		return nil, fmt.Errorf("parseHeader: cannot advance")
	}
	if p.tok.Type == token.STR {
		h.Text = p.tok.Literal
		//p.log.Debug(context.Background(), "parseHeader", slog.F("token", p.tok))

		if !p.advance() {
			return nil, fmt.Errorf("parseHeader: cannot advance")
		}

		return &h, nil
	}

	return nil, fmt.Errorf("invalid header text token: %v", p.tok)
}

func (p *Parser) parseParagraph() (*ast.Paragraph, error) {
	var par ast.Paragraph
	for {
		switch p.tok.Type {
		case token.URL:
			link, err := p.parseLink()
			if err != nil {
				return nil, err
			}
			par.Add(link)
		case token.STR:
			par.Add(&ast.Text{Text: p.tok.Literal})
		case token.INLINE_IMAGE:
			im, err := p.parseInlineImage()
			if err != nil {
				return nil, err
			}
			par.Add(im)
		case token.INT_LINK:
			link, err := p.parseInternalLink()
			if err != nil {
				return nil, err
			}
			par.Add(link)
		}
		// EOF reached
		if !p.advance() {
			break
		}
		// read until double NEWLINE or list marker (which means we're inside the list) or "+" paragraph concatenation
		if p.isParagraphEnd() {
			break
		}
	}
	return &par, nil
}

var imageRE = regexp.MustCompile(`^image::?(.*)\[`)
var includeRE = regexp.MustCompile(`^include::?(.*)\[(.*)\]$`)
var inlineImageRE = regexp.MustCompile(`^image:(.*)\[`)

func (p *Parser) parseImage(options string) (*ast.Image, error) {
	matches := imageRE.FindStringSubmatch(p.tok.Literal)
	if len(matches) != 2 {
		return nil, fmt.Errorf("invalid image literal: %v", p.tok.Literal)
	}
	//skip newline after image
	if !p.advanceMany(2) {
		return nil, fmt.Errorf("parseImage: cannot advance")
	}
	if p.prevTok.Type != token.NEWLINE {
		return nil, fmt.Errorf("parseImage: no NEWLINE after image")
	}
	return &ast.Image{Options: options, Path: matches[1]}, nil
}

//include::RoutingGuide.adoc[leveloffset=+1]
func (p *Parser) parseInclude(options string) (*ast.Document, error) {
	var err error
	matches := includeRE.FindStringSubmatch(p.tok.Literal)
	if len(matches) != 3 {
		return nil, fmt.Errorf("invalid include literal: %v", p.tok.Literal)
	}
	//skip newline after image
	if !p.advanceMany(2) {
		return nil, ErrCannotAdvance
	}
	if p.prevTok.Type != token.NEWLINE {
		return nil, fmt.Errorf("parseImage: no NEWLINE after image")
	}
	file := matches[1]
	if strings.Contains(file, "yandex-counter.adoc") {
		return nil, nil
	}

	opts := strings.Split(matches[2], "=")
	var levelOffset int64
	if len(opts) > 0 && opts[0] == "leveloffset" {
		levelOffset, err = strconv.ParseInt(opts[1], 10, 32)
		if err != nil {
			return nil, err
		}
	}
	var data []byte
	p.log.Info(context.Background(), "parsing include file", slog.F("name", file), slog.F("leveloffset", levelOffset))
	if p.f == nil {
		return nil, fmt.Errorf("no callback, cannot get inlude file content: %v", file)
	}
	data, err = p.f(file)
	if err != nil {
		return nil, err
	}
	parser := New(string(data), p.f, p.log)
	var doc *ast.Document
	doc, err = parser.Parse(file)
	if err != nil {
		return nil, err
	}
	p.log.Info(context.Background(), "parsed include file", slog.F("name", file))
	if levelOffset > 0 {
		doc.Walk(func(b ast.Block, root *ast.Document) bool {
			h, ok := b.(*ast.Header)
			if ok {
				h.Level += int(levelOffset)
			}
			return true
		}, nil)
	}
	//TODO: include could be bookmarked

	//if p.prevTok.Type == token.BOOKMARK && len(doc.Blocks) > 0 {
	//	hdr, ok := doc.Blocks[0].(*ast.Header)
	//	if ok {
	//		hdr.Id = p.prevTok.Literal
	//	}
	//}
	return doc, nil
}


func (p *Parser) parseInlineImage() (*ast.InlineImage, error) {
	matches := inlineImageRE.FindStringSubmatch(p.tok.Literal)
	if len(matches) != 2 {
		return nil, fmt.Errorf("invalid inline image literal: %v", p.tok.Literal)
	}
	//skip to the next token
	if !p.advance() {
		return nil, fmt.Errorf("parseInlineImage: cannot advance")
	}
	return &ast.InlineImage{ Path: matches[1]}, nil
}

//level is a nested list level
func (p *Parser) parseListItem() (*ast.ContainerBlock, error) {
	var item ast.ContainerBlock

l1:
	for {
		switch {
		case p.isDoubleNewline() ||	p.isListMarker() ||	p.isColumn() || p.tok.Type == token.TABLE || p.tok.Type == token.EOF:
			break l1
		case p.tok.Type == token.NEWLINE:
			if !p.advance() { return nil, fmt.Errorf("parse list item: cannot advance tokens") }
		case p.tok.Type == token.CONCAT_PAR:
			//skip newline after CONCAT_PAR
			if !p.advanceMany(2) {
				return nil, fmt.Errorf("parseListItem: cannot advance by 2 elements")
			}

		default:
			//are we inside example block?
			if p.curBlock != nil && p.tok.Type == token.EX_BLOCK {
				_, yes := p.curBlock.(*ast.ExampleBlock)
				if yes {
					break l1
				}
			}

			b, err := p.parseBlock()
			if err != nil {
				return nil, err
			}
			if b != nil {
				item.Add(b)
			}
		}
	}
	return &item, nil
}

/* parseList is called for the 1st list item

ex0:
* item1
** item1.1
*** item 1.1.1
. nested list item 1.1.1.1
** item1.2
* item2

ex1:
* list item 1
. nested numbered list item 1.1
. nested numbered list item 1.2
** nested not-numbered list item 1.1
** nested not-numbered list item 1.2

it means:
· list item 1
	· nested ... 1.1
	· nested ... 1.2
	1. nested ... 1.1
	2. nested ... 1.2

ex2:
. list item 1
* nested list item 1.1
** nested list item 1.1.1
. list item 2

Parsing rules:
1. If list marker == current list marker then: current list item
2. If list marker == (any list marker in the chain of parents): parent list
2. Else: nested list

 */
func (p *Parser) parseList(parent *ast.List) (*ast.List, error) {
	var err error
	var blok ast.Block
	var item *ast.ContainerBlock
	var list ast.List
	//store list marker
	list.Marker = p.tok.Literal
	if strings.HasPrefix(list.Marker, ".") {
		//numbered list
		list.Numbered = true
	}
	list.Parent = parent
	if parent != nil {
		list.Level = parent.Level + 1
	}

	for {
		switch {
		case p.isDoubleNewline() || p.tok.Type == token.EOF ||
				p.tok.Type == token.EX_BLOCK || p.isColumn() || p.tok.Type == token.TABLE:
			//end of the list
			//p.nestedListLevel = 0
			return &list, nil
		case p.isListMarker() && p.tok.Literal == list.Marker:
			//current list item
			if !p.advance() {return nil, fmt.Errorf("parseList: cannot advance")}
			item, err = p.parseListItem()
			if err != nil {
				return nil, err
			}
			list.AddItem(item)
		case p.isListMarker() && list.CheckMarker(p.tok.Literal):
			//parent list item
			return &list, nil
		case p.isListMarker():
			//nested list
			blok, err = p.parseList(&list)

			if err != nil {
					return nil, err
			}
			//p.log.Debug(context.Background(), "nested list parsed", slog.F("list", blok))
			list.LastItem().Add(blok)
		default:
			//error
			return nil, fmt.Errorf("invalid nested list item")
		}
	}

	//return &list, nil
}

func (p *Parser) parseTable(options string) (*ast.Table, error) {
	//skip delimiter + newline tokens
	var t ast.Table
	t.SetOptions(options)

	p.tableFlag = true //when tableFlag == true, paragraph could end at "|" symbol
	defer func() { p.tableFlag = false } ()
	//p.curBlock = &t
	//defer func(old ast.Block) { p.curBlock = old }(p.curBlock)

	if !p.advanceMany(2) {
		return nil, fmt.Errorf("parse table: cannot advance tokens")
	}
	var countColumns = true
	var cell *ast.ContainerBlock //current cell

	for p.tok.Type != token.TABLE && p.tok.Type != token.EOF {
		switch {
		case p.tok.Type == token.COLUMN || p.tok.Type == token.A_COLUMN: //new cell
			if countColumns {
				t.Columns++
			}
			if cell != nil {
				t.AddColumn(cell)
			}
			if !p.advance() {
				return nil, ErrCannotAdvance
			}
			cell = &ast.ContainerBlock{} //current cell

		case p.tok.Type == token.NEWLINE:
			//stop counting at newline after some actual columns, thus "t.Columns>0"
			if countColumns && t.Columns > 0 {
				countColumns = false
			}
			if !p.advance() {
				return nil, ErrCannotAdvance
			}
		case p.tok.Type == token.INDENT:
			if !p.advance() {
				return nil, ErrCannotAdvance
			}
		default:
			b, err := p.parseBlock()
			if err != nil {
				return nil, err
			}
			if cell == nil {
				return nil, fmt.Errorf("parse table: null cell")
			}
			if b != nil {
				cell.Add(b)
			}
		}
	}
	if p.tok.Type == token.TABLE {
		//skip closing token
		if !p.advance() {
			return nil, fmt.Errorf("parse table: cannot advance tokens")
		}
	}
	t.AddColumn(cell)
	return &t, nil
}