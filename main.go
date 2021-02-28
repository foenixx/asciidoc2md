package main

import (
	"asciidoc2md/markdown"
	"asciidoc2md/parser"
	"bufio"
	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/sloghuman"
	"context"
	"github.com/fatih/color"
	"io/ioutil"
	stdLog "log"
	"os"
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


func main() {
	initLog(true)
	log.Debug(context.Background(), "started")
	input, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	p := parser.New(string(input), log)
	doc, err := p.Parse()
	if err != nil {
		panic(err)
	}
	os.Stdout.WriteString(doc.String(""))

	fo, err := os.Create(os.Args[2])
	if err != nil {
		panic(err)
	}
	defer fo.Close()
	w := bufio.NewWriter(fo)
	conv := markdown.New("data/images/", log)
	conv.RenderMarkdown(doc, w)
	err = w.Flush()
	if err != nil {
		panic(err)
	}
}
