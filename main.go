package main

import (
	"asciidoc2md/ast"
	"asciidoc2md/markdown"
	"asciidoc2md/parser"
	"bufio"
	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/sloghuman"
	"context"
	"fmt"
	"github.com/fatih/color"
	"io"
	"io/ioutil"
	stdLog "log"
	"os"
	"strings"
)

var log slog.Logger //global logger

func initLog(verbose bool) {
	os.Setenv("FORCE_COLOR", "TRUE")
	if verbose {
		log = sloghuman.Make(color.Output).Leveled(slog.LevelDebug)
		return
	}
	log = sloghuman.Make(color.Output)
	stdLog.SetOutput(slog.Stdlib(context.Background(), log).Writer())
}

//asciidoc2md input_file output_path output_file_slug image_path
// go run asciidoc2md data/adm.adoc /mnt/c/personal/mkdocs/my-project/docs/ adm/adm images/
func main() {
	ctx := context.Background()
	inputFile := os.Args[1]
	outputPath := os.Args[2]
	outputSlug := os.Args[3]
	imagePath := os.Args[4]

	initLog(false)
	log.Info(ctx, "image path", slog.F("path", imagePath))
	log.Debug(ctx, "started")
	input, err := ioutil.ReadFile(inputFile)
	if err != nil {
		panic(err)
	}

	p := parser.New(string(input), log)
	doc, err := p.Parse()
	if err != nil {
		panic(err)
	}
	//os.Stdout.WriteString(doc.String(""))
	//log.Info(ctx, "before md")
	i := 0

		fileName := fmt.Sprintf("%s_1.md", outputSlug)
		//log.Info(ctx, "writing fileName", slog.F("name", fileName))
		fo, err := os.Create(outputPath + fileName)
		if err != nil {
			panic(err)
		}
		w := bufio.NewWriter(fo)
		conv := markdown.New(imagePath, log, func(header *ast.Header) io.Writer {
			header.Level--
			if header.Level == 0 { //former 1 level
				header.Text = "<skip>"
			}
			if header.Level == 1 { //former 2 level
				i++
				fileName = fmt.Sprintf("%s_%v.md", outputSlug, i)
				//log.Info(ctx, "header", slog.F("text", header.Text))
				os.Stdout.WriteString("    - " + strings.TrimSpace(header.Text) + ": " + fileName + "\n")
				if i > 1 {
					//skip first header
					w.Flush()
					fo.Close()
					//log.Info(ctx, "writing fileName", slog.F("name", fileName))
					fo, err = os.Create(outputPath + fileName)
					if err != nil {
						panic(err)
					}
					//defer fo.Close()
					w = bufio.NewWriter(fo)
					return w
				}
			}
			return nil
		})
		conv.RenderMarkdown(doc, w)
		log.Info(ctx, "after md")
		err = w.Flush()
		fo.Close()
		if err != nil {
			panic(err)
		}


}
