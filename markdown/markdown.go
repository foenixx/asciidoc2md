package markdown

import (
	"asciidoc2md/ast"
	"asciidoc2md/utils"
	"cdr.dev/slog"
	"context"
	"fmt"
	"io"
	"strings"
)

type GetWriterFunc func(*ast.Header) io.Writer

type Converter struct {
	imageFolder string
	curIndent   string //current indentation level: 2 spaces, 4 spaces, ...
	log         slog.Logger
	writerFunc  GetWriterFunc
	writer      io.Writer
	//writerFile  string
	idMap	map[string]string //header id to file mapping
}

func New(imFolder string, idMap map[string]string, logger slog.Logger, writerFunc GetWriterFunc) *Converter {
	return &Converter{imageFolder: imFolder,
		idMap: idMap,
		log: logger,
		writerFunc: writerFunc}
}

func (c *Converter) RenderMarkdown(doc *ast.Document, w io.Writer) {
	c.writer = w
	//c.writerFile = file
	c.WriteContainerBlock(&doc.ContainerBlock, false)
}

func (c *Converter) WriteList(l *ast.List) {
	//var output strings.Builder
	var m = "* "
	if l.Numbered {
		m = "1. "
	}
	indent := c.curIndent

	for _, i := range l.Items {
		c.curIndent = indent + strings.Repeat(" ", len(m))
		c.WriteString("\n" + indent + m)
		c.WriteContainerBlock(i, false)
		//c.log.Debug(context.Background(), str)
		//c.writer.Write([]byte(str))
	}
	c.curIndent = indent
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
func (c *Converter)	ConvertComplexTable(t *ast.Table) *ast.List {
	var list ast.List
	list.Numbered = false
	list.Marker = "*"
	header := []string{}
	isDefList := t.IsDefList()


	//if !t.Header {
	//	return "header!!!"
	//}
	var par *ast.Paragraph
	var ok bool
	for _, cell := range t.Cells[:t.Columns] {
		if len(cell.Blocks) == 0 {
			//empty header
			header = append(header, "")
			continue
		}
		if par, ok = cell.Blocks[0].(*ast.Paragraph); !ok {
			header = append(header, "HEADER IS NOT A PARAGRAPH!")
		} else {
			header = append(header, c.ConvertParagraph(par))
		}
	}
	i := t.Columns
	for row := 0; row < (len(t.Cells) - t.Columns) / t.Columns; row++ {
		//every row
		rowCont := &ast.ContainerBlock{}

		for col := 0; col < t.Columns; col++ {
			switch {
			case (col == 0 && isDefList) || header[col] == "":
				//first column text becomes a header
				h := strings.TrimSpace(c.ConvertParagraph(t.Cells[i].Blocks[0].(*ast.Paragraph)))
				if h[0] != '`' {
					h = "`" + h + "`"
				}
				rowCont.Add(ast.NewParagraphFromStr(h))
			case col == 1 && isDefList:
				//second column goes without a header
				rowCont.Append(t.Cells[i].Blocks...)
			default:
				if header[col] != "" {
					rowCont.Add(ast.NewParagraphFromStr(fmt.Sprintf("%v:", strings.TrimSpace(header[col]))))
				}
				rowCont.Append(t.Cells[t.Columns*(row+1)+col].Blocks...)
			}
			i++
		}
		list.AddItem(rowCont)
	}
	//c.log.Debug(context.Background(), list.String(""))
	return &list
}

func (c *Converter) WriteTable(t *ast.Table) {
	//var output strings.Builder
	//indent := c.curIndent

	if !t.IsSimple() {
		c.WriteList(c.ConvertComplexTable(t))
		return
	}
	if t.Columns == 0 {
		//return "ZERO COLUMNS"
	}
	t.Header = true
	row := 0
	for i, cell := range t.Cells {
		 if i % t.Columns == 0 {
		 	//new row
		 	row++
		 	c.WriteString(c.curIndent + "| ")
		 }
		 if t.Header && row == 2 {
		 	//let's write header delimiter
		 	t.Header = false //TODO: remove dirty hack
			 c.WriteString(strings.Repeat(" --- |", t.Columns) + "\n" + c.curIndent + "| ")
		 }
		 if len(cell.Blocks) == 0 {
			 c.WriteString(" |")
		 } else {
			 c.WriteParagraph(cell.Blocks[0].(*ast.Paragraph), c.writer)
			 c.WriteString(" |")
		 }
		if i % t.Columns == t.Columns - 1 {
			//last cell of the current column
			c.WriteString("\n")
		}
	}
}

func (c *Converter) WriteBlockTitle(h *ast.BlockTitle, w io.Writer) {
	w.Write([]byte(fmt.Sprintf("_%v_\n", h.Title)))
}

func (c *Converter) WriteHeader(h *ast.Header, w io.Writer) {
	if h.Id != "" {
		w.Write([]byte(fmt.Sprintf(`<a id="%v"></a>` + "\n", h.Id)))
	}
	w.Write([]byte(strings.Repeat("#", h.Level) + " " + h.Text + "\n"))
}

//WriteAdmonition will work only if "Admonition" markdown extension is enabled.
//For details see https://squidfunk.github.io/mkdocs-material/reference/admonitions/.
func (c *Converter) WriteAdmonition(a *ast.Admonition, w io.Writer) {
	//writer == "NOTE:" || writer == "TIP:" || writer == "IMPORTANT:" || writer == "WARNING:" || writer == "CAUTION:":
	var kind string
	if a.Kind == "CAUTION" {
		kind = "danger"
	} else {
		kind = strings.ToLower(a.Kind)
	}
	w.Write([]byte(fmt.Sprintf("!!! %s\n%v    ", kind, c.curIndent)))
	c.WriteParagraph(a.Content, w)
	w.Write([]byte("\n"))
}

func (c *Converter) ConvertParagraph(p *ast.Paragraph) string {
	var res strings.Builder
	c.WriteParagraph(p, &res)
	return res.String()
}

func isPunctuation(s string) bool {
	return utils.RuneIs(rune(s[0]), ',','.',':',';')
}

func (c *Converter) WriteParagraph(p *ast.Paragraph, w io.Writer) {
	var needSpace bool
	for _, b := range p.Blocks {
		switch b.(type) {
		case *ast.Text:
			txt := b.(*ast.Text)
			if needSpace && !isPunctuation(txt.Text){
				w.Write([]byte(" "))
			}
			//converting "*" (asciidoc bold) to "**" (markdown bold)
			//no need to convert asciidoc italic "_" since it's still an italic in markdown
			str := strings.ReplaceAll(txt.Text, "`*", "`")
			str = strings.ReplaceAll(str, "*`", "`")
			str = strings.ReplaceAll(str, "*", "**")
			w.Write([]byte(str))
			needSpace = false
		case *ast.InlineImage:
			c.WriteInlineImage(b.(*ast.InlineImage), w)
			needSpace = true
		case *ast.Link:
			c.WriteLink(b.(*ast.Link),w)
			needSpace = true
		}
	}
}

func (c *Converter) WriteExampleBlock(ex *ast.ExampleBlock) {
	ind := c.curIndent
	c.curIndent += "    "
	c.writer.Write([]byte("!!! example\n"))
	c.WriteContainerBlock(&ex.ContainerBlock, true)
	c.curIndent = ind
}

/*
	case *ast.Header:
		h := blok.(*ast.Header)
		if c.writerFunc != nil {
			newWriter := c.writerFunc(h)
			if newWriter != nil {
				c.writer = newWriter
			}
		}
		data = c.ConvertHeader(h)
	case *ast.ContainerBlock:
		data = c.WriteContainerBlock(blok.(*ast.ContainerBlock), true)
	case *ast.HorLine:
		data = c.ConvertHorLine(blok.(*ast.HorLine))
	case *ast.Table:
		data = c.WriteTable(blok.(*ast.Table))
	}
*/

func (c *Converter) WriteString(s string) error {
	_, err := c.writer.Write([]byte(s))
	return err
}

func (c *Converter) WriteContainerBlock(p *ast.ContainerBlock, firstLineIndent bool)  {
	//var output strings.Builder

	for i, b := range p.Blocks {

		_, isList := b.(*ast.List)
		if i > 0 {
			//write extra newline before every paragraph, except the first one
			c.WriteString("\n")
		}
		if !isList && ((i == 0 && firstLineIndent) || i > 0) {
			c.WriteString(c.curIndent)
		}

		switch b.(type) {
		case *ast.Header:
			h := b.(*ast.Header)
			if c.writerFunc != nil {
				newWriter := c.writerFunc(h)
				if newWriter != nil {
					c.writer = newWriter
					//c.writerFile = newFile
				}
			}
			//converter hack
			if h.Text != "<skip>" {
				c.WriteHeader(h, c.writer)
			}
		case *ast.ContainerBlock:
			c.WriteContainerBlock(b.(*ast.ContainerBlock),firstLineIndent)
		case *ast.HorLine:
			c.WriteHorLine(b.(*ast.HorLine), c.writer)
		case *ast.Table:
			c.WriteTable(b.(*ast.Table))
		case *ast.Image:
			c.WriteImage(b.(*ast.Image), c.writer)
		case *ast.Paragraph:
			c.WriteParagraph(b.(*ast.Paragraph), c.writer)
			c.WriteString("\n")
		case *ast.List:
			c.WriteList(b.(*ast.List))
		case *ast.Admonition:
			c.WriteAdmonition(b.(*ast.Admonition), c.writer)
		case *ast.BlockTitle:
			c.WriteBlockTitle(b.(*ast.BlockTitle), c.writer)
		case *ast.ExampleBlock:
			c.WriteExampleBlock(b.(*ast.ExampleBlock))
		case *ast.SyntaxBlock:
			c.WriteSyntaxBlock(b.(*ast.SyntaxBlock))
		case *ast.Bookmark:
			c.WriteString(fmt.Sprintf(`<a id="%v"></a>`, b.(*ast.Bookmark).Literal))

		default:
			panic("invalid ast block\n" + b.String(""))
		}
	}
}

func (c *Converter) WriteImage(p *ast.Image, w io.Writer) {
	w.Write([]byte(fmt.Sprintf("![](%v)\n", c.imageFolder+ strings.ReplaceAll(p.Path, `\`, `/`))))
}

func (c *Converter) WriteInlineImage(p *ast.InlineImage, w io.Writer) {
	w.Write([]byte("![](" + c.imageFolder+ strings.ReplaceAll(p.Path, `\`, `/`) + ")"))
}

func (c *Converter) WriteHorLine(p *ast.HorLine, w io.Writer) {
	w.Write([]byte("***\n"))
}
//TODO: ссылки на документацию на сайте надо конвертировать:
// например https://mytessa.ru/docs/AdministratorGuide/AdministratorGuide.html#publishDeski[Руководстве администратора]
// <<repair-hosting, Возможных проблем>>
func (c *Converter) WriteLink(l *ast.Link, w io.Writer)  {
	if l.Internal &&  c.idMap != nil {
		file, ok := c.idMap[l.Url]
		if !ok {
			c.log.Error(context.Background(), "cannot find file map for link", slog.F("link", l))
		}
		w.Write([]byte(fmt.Sprintf("[%s](%s#%s)", l.Text, file, l.Url)))
	} else {
		w.Write([]byte(fmt.Sprintf("[%s](%s)", l.Text, l.Url)))
	}

}

func (c *Converter) WriteSyntaxBlock(sb *ast.SyntaxBlock) {
	var str string
	if sb.Literal[len(sb.Literal)-1:] == "\n" {
		//trim last newline
		str = sb.Literal[:len(sb.Literal)-1]
	}
	if str[0] == '\n' {
		//trim first newline
		//c.log.Error(context.Background(), "!!! fuck!!!!")
		str = str[1:]
	}

	if sb.InlineHighlight {
		//there are `pass:quotes[#some_text#]` highlighting
		str = strings.ReplaceAll(str, `pass:quotes[#`, `<span style="background-color: lightyellow">`)
		str = strings.ReplaceAll(str, `#]`, `</span>`)
		c.WriteString(fmt.Sprintf(`<pre><code lang="%s">`, sb.Lang))
		c.WriteString(str)
		c.WriteString("</code></pre>\n")
		return
	}
	str = strings.ReplaceAll(str,"\n", "\n" + c.curIndent + "   ")
	c.WriteString(fmt.Sprintf("``` %s\n%s   %s\n%s```\n", sb.Lang, c.curIndent, str, c.curIndent))
}