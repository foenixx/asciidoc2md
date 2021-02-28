package markdown

import (
	"asciidoc2md/ast"
	"cdr.dev/slog"
	"context"
	"fmt"
	"io"
	"strings"
)

type Converter struct {
	imageFolder string
	curIndent   string //current indentation level: 2 spaces, 4 spaces, ...
	log         slog.Logger
}

func New(imFolder string, logger slog.Logger) *Converter {
	return &Converter{imageFolder: imFolder, log: logger}
}

func (c *Converter) RenderMarkdown(doc *ast.ContainerBlock, w io.Writer) {
	for i, blok := range doc.Blocks {
		var data string
		//write extra newline before every block except the first one
		if i > 0 {
			w.Write([]byte("\n"))
		}
		switch blok.(type) {
		case *ast.List:
			data = c.ConvertList(blok.(*ast.List)) //+ "\n\n"
			c.log.Debug(context.Background(), data)
		case *ast.Paragraph:
			data = c.ConvertParagraph(blok.(*ast.Paragraph)) + "\n"
		case *ast.BlockTitle:
			data = c.ConvertBlockTitle(blok.(*ast.BlockTitle))
		case *ast.Header:
			data = c.ConvertHeader(blok.(*ast.Header))
		case *ast.ContainerBlock:
			data = c.ConvertContainerBlock(blok.(*ast.ContainerBlock), true)
		case *ast.Image:
			data = c.ConvertImage(blok.(*ast.Image))
		case *ast.InlineImage:
			data = c.ConvertInlineImage(blok.(*ast.InlineImage))
		case *ast.HorLine:
			data = c.ConvertHorLine(blok.(*ast.HorLine))
		case *ast.Admonition:
			data = c.ConvertAdmonition(blok.(*ast.Admonition)) + "\n\n"
		case *ast.ExampleBlock:
			data = c.ConvertExampleBlock(blok.(*ast.ExampleBlock))
		case *ast.Table:
			data = c.ConvertTable(blok.(*ast.Table))
		}
		w.Write([]byte(data))
	}
}

func (c *Converter) ConvertList(l *ast.List) string {
	var output strings.Builder
	var m = "* "
	if l.Numbered {
		m = "1. "
	}
	indent := c.curIndent

	for _, i := range l.Items {
		c.curIndent = indent + strings.Repeat(" ", len(m))
		output.WriteString(indent + m)
		str := c.ConvertContainerBlock(i, false)
		//c.log.Debug(context.Background(), str)
		output.WriteString(str)
	}
	c.curIndent = indent
	return output.String()
}

// ConvertComplexTable converts complex table into a list.
// For example, if input table has 3 columns, then output list would be:
//  * _col1 header:_ (like italic)
//    col1 text
//    _col2 header:_
//    col2 text
//    _col3 header:_
//    col3 text
//
func (c *Converter)	ConvertComplexTable(t *ast.Table) string {
	var list ast.List
	list.Numbered = false
	list.Marker = "*"
	header := []string{}

	if !t.Header {
		return "header!!!"
	}
	var par *ast.Paragraph
	var ok bool
	for _, cell := range t.Cells[:t.Columns] {
		if par, ok = cell.Blocks[0].(*ast.Paragraph); !ok {
			header = append(header, "HEADER IS NOT A PARAGRAPH!")
		} else {
			header = append(header, c.ConvertParagraph(par))
		}
	}
	for row := 0; row < (len(t.Cells) - t.Columns) / t.Columns; row++ {
		//every row
		rowCont := &ast.ContainerBlock{}
		for col := 0; col < t.Columns; col++ {
			//every column
			rowCont.Add(ast.NewParagraphFromStr(fmt.Sprintf("_%v_:", header[col])))
			rowCont.Append(t.Cells[t.Columns * (row +1) + col].Blocks...)

		}
		list.AddItem(rowCont)
	}
	return c.ConvertList(&list)
}

