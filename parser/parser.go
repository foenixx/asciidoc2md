package parser

import (
	"asciidoc2md/ast"
	"asciidoc2md/lexer"
	"asciidoc2md/token"
	"cdr.dev/slog"
	"context"
	"fmt"
	"regexp"
	"strings"
)

const ILLEGAL = "ILLEGAL"

type Parser struct{
	l       *lexer.Lexer
	tokens  []*token.Token
	next    int          //next token index
	tok     *token.Token //current token
	prevTok *token.Token // previous token
	//nextTok        *token.Token // next token
	//nestedListLevel int
	curBlock ast.Block //block which is being parsed
	log slog.Logger
}

func New(input string, logger slog.Logger) *Parser {
	var p Parser
	p.log = logger
	p.l = lexer.New(input, func(tok *token.Token) {
		//p.log.Debug(context.Background(), "got token", slog.F("token", tok))
		p.tokens = append(p.tokens, tok)
	})
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


func (p *Parser) Parse() (*ast.ContainerBlock, error) {
	var doc ast.ContainerBlock
	p.l.ReadAll()

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
			if b != nil {
				doc.Add(b)
			}
		}
	}
	return &doc, nil
}

func (p *Parser) parseBlock() (ast.Block, error) {

	var options string
	if p.tok.Type == token.BLOCK_OPTS {
		options = p.tok.Literal
		p.log.Debug(context.Background(), "parseBlock: BLOCK_OPTS", slog.F("token", p.tok))
		//skip to the token after newline
		if !p.advanceMany(2) {
			return nil, fmt.Errorf("cannot skip newline: unexpected EOF")
		}
	}

	switch {
	case p.isListMarker():
		l, err := p.parseList(nil)
		if err != nil {
			return nil, err
		}
		return l, nil
	case p.tok.Type == token.BLOCK_TITLE:
		t := ast.BlockTitle{Title: p.tok.Literal}
		if !p.advance() { return nil, fmt.Errorf("parse block title: cannot advance tokens") }
		return &t, nil
	case p.tok.Type == token.HEADER:
		h, err := p.parseHeader()
		if err != nil {
			return nil, err
		}
		return h, nil
	case p.tok.Type == token.STR || p.tok.Type == token.INLINE_IMAGE: //paragraph may begin with image
		//paragraph
		p, err := p.parseParagraph()
		return (ast.Block)(p), err
	case p.tok.Type == token.BLOCK_IMAGE:
		im, err := p.parseImage()
		if err != nil {
			return nil, err
		}
		im.Options = options
		return (ast.Block)(im), nil
	case p.tok.Type == token.HOR_LINE:
		if !p.advance() {
			return nil, fmt.Errorf("cannot advance after HOR_LINE token")
		}
		return &ast.HorLine{}, nil
	case p.tok.Type == token.ADMONITION:
		adn, err := p.parseAdmonition()
		if err != nil {
			return nil, err
		}
		return adn, nil
	case p.tok.Type == token.EX_BLOCK:
		//example block
		//if _, ok := p.curBlock.(*ast.ExampleBlock); ok {
		//	//we're inside parseExampleBlock, just return
		//}
		b, err := p.parseExampleBlock()
		if err != nil {
			return nil, err
		}
		b.Options = options
		return (ast.Block)(b), nil
	}
	return nil, nil
}

func (p *Parser) isDoubleNewline() bool {
	return p.tok.Type == token.NEWLINE && p.prevTok.Type == token.NEWLINE
}

func (p *Parser) isListMarker() bool {
	return p.tok.Type == token.NL_MARK || p.tok.Type == token.L_MARK
}

func (p *Parser) isParagraphEnd() bool {
	return (p.tok.Type == token.NEWLINE && p.prevTok.Type == token.NEWLINE) ||
		p.tok.Type == token.EOF ||
		p.isListMarker() ||
		p.tok.Type == token.CONCAT_PAR ||
		p.tok.Type == token.EX_BLOCK ||
		p.tok.Type == token.QUOTE_BLOCK
}

func (p *Parser) parseExampleBlock() (*ast.ExampleBlock, error) {
	//skip delimiter + newline tokens
	var ex ast.ExampleBlock
	p.curBlock = &ex
	defer func() { p.curBlock = nil }()

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
			ex.Add(b)
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

func (p *Parser) parseHeader() (*ast.Header, error) {
	var h ast.Header
	h.Level = len(p.tok.Literal)
	if !p.advance() {
		return nil, fmt.Errorf("parseHeader: cannot advance")
	}
	if p.tok.Type == token.STR {
		h.Text = p.tok.Literal
		p.log.Debug(context.Background(), "parseHeader", slog.F("token", p.tok))

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
		case token.STR:
			par.Add(&ast.Text{Text: p.tok.Literal})
		case token.INLINE_IMAGE:
			im, err := p.parseInlineImage()
			if err != nil {
				return nil, err
			}
			par.Add(im)
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

var imageRE = regexp.MustCompile(`^image::(.*)\[`)
var inlineImageRE = regexp.MustCompile(`^image:(.*)\[`)

func (p *Parser) parseImage() (*ast.Image, error) {
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
	return &ast.Image{Path: matches[1]}, nil
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
		case p.isDoubleNewline() ||	p.isListMarker() ||	p.tok.Type == token.EOF:
			break l1
		case p.tok.Type == token.NEWLINE:
			if !p.advance() { return nil, fmt.Errorf("parse list item: cannot advance tokens") }
		case p.tok.Type == token.CONCAT_PAR:
			//skip newline after CONCAT_PAR
			if !p.advanceMany(2) {
				return nil, fmt.Errorf("parseListItem: cannot advance by 2 elements")
			}

		default:
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
		case (p.tok.Type == token.NEWLINE && p.prevTok.Type == token.NEWLINE) || p.tok.Type == token.EOF:
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