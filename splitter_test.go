package main

import (
	"asciidoc2md/parser"
	"asciidoc2md/settings"
	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/slogtest"
	"context"
	"github.com/stretchr/testify/assert"
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

	splitter := NewFileSplitter(doc, "slug", testConf(), "", 2, logger)
	h := splitter.findFirstHeader()
	if !assert.NotNil(t, h) {
		return
	}
	assert.Equal(t, "Header2", h.Text)
}

func testConf() *settings.Config {
	return &settings.Config{Headers: map[string]settings.Headers2FileMap{}}
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
	conf := testConf()
	conf.Headers["gotest.adoc"] = settings.Headers2FileMap{}
	conf.Headers["gotest.adoc"]["Header2"] = "part2.md"

	logger := slogtest.Make(t, nil).Leveled(slog.LevelDebug)
	logger.Info(ctx, "splitter test")
	p := parser.New(splitterTestInput, nil, log)
	log.Info(context.Background(), "test message")
	doc, err := p.Parse("gotest.adoc")
	if !assert.NoError(t, err) {
		return
	}
	splitter := NewFileSplitter(doc, "slug", conf, ".",2, logger)
	splitter.init(true)
	//4 headers + 2 anchors + "gotest.adoc" record
	assert.Len(t, splitter.idMaps["gotest.adoc"], 7)
	assert.Equal(t, []string{"part2.md", "slug_2.md"}, splitter.fileNames)
	//logger.Info(ctx, "filling idMaps", slog.F("idmap", splitter.idMaps))
}

func TestSplitter_Debug1(t *testing.T) {
	logger := slogtest.Make(t, nil).Leveled(slog.LevelInfo)

	inputFile := "docs/installation/InstallationGuide.adoc"
	//inputFile := "C:\\SynProjects\\Syntellect\\Tessa\\Docs\\UserGuide\\UserGuide.adoc"
	//includePath := filepath.Dir(inputFile)
	outputSlug := "install"
	//outputPath := "C:\\SynProjects\\Syntellect\\tessa_docs\\docs\\user"
	outputPath := "mkdocs_test/docs"
	dumpFile := ""
	imagePath := "/images"

	config := initConfigCLI("settings.yml", nil)
	splitter := initSplitter(inputFile, "", outputPath, outputSlug, 2, dumpFile, config, logger)

	err := splitter.RenderMarkdown(imagePath)
	assert.NoError(t, err)
}

func testSplitterDbg2(t *testing.T) {
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
	conf := initConfigCLI("settings.yml", nil)
	splitter := NewFileSplitter(doc, "outputSlug", conf, ".", 2, logger)
	//splitter.init()
	err = splitter.RenderMarkdown("")
	assert.NoError(t, err)
	for i := range splitter.fileNames {
		os.Remove(splitter.fileNames[i])
		//assert.NoError(t, err)
	}

}