func (c *Converter) ConvertTable(t *ast.Table) string {
	var output strings.Builder
	//indent := c.curIndent

	if !t.IsSimple() {
		return c.ConvertComplexTable(t)
	}
	if t.Columns == 0 {
		return "ZERO COLUMNS"
	}
	t.Header = true
	row := 0
	for i, cell := range t.Cells {
		 if i % t.Columns == 0 {
		 	//new row
		 	row++
		 	output.WriteString(c.curIndent + "| ")
		 }
		 if t.Header && row == 2 {
		 	//let's write header delimiter
		 	t.Header = false //TODO: remove dirty hack
		 	output.WriteString(strings.Repeat(" --- |", t.Columns) + "\n" + c.curIndent + "| ")
		 }
		output.WriteString(c.ConvertParagraph(cell.Blocks[0].(*ast.Paragraph)) + " |")
		if i % t.Columns == t.Columns - 1 {
			//last cell of the current column
			output.WriteString("\n")
		}
	}
	//c.curIndent = indent
	return output.String()
}

func (c *Converter) ConvertBlockTitle(h *ast.BlockTitle) string {
	return fmt.Sprintf("_%v_\n", h.Title)
}

func (c *Converter) ConvertHeader(h *ast.Header) string {
	return strings.Repeat("#", h.Level) + " " + h.Text + "\n"
}

//ConvertAdmonition will work only if "Admonition" markdown extension is enabled.
//For details see https://squidfunk.github.io/mkdocs-material/reference/admonitions/.
func (c *Converter) ConvertAdmonition(a *ast.Admonition) string {
	//w == "NOTE:" || w == "TIP:" || w == "IMPORTANT:" || w == "WARNING:" || w == "CAUTION:":
	return fmt.Sprintf("!!! note\n%v    %v\n", c.curIndent, c.ConvertParagraph(a.Content))
}

func (c *Converter) ConvertParagraph(p *ast.Paragraph) string {
	var output strings.Builder

	for _, b := range p.Blocks {
		switch b.(type) {
		case *ast.Text:
			output.WriteString(b.(*ast.Text).Text)
		case *ast.InlineImage:
			output.WriteString(c.ConvertInlineImage(b.(*ast.InlineImage)))
		}

	}
	//output.WriteString("\n")
	//return fmt.Sprintf("\n%v%v\n", c.curIndent, output.String())
	return output.String()
}

func (c *Converter) ConvertExampleBlock(ex *ast.ExampleBlock) string {
	ind := c.curIndent
	c.curIndent += "    "
	s := "!!! example\n" + c.ConvertContainerBlock(&ex.ContainerBlock, true)
	c.curIndent = ind
	return s
}

func (c *Converter) ConvertContainerBlock(p *ast.ContainerBlock, firstLineIndent bool) string {
	var output strings.Builder

	for i, b := range p.Blocks {
		_, isList := b.(*ast.List)
		//switch b.(type) {
		////case *ast.List:
		//	//do nothing
		//default:
			if i > 0 {
				//write extra newline before every paragraph, except the first one
				output.WriteString("\n")
			}
			if !isList && ((i == 0 && firstLineIndent) || i > 0) {
				output.WriteString(c.curIndent)
			}
		//}
		switch b.(type) {
//		case *ast.Text:
//			output.WriteString(b.(*ast.Text).Text)
		case *ast.Image:
			output.WriteString(c.ConvertImage(b.(*ast.Image)))
		case *ast.Paragraph:
			output.WriteString(c.ConvertParagraph(b.(*ast.Paragraph)) + "\n")
//		case *ast.ContainerBlock:
//			output.WriteString(c.ConvertContainerBlock(b.(*ast.ContainerBlock)))
		case *ast.List:
			output.WriteString(c.ConvertList(b.(*ast.List)))
		case *ast.Admonition:
			output.WriteString(c.ConvertAdmonition(b.(*ast.Admonition)))
		case *ast.BlockTitle:
			output.WriteString(c.ConvertBlockTitle(b.(*ast.BlockTitle)))
		case *ast.ExampleBlock:
			output.WriteString(c.ConvertExampleBlock(b.(*ast.ExampleBlock)))
		}
	}
	return output.String()
}

func (c *Converter) ConvertImage(p *ast.Image) string {
	return fmt.Sprintf("![](%v)\n", c.imageFolder+ p.Path)
}

func (c *Converter) ConvertInlineImage(p *ast.InlineImage) string {
	return "![](" + p.Path + ")"
}

func (c *Converter) ConvertHorLine(p *ast.HorLine) string {
	return "***\n"
}