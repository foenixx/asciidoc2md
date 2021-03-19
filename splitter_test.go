package main

import (
	"asciidoc2md/parser"
	"asciidoc2md/settings"
	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/slogtest"
	"context"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

const splitterTestInput = `
= Header1

== Header2

* this is a <<id_listitem2, link to second file>>
* this is not

[[id_header3]]
=== header3

some text below header3

== Header4

* list item1 and <<id_header3, link to header3>>
* [[id_listitem2]] list item2
`
func TestSplitter_FindFirstHeader(t *testing.T) {
	//ctx := context.Background()
	logger := slogtest.Make(t, nil).Leveled(slog.LevelDebug)

	p := parser.New(splitterTestInput, nil, log)
	doc, err := p.Parse("gotest.adoc")
	if !assert.NoError(t, err) {
		return
	}

	splitter := NewFileSplitter(doc, "slug", nil, "", 2, logger)
	h := splitter.findFirstHeader()
	if !assert.NotNil(t, h) {
		return
	}
	assert.Equal(t, "Header2", h.Text)
}

func TestSplitter_NextFile(t *testing.T) {
	//ctx := context.Background()
	logger := slogtest.Make(t, nil).Leveled(slog.LevelDebug)
	//headerMap := map[string]string{ "Header2": "part2.md"}

	p := parser.New(splitterTestInput, nil, log)
	doc, err := p.Parse("gotest.adoc")
	if !assert.NoError(t, err) {
		return
	}

	splitter := NewFileSplitter(doc, "slug", &settings.Config{}, "", 2, logger)
	// init splitter
	splitter.init(false)
	for i := range []int{0,1} {
		err = splitter.nextFile()
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, splitter.fileNames[i], splitter.fileName)
		assert.NotNil(t, splitter.file)
		assert.NotNil(t, splitter.w)
		assert.FileExists(t, splitter.fileName)
	}
	splitter.Close()
	for i := range []int{0,1} {
		err = os.Remove(splitter.fileNames[i])
		assert.NoError(t, err)
	}
}


func TestSplitter_WriteMarkdown(t *testing.T) {
	//ctx := context.Background()
	logger := slogtest.Make(t, nil).Leveled(slog.LevelDebug)
	//headerMap := map[string]string{ "Header2": "part2.md"}

	p := parser.New(splitterTestInput, nil, log)
	doc, err := p.Parse("gotest.adoc")
	if !assert.NoError(t, err) {
		return
	}

	splitter := NewFileSplitter(doc, "slug", &settings.Config{}, "", 2, logger)
	// init splitter
	err = splitter.RenderMarkdown("")
	assert.NoError(t, err)
	splitter.Close()
	for i := range splitter.fileNames {
		os.Remove(splitter.fileNames[i])
		//assert.NoError(t, err)
	}
}

func TestSplitter(t *testing.T) {
	ctx := context.Background()
	//headerMap := map[string]string{ "Header2": "part2.md"}

	logger := slogtest.Make(t, nil).Leveled(slog.LevelDebug)
	logger.Info(ctx, "splitter test")
	p := parser.New(splitterTestInput, nil, log)
	log.Info(context.Background(), "test message")
	doc, err := p.Parse("gotest.adoc")
	if !assert.NoError(t, err) {
		return
	}
	splitter := NewFileSplitter(doc, "slug", &settings.Config{}, ".",2, logger)
	splitter.init(true)
	assert.Equal(t, map[string]string{"id_header3": "part2.md","id_listitem2": "slug_2.md"}, splitter.idMaps)
	assert.Equal(t, []string{"part2.md", "slug_2.md"}, splitter.fileNames)
	//logger.Info(ctx, "filling idMaps", slog.F("idmap", splitter.idMaps))
}

func TestSplitter_Debug1(t *testing.T) {
	logger := slogtest.Make(t, nil).Leveled(slog.LevelInfo)

	inputFile := "docs/linux_inst/LinuxInstallationGuide.adoc"
	includePath := filepath.Dir(inputFile)
	outputSlug := "linux_inst"
	outputPath := "" //"c:/personal/mkdocs/tessa_docs/docs/dev"
	imagePath := "/images"

/*	inputFile := "docs/beginners/BeginnersGuide.adoc"
	//inputFile := "test.adoc"
	includePath := filepath.Dir(inputFile)
	outputSlug := "tt"
	outputPath := ""
	//imagePath := "/images"
*/	config := loadConfig("settings.yml")

	input, err := ioutil.ReadFile(inputFile)
	if !assert.NoError(t, err) {
		return
	}

	p := parser.New(string(input), func(name string) ([]byte, error) {
		return ioutil.ReadFile(filepath.Join(includePath, name))
	}, log)
	doc, err := p.Parse(inputFile)
	ioutil.WriteFile("test.out", []byte(doc.StringWithIndent("")), os.ModePerm)
	if err != nil {
		panic(err)
	}
	splitter := NewFileSplitter(doc, outputSlug, config, outputPath, 2, logger)
	//err = splitter.RenderMarkdown(imagePath)
	//err = splitter.GenerateIdMap()
	err = splitter.RenderMarkdown(imagePath)
	assert.NoError(t, err)

}

func TestSplitter_Debug2(t *testing.T) {
	logger := slogtest.Make(t, nil).Leveled(slog.LevelDebug)
	abs, _ := filepath.Abs(".")
	logger.Info(context.Background(), abs)

	input :=`
IMPORTANT: Не рекомендуется настраивать замещение непосредственно из карточки роли. В системе есть существенно более удобный механизм - карточка <<..\UserGuide\UserGuide.adoc#my-deputies,Мои замещения>>. Администраторы могут настраивать замещения пользователей непосредственно из карточки пользователя.
`
	inc :=``

	p := parser.New(input, func(name string) ([]byte, error) {
		return []byte(inc), nil
	}, log)
	doc, err := p.Parse("gotest.adoc")
	if err != nil {
		panic(err)
	}
	conf := loadConfig("settings.yml")
	splitter := NewFileSplitter(doc, "outputSlug", conf, ".", 2, logger)
	//splitter.init()
	err = splitter.RenderMarkdown("")
	assert.NoError(t, err)
	for i := range splitter.fileNames {
		os.Remove(splitter.fileNames[i])
		//assert.NoError(t, err)
	}

}
