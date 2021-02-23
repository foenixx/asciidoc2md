package main

import (
	"asciidoc/ast"
	"io"
	"strings"
)

type Converter struct {
	ImageFolder string
}

func (c *Converter) RenderMarkdown(doc *ast.ContainerBlock, w io.Writer) {
	for _, blok := range doc.Blocks {
		var data string
		switch blok.(type) {
		case *ast.Header:
			data = c.ConvertHeader(blok.(*ast.Header))
		case *ast.ContainerBlock:
			data = c.ConvertParagraph(blok.(*ast.ContainerBlock))
		case *ast.Image:
			data = c.ConvertImage(blok.(*ast.Image))
		case *ast.InlineImage:
			data = c.ConvertInlineImage(blok.(*ast.InlineImage))
		case *ast.HorLine:
			data = c.ConvertHorLine(blok.(*ast.HorLine))
		}
		w.Write([]byte(data))
	}
}


func (c *Converter) ConvertHeader(h *ast.Header) string {
	return strings.Repeat("#", h.Level) + " " + h.Text + "\n\n"
}

func (c *Converter) ConvertParagraph(p *ast.ContainerBlock) string {
	var output string
	for _, b := range p.Blocks {
		switch b.(type) {
		case *ast.Text:
			output += b.(*ast.Text).Text
		case *ast.InlineImage:
			output += c.ConvertInlineImage(b.(*ast.InlineImage))
		}
	}
	output += "\n\n"
	return output
}

func (c *Converter) ConvertImage(p *ast.Image) string {
	return "![](" + c.ImageFolder + p.Path + ")\n\n"
}

func (c *Converter) ConvertInlineImage(p *ast.InlineImage) string {
	return "![](" + p.Path + ")"
}

func (c *Converter) ConvertHorLine(p *ast.HorLine) string {
	return "***\n\n"
}