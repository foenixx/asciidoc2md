package ast

import (
	"asciidoc2md/token"
	"asciidoc2md/utils"
	"fmt"
	"strings"
)

type Block interface {
	StringWithIndent(indent string) string
	String() string
}

type (
	WalkerFunc func (Block, *Document) bool
	Walker interface {
		Walk(WalkerFunc, *Document) bool
	}
)

type ContainerBlock struct {
	Blocks []Block
}

func (b *ContainerBlock) Add(blok Block) {
	b.Blocks = append(b.Blocks, blok)
}

func (b *ContainerBlock) Append(blok ...Block) {
	b.Blocks = append(b.Blocks, blok...)
}

func (b *ContainerBlock) Walk(f WalkerFunc, root *Document) bool {
	for _, blok := range b.Blocks {
		if !f(blok, root) {
			return false
		}
		wkr, ok := blok.(Walker)
		if ok {
			if !wkr.Walk(f, root) {
				return false
			}
		}
	}
	return true
}
func (b *ContainerBlock) String() string {
	return b.StringWithIndent("")
}

func (b *ContainerBlock) StringWithIndent(indent string) string {
	str := strings.Builder{}
	str.WriteString(fmt.Sprintf("\n%scontainer block:", indent))
	for _, blok := range b.Blocks {
		if blok != nil {
			str.WriteString(blok.StringWithIndent(indent + "  "))
		} else {
			str.WriteString(fmt.Sprintf("\n%snil block", "  " + indent))
		}
	}
	return str.String()
}

type Document struct {
	ContainerBlock
	Name string //adoc file name, empty for root doc
}

func (d *Document) StringWithIndent(indent string) string {
	s := d.ContainerBlock.StringWithIndent(indent)
	return strings.Replace(s, "container block", "document", 1)
}

func (d *Document) Walk(f WalkerFunc, root *Document) bool {
	return d.ContainerBlock.Walk(f, d)
}

var _ Walker = (*Document)(nil)

type Paragraph struct {
	ContainerBlock
}

var _ Walker = (*Paragraph)(nil)
/*
func (d *Document) Walk(f func(b Block)) {
	d.ContainerBlock.Walk(f)
}
*/
func (b *Paragraph) StringWithIndent(indent string) string {
	s := b.ContainerBlock.StringWithIndent(indent)
	return strings.Replace(s, "container block", "paragraph", 1)
}

func (b *Paragraph)	IsSingleText() bool {
	if len(b.Blocks) != 1 {
		return false
	}
	_, ok := b.Blocks[0].(*Text)
	if ok {
		return true
	}
	return false
}

/*
func (b *Paragraph) String() string {
	return b.StringWithIndent("")
}

 */

func NewParagraphFromStr(s string) *Paragraph {
	par := Paragraph{}
	par.ContainerBlock.Add(&Text{Text: s})
	return &par
}

type ExampleBlock struct {
	ContainerBlock
	Options string
	Collapsible bool
	Delim *token.Token
}

var _ Walker = (*ExampleBlock)(nil)

func (ex *ExampleBlock)	ParseOptions(opts string) {
	ex.Options = opts
	if strings.Contains(opts, "collapsible") {
		ex.Collapsible = true
	}
}

func (ex *ExampleBlock) StringWithIndent(indent string) string {
	s := ex.ContainerBlock.StringWithIndent(indent)
	return strings.Replace(s, "container", "example", 1)
}



type Header struct {
	Level int
	Text string
	Id string
	Float bool //not a header, just formatted like a header text
	Options string
}

func (h *Header) StringWithIndent(indent string) string {
	var id string
	if h.Id != "" {
		id = " [" + h.Id + "]"
	}
	var opts string
	if h.Options != "" {
		opts = ", " + h.Options
	}
	return fmt.Sprintf("\n%sheader: %v, %v%v%v", indent, h.Level, h.Text, id, opts)
}

func (h *Header) String() string {
	return h.StringWithIndent("")
}

