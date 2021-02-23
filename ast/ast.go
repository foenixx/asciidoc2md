package ast

import (
	"asciidoc/utils"
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


type Header struct {
	Level int
	Text string
}

func (h *Header) String(indent string) string {
	return fmt.Sprintf("\n%sheader: %v, %v", indent, h.Level, h.Text)
}

type List struct {
	Items []*ContainerBlock
	Level int
	Numbered bool
}

func (l *List) String(indent string) string {
	str := strings.Builder{}
	//ind2 := strings.Repeat("  ", l.Level)
	str.WriteString(fmt.Sprintf("\n%slist begin: %v, %v", indent, l.Level, l.Numbered))

	for _, item := range l.Items {
		if item != nil {
			str.WriteString(fmt.Sprintf("\n%sitem:", indent))
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
	return fmt.Sprintf("\n%stext: %v", indent, utils.FirstN(t.Text, 100))
}

type HorLine struct {
}

func (i *HorLine) String(indent string) string {
	return fmt.Sprintf("\n%shor line")
}

type Admonition struct {
	Kind string
	Content *ContainerBlock
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

var _ Block = (*Header)(nil)