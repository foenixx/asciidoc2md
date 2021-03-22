package main

import (
	"asciidoc2md/parser"
	"asciidoc2md/settings"
	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/sloghuman"
	"context"
	"github.com/alecthomas/kong"
	"github.com/fatih/color"
	"io/ioutil"
	stdLog "log"
	"os"
	"path/filepath"
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



var cli struct {
	Debug bool `help:"Debug mode."`
	Config     string `help:"Configuration file." short:"c" type:"existingfile"`
	Slug string `optional help:"A template for split file name. Output files would have names like <slug>_[1...N].md." default:"part"`
	SplitLevel int `optional help:"A level of the headers to split a file at." default:2`
	Dump string `help:"Write parsed document to file."`
	GenMap struct {
		Input string `arg help:"*.adoc file to process." type:"existingfile" name:"file.adoc"`
		WriteNav string `optional help:"Path to mkdocs.yml file to write navigation index." type:"existingfile"`
	} `cmd:"" help:"Generate <file.adoc.idmap> file."`
	Convert struct {
		Input string `arg help:"*.adoc file to process." type:"existingfile" name:"file.adoc"`
		Out string `help:"Output directory." short:"o" type:"existingdir"`
		ImagePath string `help:"A relative path to the images folder." short:"im" default:"images/" `
	} `cmd:"" help:"Convert <file.adoc> into markdown."`
}


//asciidoc2md input_file output_path output_file_slug image_path
// go run asciidoc2md data/adm.adoc /mnt/c/personal/mkdocs/my-project/docs/ adm/adm images/
func main() {
	ctx := kong.Parse(&cli,
		kong.Name("asciidoc2md"),
		kong.Description("Asciidoc to markdown file converter."),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}))
	if cli.Debug {
		initLog(true)
	} else {
		initLog(false)
	}
	switch ctx.Command() {
	case "gen-map <file.adoc>":
		genIdMap()

	case "convert <file.adoc>":
		convert()
	}

}

func loadConfig(configFile string) *settings.Config {
	var config *settings.Config
	if configFile != "" {

		str, err := ioutil.ReadFile(configFile)
		if err != nil {
			panic(err)
		}
		config, err = settings.Parse(str)
		if err != nil {
			panic(err)
		}
	} else {
		config = &settings.Config{}
	}
	return config
}

func genIdMap() {
	log.Debug(context.Background(), "genIdMap")
	splitter := initSplitter(cli.GenMap.Input,
		cli.Config,
		"",
		"",
		cli.Slug,
		cli.SplitLevel,
		cli.Dump,
		cli.GenMap.WriteNav)

	err := splitter.GenerateIdMap()
	if err != nil {
		panic(err)
	}
}

func initSplitter(inputFile string, configFile string, imagePath string, outPath string, slug string, splitLvl int, dumpFile string, navFile string) *FileSplitter {
	ctx := context.Background()
	log.Debug(ctx, "convert")
	log.Info(ctx, "input file", slog.F("name", inputFile))
	log.Info(ctx, "image path", slog.F("path", imagePath))
	log.Info(ctx, "settings", slog.F("settings", configFile))
	conf := loadConfig(configFile)
	conf.NavFile = navFile
	log.Debug(ctx, "config file loaded")
	input, err := ioutil.ReadFile(inputFile)
	if err != nil {
		panic(err)
	}

	dir, name := filepath.Split(inputFile)
	p := parser.New(string(input), func(name string) ([]byte, error) {
		return ioutil.ReadFile(filepath.Join(dir, name))
	}, log)
	doc, err := p.Parse(name)
	if err != nil {
		panic(err)
	}
	if dumpFile != "" {
		err = ioutil.WriteFile(dumpFile, []byte(doc.String()), os.ModePerm)
		if err != nil {
			panic(err)
		}
	}

	return NewFileSplitter(doc, slug, conf, outPath, splitLvl, log)
}

func convert() {
	splitter := initSplitter(cli.Convert.Input, cli.Config, cli.Convert.ImagePath, cli.Convert.Out, cli.Slug, cli.SplitLevel, cli.Dump, "")
	err := splitter.RenderMarkdown(cli.Convert.ImagePath)
	if err != nil {
		panic(err)
	}
}
