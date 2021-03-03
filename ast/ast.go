package ast

import (
	"asciidoc2md/utils"
	"fmt"
	"strings"
)

type Block interface {
	String(indent string) string
}

type ContainerBlock struct {
	Blocks []Block
}

func (b *ContainerBlock) Add(blok Block) {
	b.Blocks = append(b.Blocks, blok)
}

func (b *ContainerBlock) Append(blok ...Block) {
	b.Blocks = append(b.Blocks, blok...)
}

func (b *ContainerBlock) String(indent string) string {
	str := strings.Builder{}
	str.WriteString(fmt.Sprintf("\n%scontainer block:", indent))
	for _, blok := range b.Blocks {
		if blok != nil {
			str.WriteString(blok.String(indent + "  "))
		} else {
			str.WriteString(fmt.Sprintf("\n%snil block", "  " + indent))
		}
	}
	return str.String()
}

type Document struct {
	ContainerBlock
}

func (b *Document) String(indent string) string {
	s := b.ContainerBlock.String(indent)
	return strings.Replace(s, "container block", "document", 1)
}

type Paragraph struct {
	ContainerBlock
}

func (b *Paragraph) String(indent string) string {
	s := b.ContainerBlock.String(indent)
	return strings.Replace(s, "container block", "paragraph", 1)
}

func NewParagraphFromStr(s string) *Paragraph {
	par := Paragraph{}
	par.ContainerBlock.Add(&Text{Text: s})
	return &par
}

type ExampleBlock struct {
	ContainerBlock
	Options string
}

func (ex *ExampleBlock) String(indent string) string {
	s := ex.ContainerBlock.String(indent)
	return strings.Replace(s, "container", "example", 1)
}

type Header struct {
	Level int
	Text string
}

func (h *Header) String(indent string) string {
	return fmt.Sprintf("\n%sheader: %v, %v", indent, h.Level, h.Text)
}

type BlockTitle struct {
	Title string
}

func (t *BlockTitle) String(indent string) string {
	return fmt.Sprintf("\n%sblock title: %v", indent, t.Title)
}

type List struct {
	Items []*ContainerBlock
	Parent *List
	Marker string
	Level int
	Numbered bool
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

func (l *List) String(indent string) string {
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
			str.WriteString(item.String(indent + "  "))
			//str.WriteString("\n")
		} else {
			str.WriteString(fmt.Sprintf("\n%sitem: nil", indent))
		}
	}
	str.WriteString(fmt.Sprintf("\n%slist end", indent))
	return str.String()
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
}

func (sb *SyntaxBlock) String(indent string) string {
	return fmt.Sprintf("\n%ssyntax block: %s", indent, utils.ShortenString(sb.Literal, 30, 30))
}

type Image struct {
	Path string
	Options string
}

func (i *Image) String(indent string) string {
	return fmt.Sprintf("\n%simage: %v", indent, i.Path)
}

type InlineImage struct {
	Path string
	Options string
}

func (i *InlineImage) String(indent string) string {
	return fmt.Sprintf("\n%sinline image: %v", indent, i.Path)
}


type Text struct {
	Text string
}


func (t *Text) String(indent string) string {
	return fmt.Sprintf("\n%stext: %v", indent, utils.ShortenString(t.Text, 30, 30))
}

type HorLine struct {
}

func (i *HorLine) String(indent string) string {
	return fmt.Sprintf("\n%shor line")
}

type Admonition struct {
	Kind string
	Content *Paragraph
}

func (a *Admonition) String(indent string) string {
	var cStr string
	if a.Content == nil {
		cStr = "nil"
	} else {
		cStr = a.Content.String(indent + "  ")
	}
	return fmt.Sprintf("\n%sadmonition: %s%s", indent, a.Kind, cStr)
}

type Table struct {
	Header bool
	Options string
	Columns int
	Cells   []*ContainerBlock
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

func (t *Table) String(indent string) string {
	str := strings.Builder{}
	//ind2 := strings.Repeat("  ", l.Level)
	str.WriteString(fmt.Sprintf("\n%stable begin: %v cols", indent, t.Columns))
	if t.IsSimple() {
		str.WriteString(" (simple)")
	}
	for _, cell := range t.Cells {
		if cell != nil {
			str.WriteString(fmt.Sprintf("\n%scell:", indent))
			str.WriteString(cell.String(indent + "  "))
			//str.WriteString("\n")
		} else {
			str.WriteString(fmt.Sprintf("\n%scell: nil", indent))
		}
	}
	str.WriteString(fmt.Sprintf("\n%stable end", indent))
	return str.String()
}

//  IsSimple checks if every cell is a single text paragraph.
func (t *Table) IsSimple() bool {
	for _, c := range t.Cells {
		if len(c.Blocks) != 1 {
			return false
		}
		if _, ok := c.Blocks[0].(*Paragraph); !ok {
			return false
		}
	}
	return true
}

type Bookmark struct {
	Literal string
}

func (b *Bookmark) String(indent string) string {
	return fmt.Sprintf("\n%sbookmark: %s", indent, b.Literal)
}


var _ Block = (*Header)(nil)