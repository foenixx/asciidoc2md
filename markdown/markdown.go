package markdown

import (
	"asciidoc2md/ast"
	"asciidoc2md/token"
	"asciidoc2md/utils"
	"cdr.dev/slog"
	"context"
	"fmt"
	"io"
	"regexp"
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
	idMap	map[string]string//header id to file mapping
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
	c.WriteDocument(doc)
}

func (c *Converter) WriteDocument(doc *ast.Document) {
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
			header = append(header, c.ConvertParagraph(par, true))
		}
	}
	i := t.Columns
	for row := 0; row < (len(t.Cells) - t.Columns) / t.Columns; row++ {
		//every row
		rowCont := &ast.ContainerBlock{}

		for col := 0; col < t.Columns; col++ {
			switch {
			case col == 0 && isDefList:
				//first column text becomes a header
				//c.log.Info(context.Background(), "cell", slog.F("h", t.Cells[i].StringWithIndent("")))
				firstPar := t.Cells[i].Blocks[0].(*ast.Paragraph)
				if firstPar.IsSingleText() {
					h := strings.TrimSpace(c.ConvertParagraph(firstPar, true))
					//c.log.Info(context.Background(), "header", slog.F("h", h))
					if h != "" && !utils.RuneIs(rune(h[0]), '`', '*') {
						h = "`" + h + "`"
					}
					rowCont.Add(ast.NewParagraphFromStr(h))
				} else {
					rowCont.Add(firstPar)
				}
			case col == 1 && t.Columns == 2:
				//second column goes without a header (only for tables with 2 columns)
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
	//c.log.Debug(context.Background(), list.StringWithIndent(""))
	return &list
}

func (c *Converter) WriteTable(t *ast.Table) {
	//var output strings.Builder
	//indent := c.curIndent

	if !t.IsSimple() {
		c.WriteList(c.ConvertComplexTable(t))
		return
	}
	if t.Columns == 1 && len(t.Cells) == 1 {
		//single cell table without header -> convert to admonition
		adm := ast.Admonition{}
		adm.Content = t.Cells[0]
		adm.Kind = "info"
		c.WriteAdmonition(&adm)
		return
	}

	if t.Columns == 0 {
		//return "ZERO COLUMNS"
	}
	t.Header = true
	row := 0
	col := 1
	for i, cell := range t.Cells {
		 if i % t.Columns == 0 {
		 	//new row
		 	row++
		 	col = 1
		 	c.WriteString(c.curIndent + "| ")
		 } else {
		 	col++
		 }
		 if t.Header && row == 2 {
		 	//let's write header delimiter
		 	t.Header = false //TODO: remove dirty hack
			 c.WriteString(strings.Repeat(" --- |", t.Columns) + "\n" + c.curIndent + "| ")
		 }
		 //cell can be empty
		 if len(cell.Blocks) > 0 {
			 val := c.ConvertParagraph(cell.Blocks[0].(*ast.Paragraph), false)
			 //expand first column
			 if t.Header && row == 1 && col == 1 && t.Columns < 5 {
				 //https://stackoverflow.com/a/57420043
				 val = fmt.Sprintf(`<div style="width:13em">%s</div>`, val)
			 }
			 c.WriteString(val)
		 }
		 c.WriteString(" |")

		 if i % t.Columns == t.Columns - 1 {
			//last cell of the current column
			c.WriteString("\n")
		 }
	}
}

func (c *Converter) WriteBlockTitle(h *ast.BlockTitle, w io.Writer) {
	w.Write([]byte(fmt.Sprintf("_%v_\n", fixText(h.Title))))
}

func (c *Converter) WriteHeader(h *ast.Header, w io.Writer) {
	if h.Id != "" {
		w.Write([]byte(fmt.Sprintf(`<a id="%v"></a>` + "\n", h.Id)))
	}
	if h.Float {
		//render float headers as italic text
		w.Write([]byte("_" + h.Text + "_\n"))
		return
	}
	w.Write([]byte(strings.Repeat("#", h.Level) + " " + h.Text + "\n"))
}

//WriteAdmonition will work only if "Admonition" markdown extension is enabled.
//For details see https://squidfunk.github.io/mkdocs-material/reference/admonitions/.
func (c *Converter) WriteAdmonition(a *ast.Admonition) {
	//writer == "NOTE:" || writer == "TIP:" || writer == "IMPORTANT:" || writer == "WARNING:" || writer == "CAUTION:":
	var kind string
	if a.Kind == "CAUTION" {
		kind = "danger"
	} else {
		kind = strings.ToLower(a.Kind)
	}
	c.WriteString(fmt.Sprintf("!!! %s\n%v    ", kind, c.curIndent))
	c.WriteContainerBlock(a.Content, false)
	//c.WriteParagraph(a.Content, false, w)
	c.WriteString("\n")
}

func (c *Converter) ConvertParagraph(p *ast.Paragraph, noFormatFix bool) string {
	var res strings.Builder
	c.WriteParagraph(p, noFormatFix, &res)
	return res.String()
}

// This regexp is only taking into account backticks at the word boundary,
// that is "ab`cd" is ignored while "ab `cd` ef" is not.
var backticksRE = regexp.MustCompile(`\B(\x60)[^\x60]*(\x60)\B`)
// "`*mono and bold text*`"
var monoboldRE = regexp.MustCompile(`^\x60\*|\*\x60$`)
// "`+++strange formatting+++`"
var strangeFormatRE = regexp.MustCompile(`^\x60\+{3}|\+{3}\x60$`)
var checkedRE = regexp.MustCompile(`^\[\*\]`)
// match only single stars "*" at word boundary and ignore double stars "**"
var boldRE = regexp.MustCompile(`([^\*]|^)\B\*\b|\b\*\B([^\*]|$)`)
var sharpRE = regexp.MustCompile(`#(\s)`) //leave double stars "**" as-is
var hardBreakRE = regexp.MustCompile(`\s\+$`)
var smallTextRE = regexp.MustCompile(`\[small]#(.*?)#`)

func fixString(s string, backticked bool) string {
	// fix "`*monospace and bold*`" since it isn't allowed in markdown
	// if strings.HasPrefix(s, "`*") && strings.HasSuffix(s, "")
	s = monoboldRE.ReplaceAllLiteralString(s, "`")
	s = strangeFormatRE.ReplaceAllLiteralString(s, "`")
	if !backticked {
		// fix checked lists "[*]" -> "[x]"
		s = checkedRE.ReplaceAllLiteralString(s, "[x]")
		// converting "*" (asciidoc bold) to "**" (markdown bold)
		// no need to convert asciidoc italic "_" since it's still an italic in markdown
		//s = boldRE.ReplaceAllLiteralString(s, "**")
		s = boldRE.ReplaceAllString(s, "$1**$2")
		s = sharpRE.ReplaceAllString(s, `\#$1`)
		s = strings.ReplaceAll(s, "<", "&lt;")
		s = strings.ReplaceAll(s, ">", "&gt;")
	}
	return s
}

func fixText(s string) string {
	// asciidoc magic "Section1.Field1\=>Section2.Field2"
	s = strings.ReplaceAll(s, `\->`, `->`)
	s = strings.ReplaceAll(s, `\=>`, `=>`)
	// removing "[small]#small text# magic"
	s = smallTextRE.ReplaceAllString(s, "$1")
	var fixed = strings.Builder{}
	indices := append(append([][]int{}, backticksRE.FindAllStringIndex(s, -1)...), []int{len(s),-1})
	beg := 0
	var s1, s2 string
	for _, ind := range indices {
		s1 = s[beg:ind[0]]
		if len(s1) > 0 {
			//fmt.Printf("'%s'\n", s1)
			fixed.WriteString(fixString(s1, false))
		}
		if ind[1] != -1 {
			s2 = s[ind[0]:ind[1]]
			fixed.WriteString(fixString(s2, true))
			//fmt.Printf("'%s'\n", s2)
		}
		beg = ind[1]
	}
	return hardBreakRE.ReplaceAllString(fixed.String(), `</br>`)
}

func (c *Converter) WriteParagraph(p *ast.Paragraph, noFormatFix bool, w io.Writer) {
	for _, b := range p.Blocks {
		switch b.(type) {
		case *ast.Text:
			txt := b.(*ast.Text)
			str := txt.Text
			if !noFormatFix {
				str = fixText(str)
			}
			w.Write([]byte(str))
		case *ast.InlineImage:
			c.WriteInlineImage(b.(*ast.InlineImage), w)
		case *ast.Link:
			c.WriteLink(b.(*ast.Link),w)
		}
	}
}

func (c *Converter) WriteExampleBlock(ex *ast.ExampleBlock) {
	ind := c.curIndent
	c.curIndent += "    "
	if ex.Collapsible {
		c.writer.Write([]byte("???"))
	} else {
		c.writer.Write([]byte("!!!"))
	}
	if ex.Delim.Type == token.EX_BLOCK {
		c.writer.Write([]byte(" example\n"))
	} else {
		c.writer.Write([]byte(" info\n"))
	}
	c.WriteContainerBlock(&ex.ContainerBlock, true)
	c.curIndent = ind
}


func (c *Converter) WriteString(s string) error {
	_, err := c.writer.Write([]byte(s))
	return err
}

func (c *Converter) WriteContainerBlock(p *ast.ContainerBlock, firstLineIndent bool)  {
	//var output strings.Builder

	for i, b := range p.Blocks {

		_, isList := b.(*ast.List)
		_, isTable := b.(*ast.Table)
		if i > 0 {
			//write extra newline before every paragraph, except the first one
			c.WriteString("\n")
		}
		if !isList && !isTable && ((i == 0 && firstLineIndent) || i > 0) {
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
		case *ast.Document:
			//include
			c.WriteDocument(b.(*ast.Document))
		case *ast.ContainerBlock:
			c.WriteContainerBlock(b.(*ast.ContainerBlock),firstLineIndent)
		case *ast.HorLine:
			c.WriteHorLine(b.(*ast.HorLine), c.writer)
		case *ast.Table:
			c.WriteTable(b.(*ast.Table))
		case *ast.Image:
			c.WriteImage(b.(*ast.Image), c.writer)
		case *ast.Paragraph:
			c.WriteParagraph(b.(*ast.Paragraph), false, c.writer)
			c.WriteString("\n")
		case *ast.List:
			c.WriteList(b.(*ast.List))
		case *ast.Admonition:
			c.WriteAdmonition(b.(*ast.Admonition))
		case *ast.BlockTitle:
			c.WriteBlockTitle(b.(*ast.BlockTitle), c.writer)
		case *ast.ExampleBlock:
			c.WriteExampleBlock(b.(*ast.ExampleBlock))
		case *ast.SyntaxBlock:
			c.WriteSyntaxBlock(b.(*ast.SyntaxBlock))
		case *ast.Bookmark:
			c.WriteString(fmt.Sprintf(`<a id="%v"></a>`, b.(*ast.Bookmark).Literal))

		default:
			panic("invalid ast block\n" + b.StringWithIndent(""))
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

// <<repair-hosting, Возможных проблем>>
func (c *Converter) WriteLink(l *ast.Link, w io.Writer)  {
	caption := l.Text
	if caption == "" {
		//shouldn't happen
		c.log.Error(context.Background(), "empty link caption", slog.F("link", l))
		caption = "(ссылка)"
	}
	w.Write([]byte(fmt.Sprintf("[%s](%s)", fixText(caption), l.Url)))
}

func (c *Converter) WriteSyntaxBlock(sb *ast.SyntaxBlock) {
	var str string
	if sb.Literal[len(sb.Literal)-1:] == "\n" {
		//trim last newline
		str = sb.Literal[:len(sb.Literal)-1]
	}
	if len(str)==0 {
		c.log.Error(context.Background(), "fuck!!", slog.F("sb", sb))
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