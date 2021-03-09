package main

import (
	"asciidoc2md/parser"
	"asciidoc2md/settings"
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
	//os.Setenv("FORCE_COLOR", "TRUE")
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
	outputSlug := os.Args[2]
	outputPath := os.Args[3]
	imagePath := os.Args[4]
	var config *settings.Config
	if len(os.Args) >= 5 {
		settingsFile := os.Args[5]
		str, err := ioutil.ReadFile(settingsFile)
		if err != nil {
			panic(err)
		}
		config, err = settings.Parse(str)
		if err != nil {
			panic(err)
		}
	}

	initLog(false)
	log.Info(ctx, "image path", slog.F("path", imagePath))
	log.Info(ctx, "settings", slog.F("settings", config))
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
	splitter := NewFileSplitter(doc, outputSlug, config.Headers, outputPath, log)
	err = splitter.RenderMarkdown(imagePath)
	if err != nil {
		panic(err)
	}

}