type BlockTitle struct {
	Title string
}

func (t *BlockTitle) StringWithIndent(indent string) string {
	return fmt.Sprintf("\n%sblock title: %v", indent, t.Title)
}

func (t *BlockTitle) String() string {
	return t.StringWithIndent("")
}


type List struct {
	Items []*ContainerBlock
	Parent *List
	Marker string
	Level int
	Numbered bool
	Definition bool
}

func (l *List) Walk(f WalkerFunc, root *Document) bool {
	for _, item := range l.Items {
		if !item.Walk(f, root) {
			return false
		}
	}
	return true
}

func (l *List) CheckMarker(m string) bool {
	if l == nil {
		//checking for nil receiver cause it simplifies handling for nil if smb does "somelist.Parent.CheckMarker(...)"
		return false
	}

	if m == l.Marker {
		return true
	}

	if l.Parent == nil {
		return false
	}
	return l.Parent.CheckMarker(m)
}

func (l *List) StringWithIndent(indent string) string {
	str := strings.Builder{}
	//ind2 := strings.Repeat("  ", l.Level)
	str.WriteString(fmt.Sprintf("\n%slist begin: (%v/%v/%v)", indent, l.Level, l.Numbered, l.Marker))

	for i, item := range l.Items {
		if item != nil {
			if l.Numbered {
				str.WriteString(fmt.Sprintf("\n%sitem %v:", indent, i + 1))
			} else {
				str.WriteString(fmt.Sprintf("\n%sitem:", indent))
			}
			str.WriteString(item.StringWithIndent(indent + "  "))
			//str.WriteString("\n")
		} else {
			str.WriteString(fmt.Sprintf("\n%sitem: nil", indent))
		}
	}
	str.WriteString(fmt.Sprintf("\n%slist end", indent))
	return str.String()
}

func (l *List) String() string {
	return l.StringWithIndent("")
}


func (l *List) AddItem(item *ContainerBlock) {
	l.Items = append(l.Items, item)
}

func (l *List) LastItem() *ContainerBlock {
	if len(l.Items) == 0 {
		return nil
	}
	return l.Items[len(l.Items) - 1]
}

type SyntaxBlock struct {
	Options string
	Literal string
	Lang string
	InlineHighlight bool
}

func (sb *SyntaxBlock) SetOptions(options string) {
	switch {
	case strings.Contains(options, "xml"):
		sb.Lang = "json"
	case strings.Contains(options, "c#") || strings.Contains(options, "csharp"):
		sb.Lang = "c#"
	case strings.Contains(options, "js") || strings.Contains(options, "javascript"):
		sb.Lang = "js"
	case strings.Contains(options, "ts") || strings.Contains(options, "typescript"):
		sb.Lang = "ts"
	case strings.Contains(options, "sql"):
		sb.Lang = "sql"
	case strings.Contains(options, "json"):
		sb.Lang = "json"
	}
	if strings.Contains(options, "macros+") {
		// there are `pass:quotes[#some_text#]` highlighting
		sb.InlineHighlight = true
	}
}

func (sb *SyntaxBlock) StringWithIndent(indent string) string {
	return fmt.Sprintf("\n%ssyntax block: %s", indent, utils.ShortenString(sb.Literal, 30, 30))
}

func (sb *SyntaxBlock) String() string {
	return sb.StringWithIndent("")
}

type Image struct {
	Path string
	Options string
}

func (i *Image) StringWithIndent(indent string) string {
	return fmt.Sprintf("\n%simage: %v", indent, i.Path)
}

func (i *Image) String() string {
	return i.StringWithIndent("")
}

type InlineImage struct {
	Path string
	Options string
}

func (i *InlineImage) StringWithIndent(indent string) string {
	return fmt.Sprintf("\n%sinline image: %v", indent, i.Path)
}

func (i *InlineImage) String() string {
	return i.StringWithIndent("")
}


type Text struct {
	Text string
}


