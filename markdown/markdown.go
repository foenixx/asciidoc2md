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
	strictIndent bool // use 4 spaces as indentation for every nesting level
	imageFolder string
	curIndent   string //current indentation level: 2 spaces, 4 spaces, ...
	log         slog.Logger
	writerFunc  GetWriterFunc
	writer      io.Writer
	skipCurChapter bool
	//writerFile  string
	idMap	map[string]string//header id to file mapping
}

func New(imFolder string, idMap map[string]string, logger slog.Logger, writerFunc GetWriterFunc) *Converter {
	return &Converter{imageFolder: imFolder,
		idMap: idMap,
		log: logger,
		writerFunc: writerFunc,
		strictIndent: true}
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
	//var exp strings.Builder
	var m = "* "
	if l.Numbered {
		m = "1. "
	}
	indent := c.curIndent

	for _, i := range l.Items {
		if c.strictIndent {
			c.curIndent = indent + "    " //4 spaces
		} else {
			c.curIndent = indent + strings.Repeat(" ", len(m))
		}
		c.WriteString("\n" + indent + m)
		c.WriteContainerBlock(i, false)
		//c.log.Debug(context.Background(), str)
		//c.writer.Write([]byte(str))
	}
	c.curIndent = indent
}

// ConvertComplexTable converts complex table into a list.
// For example, if input table has 3 columns, then exp list would be:
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
	//var exp strings.Builder
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
	w.Write([]byte(fmt.Sprintf("_%v_\n", strings.TrimSpace(fixText(h.Title)))))
}

func (c *Converter) WriteHeader(h *ast.Header, w io.Writer) {
	/*
	if h.Id != "" {
		w.Write([]byte(fmt.Sprintf(`<a id="%v"></a>` + "\n", h.Id)))
	}
	 */
	if h.Float {
		//render float headers as italic text
		w.Write([]byte("_" + fixText(h.Text) + "_\n"))
		return
	}
	anchor := "\n"
	if h.Id != "" {
		anchor = fmt.Sprintf(" { #%s }\n", h.Id)
	}
	w.Write([]byte(strings.Repeat("#", h.Level) + " " + fixText(h.Text) + anchor))


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
var boldWrappedRE = regexp.MustCompile(`^\*(.*)\*$`)
// "`+++strange formatting+++`"
var passThruMarkRE = regexp.MustCompile(`\+{3}`)
var passThruRE = regexp.MustCompile(`\+{3}.+?\+{3}`)
var checkedRE = regexp.MustCompile(`^\[\*\]`)
// match only single stars "*" pairs at word boundary and ignore single "*" without pair and double stars "**"
//var boldRE = regexp.MustCompile(`([^\*]|^)\B\*\b|\b\*\B([^\*]|$)`)
var boldRE = regexp.MustCompile(`(\s|[[:punct:]]|^)\*([^\s\*])(.+?)([^\s\*])\*(\s|[[:punct:]]|$)`)
var sharpSpaceRE = regexp.MustCompile(`#(\s)`) // "# text" patterns. Need to escape sharp symbol.
var hardBreakRE = regexp.MustCompile(`\s\+\s*$`)
var smallTextRE = regexp.MustCompile(`\[small]#(.*?)#`)
var sharpTextRE = regexp.MustCompile(`(#(?:[^\s[:punct:]]|_)+)`) // "#name_id some text"-like patterns outside of backticked spans.
var mdEscAllRE = regexp.MustCompile(`([\\\x60*_{}[\]\(\)#\+-\.!\|])`) // `<>` signs are excluded since there is specific rule for them
// Only non-escaped and also we have to check that there is no backtick prepending.
// This is since we convert "#text" to "`#text`" for better readability, otherwise it gets corrected and
//   becomes "`\#text`"
var mdEscTextRE = regexp.MustCompile(`([^\\\x60])([#\|])`)

func fixString(s string, backticked bool) string {

	if backticked {
		// fix "`*monospace and bold*`" since it isn't allowed in markdown
		s = boldWrappedRE.ReplaceAllString(s, "$1")
		// remove "+++text+++" wrappers
		s = passThruMarkRE.ReplaceAllLiteralString(s, "")
		// insert "word joiner" unicode character (https://www.compart.com/en/unicode/U+2060)
		// in the middle of "{#" to prevent jinja2 from identifying it as incorrent comment tag
		s = strings.ReplaceAll(s, "{#", "{\u2060#")
	}
	if !backticked {
		// fix html passthru syntax "+++some * text # here+++"
		s = passThruRE.ReplaceAllStringFunc(s, func(s1 string) string {
			// remove triple pluses
			trimmed := s1[3:len(s1)-3]
			return mdEscAllRE.ReplaceAllString(trimmed, `\$1`)
		})
		// replace "#name" with "`#name`" BEFORE escaping markdown special symbols
		s = sharpTextRE.ReplaceAllString(s, "`$1`")
		// escaping all markdown special symbols
		s = mdEscTextRE.ReplaceAllString(s, `$1\$2`)
		// fix checked lists "[*]" -> "[x]"
		s = checkedRE.ReplaceAllLiteralString(s, "[x]")
		// converting "*" (asciidoc bold) to "**" (markdown bold)
		// no need to convert asciidoc italic "_" since it's still an italic in markdown
		s = boldRE.ReplaceAllString(s, "$1**$2$3$4**$5")
		//s = sharpSpaceRE.ReplaceAllString(s, `\#$1`)
		s = strings.ReplaceAll(s, "->", "→")
		s = strings.ReplaceAll(s, "<", "&lt;")
		s = strings.ReplaceAll(s, ">", "&gt;")
	}
	return s
}

func fixText(s string) string {
	// asciidoc magic "Section1.Field1\=>Section2.Field2"
	s = strings.ReplaceAll(s, `\->`, `->`)
	s = strings.ReplaceAll(s, `\=>`, `=>`)
	// replace NBSP with ordinary space
	s = strings.ReplaceAll(s, "\u00a0", " ")
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
			s2 = s[ind[0]+1:ind[1]-1] //exclude backticks
			fixed.WriteRune('`')
			fixed.WriteString(fixString(s2, true))
			fixed.WriteRune('`')
			//fmt.Printf("'%s'\n", s2)
		}
		beg = ind[1]
	}
	return hardBreakRE.ReplaceAllString(fixed.String(), `<br>`)
}

