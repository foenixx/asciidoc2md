package main

import (
	"asciidoc2md/parser"
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

	p := parser.New(splitterTestInput, "", log)
	doc, err := p.Parse()
	if !assert.NoError(t, err) {
		return
	}

	splitter := NewFileSplitter(doc, "slug", nil, "", logger)
	h := splitter.findFirstHeader()
	if !assert.NotNil(t, h) {
		return
	}
	assert.Equal(t, "Header2", h.Text)
}

func TestSplitter_NextFile(t *testing.T) {
	//ctx := context.Background()
	logger := slogtest.Make(t, nil).Leveled(slog.LevelDebug)
	headerMap := map[string]string{ "Header2": "part2.md"}

	p := parser.New(splitterTestInput, "", log)
	doc, err := p.Parse()
	if !assert.NoError(t, err) {
		return
	}

	splitter := NewFileSplitter(doc, "slug", headerMap, "", logger)
	// init splitter
	splitter.init()
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
	headerMap := map[string]string{ "Header2": "part2.md"}

	p := parser.New(splitterTestInput, "", log)
	doc, err := p.Parse()
	if !assert.NoError(t, err) {
		return
	}

	splitter := NewFileSplitter(doc, "slug", headerMap, "", logger)
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
	headerMap := map[string]string{ "Header2": "part2.md"}

	logger := slogtest.Make(t, nil).Leveled(slog.LevelDebug)
	logger.Info(ctx, "splitter test")
	p := parser.New(splitterTestInput, "", log)
	log.Info(context.Background(), "test message")
	doc, err := p.Parse()
	if !assert.NoError(t, err) {
		return
	}
	splitter := NewFileSplitter(doc, "slug", headerMap, ".",logger)
	splitter.init()
	assert.Equal(t, map[string]string{"id_header3": "part2.md","id_listitem2": "slug_2.md"}, splitter.idMap)
	assert.Equal(t, []string{"part2.md", "slug_2.md"}, splitter.fileNames)
	//logger.Info(ctx, "filling idMap", slog.F("idmap", splitter.idMap))
}

func TestSplitter_Debug(t *testing.T) {
	logger := slogtest.Make(t, nil).Leveled(slog.LevelDebug)

	inputFile := "docs/admin/AdministratorGuide.adoc"
	outputSlug := "admin"
	outputPath := "c:/personal/mkdocs/tessa_docs/docs/"
	imagePath := "/images"

	input, err := ioutil.ReadFile(inputFile)
	if !assert.NoError(t, err) {
		return
	}

	p := parser.New(string(input), filepath.Dir(inputFile), log)
	doc, err := p.Parse()
	if err != nil {
		panic(err)
	}
	splitter := NewFileSplitter(doc, outputSlug, nil, outputPath, logger)
	err = splitter.RenderMarkdown(imagePath)
	assert.NoError(t, err)

}