func (t *Text) StringWithIndent(indent string) string {
	return fmt.Sprintf("\n%stext: %v", indent, utils.ShortenString(t.Text, 30, 30))
}

func (t *Text) String() string {
	return t.StringWithIndent("")
}

type HorLine struct {
}

func (i *HorLine) StringWithIndent(indent string) string {
	return fmt.Sprintf("\n%shor line")
}

func (i *HorLine) String() string {
	return i.StringWithIndent("")
}

type Admonition struct {
	Kind string
	Content *ContainerBlock
}

func (l *Admonition) Walk(f WalkerFunc, root *Document) bool {
	return l.Content.Walk(f, root)
}

func (a *Admonition) StringWithIndent(indent string) string {
	var cStr string
	if a.Content == nil {
		cStr = "nil"
	} else {
		cStr = a.Content.StringWithIndent(indent + "  ")
	}
	return fmt.Sprintf("\n%sadmonition: %s%s", indent, a.Kind, cStr)
}

func (a *Admonition) String() string {
	return a.StringWithIndent("")
}

type Table struct {
	Header bool
	Options string
	Columns int
	Cells   []*ContainerBlock
}

func (t *Table) Walk(f WalkerFunc, root *Document) bool {
	for _, cell := range t.Cells {
		if !cell.Walk(f, root) {
			return false
		}
	}
	return true
}

func (t *Table) SetOptions(options string) {
	t.Options = options
	if strings.Contains(t.Options, "header") {
		t.Header = true
	}
}

func (t *Table) AddColumn(c *ContainerBlock) {
	t.Cells = append(t.Cells, c)
}

func (t *Table) StringWithIndent(indent string) string {
	str := strings.Builder{}
	//ind2 := strings.Repeat("  ", l.Level)
	str.WriteString(fmt.Sprintf("\n%stable begin: %v cols", indent, t.Columns))
	if t.IsSimple() {
		str.WriteString(" (simple)")
	}
	if t.IsDefList() {
		str.WriteString(" (not-so-simple)")
	}

	for _, cell := range t.Cells {
		if cell != nil {
			str.WriteString(fmt.Sprintf("\n%scell:", indent))
			str.WriteString(cell.StringWithIndent(indent + "  "))
			//str.WriteString("\n")
		} else {
			str.WriteString(fmt.Sprintf("\n%scell: nil", indent))
		}
	}
	str.WriteString(fmt.Sprintf("\n%stable end", indent))
	return str.String()
}

func (t *Table) String() string {
	return t.StringWithIndent("")
}

//  IsSimple checks if every cell is a single text paragraph.
func (t *Table) IsSimple() bool {
	for _, c := range t.Cells {
		switch len(c.Blocks) {
		case 0:
			continue
		case 1:
			if _, ok := c.Blocks[0].(*Paragraph); !ok {
				return false
			}
		default:
			return false
		}
	}
	return true
}

//  IsDefList checks if every cell in the first column is a single text paragraph.
func (t *Table) IsDefList() bool {
	for i, c := range t.Cells {
		if i % t.Columns > 0 {
			//only first column is taken into consideration
			continue
		}
		switch len(c.Blocks) {
		case 0:
			continue
		case 1:
			if _, ok := c.Blocks[0].(*Paragraph); !ok {
				return false
			}
		default:
			return false
		}
	}
	return true
}

type Bookmark struct {
	Literal string
}

func (b *Bookmark) StringWithIndent(indent string) string {
	return fmt.Sprintf("\n%sbookmark: %s", indent, b.Literal)
}

func (b *Bookmark) String() string {
	return b.StringWithIndent("")
}

type Link struct {
	Url string
	Text string
	Internal bool
}

func (l *Link) StringWithIndent(indent string) string {
	return fmt.Sprintf("\n%slink: (%v,%s,%s)", indent, l.Internal, l.Text, l.Url)
}

func (l *Link) String() string {
	return l.StringWithIndent("")
}

var _ Block = (*Link)(nil)
var _ Block = (*Header)(nil)