func (c *Converter) WriteParagraph(p *ast.Paragraph, noFormatFix bool, w io.Writer) {
	for _, b := range p.Blocks {
		switch b.(type) {
		case *ast.Text:
			txt := b.(*ast.Text)
			str := txt.Text
			if str == "\n" {
				// convert single newline to space
				w.Write([]byte(" "))
				continue
			}
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
	var k string
	switch {
	case ex.Kind == "CAUTION":
		//convert to "danger"
		k = "danger"
	case ex.Kind != "":
		//all others
		k = strings.ToLower(ex.Kind)
	case ex.Delim.Type == token.EX_BLOCK:
		//just an example block
		k = "example"
	default:
		k = "info"
	}
	c.writer.Write([]byte(" " + k + "\n"))
	c.WriteContainerBlock(&ex.ContainerBlock, true)
	c.curIndent = ind
}


func (c *Converter) WriteString(s string) error {
	_, err := c.writer.Write([]byte(s))
	return err
}

func (c *Converter) WriteContainerBlock(p *ast.ContainerBlock, firstLineIndent bool)  {
	//var exp strings.Builder

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
		case *ast.ListBlock:
			c.WriteContainerBlock(&b.(*ast.ListBlock).ContainerBlock,firstLineIndent)
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
			sb := b.(*ast.SyntaxBlock)
			hasAnn := c.hasAnnotations(sb)
			c.WriteSyntaxBlock(sb)
			if hasAnn {
				var cl *ast.List
				var nb ast.Block
				var ok bool
				var j int
				//let's find connected list of annotations
				for j, nb = range p.Blocks[i+1:] {
					cl, ok = nb.(*ast.List)
					if ok {
						break
					}
				}
				if ok {
					c.WriteList(cl)
					if j == 0 {
						// list of annotations without prepending text
						// writing prepending text
						c.WriteString("\n" + c.curIndent + "_Выноски:_")
						//c.WriteParagraph(ast.NewParagraphFromStr("_Выноски:_"), true, c.writer)
					}
				} else {
					c.log.Error(context.Background(), "cannot find connected list of annotations", slog.F("syntax block", sb.StringWithIndent("")))
				}
			}
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

func (c *Converter) WriteLink(l *ast.Link, w io.Writer)  {
	caption := l.Text
	if caption == "" {
		//shouldn't happen
		c.log.Debug(context.Background(), "empty link caption", slog.F("link", l))
		caption = l.Url
	}
	w.Write([]byte(fmt.Sprintf("[%s](%s)", fixText(caption), l.Url)))
}

var calloutRE = regexp.MustCompile(`<(\.|\d+)>`)

func (c *Converter)	hasAnnotations(sb *ast.SyntaxBlock) bool {
	return calloutRE.MatchString(sb.Literal)
}

func (c *Converter) fixAnnotations(sb *ast.SyntaxBlock) bool {
	var i int
	sb.Literal = calloutRE.ReplaceAllStringFunc(sb.Literal, func(s string) string {
		i++
		return fmt.Sprintf(`/* (%v) */`, i)
	})
	// return true if there were some annotations in the code
	return i > 0
}

func (c *Converter) WriteSyntaxBlock(sb *ast.SyntaxBlock) {
	var str string
	//correct annotations tags
	hasAnn := c.fixAnnotations(sb)

	if sb.Literal[len(sb.Literal)-1:] == "\n" {
		//trim last newline
		str = sb.Literal[:len(sb.Literal)-1]
	}

	if str[0] == '\n' {
		//trim first newline
		str = str[1:]
	}

	if sb.InlineHighlight {
		//there are `pass:quotes[#some_text#]` highlighting
		//str = strings.ReplaceAll(str, `pass:quotes[#`, `<span style="background-color: lightyellow">`)
		str = strings.ReplaceAll(str, `pass:quotes[#`, `<span class="tessa-code-accent">`)
		str = strings.ReplaceAll(str, `#]`, `</span>`)
		c.WriteString(fmt.Sprintf(`<pre><code lang="%s">`, sb.Lang))
		c.WriteString(str)
		c.WriteString("</code></pre>\n")
		return
	}
	str = strings.ReplaceAll(str,"\n", "\n" + c.curIndent)
	lang := sb.Lang
	if hasAnn {
		// "{ .js .annotate }"
		lang = fmt.Sprintf(`{ .%s .annotate }`, sb.Lang)
	}
	c.WriteString(fmt.Sprintf("``` %s\n%s%s\n%s```\n", lang, c.curIndent, str, c.curIndent))
}