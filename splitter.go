package main

import (
	"asciidoc2md/ast"
	"asciidoc2md/markdown"
	"asciidoc2md/settings"
	"asciidoc2md/utils"
	"bufio"
	"cdr.dev/slog"
	"context"
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type IdMapEntry struct {
	FileName string
	Caption string
}
type IdMap map[string]*IdMapEntry

type FileSplitter struct {
	doc         *ast.Document
	conf 		*settings.Config
	//headerMap   map[string]string //header -> file name
	idMaps      map[string]IdMap  //document name -> header or bookmark id -> output file name
	log         slog.Logger
	slug        string
	path        string //output path
	level       int //split at the specified level headers
	firstHeader *ast.Header
	fileIndex   int
	fileName    string //current fileName
	fileNames   []string //all the filenames
	file        *os.File 	//current file
	w           *bufio.Writer  	//current writer
}

const (
	SkipChapterMark = "<skip chapter>"
	SkipHeaderMark = "<skip>"
)


func NewFileSplitter(doc *ast.Document, nameSlug string, conf *settings.Config, path string, splitLvl int, log slog.Logger) *FileSplitter {
	return &FileSplitter{
		doc:    doc,
		conf: 	conf,
		idMaps:	make(map[string]IdMap),
		level:  splitLvl,
		log:    log,
		slug:   nameSlug,
		path:   path}
}

func (fs *FileSplitter) GenerateIdMap() error {
	return fs.init(true)
}

func (fs *FileSplitter) decreaseHeader(h *ast.Header) {
	if fs.level != 1 {
		h.Level--
	}
}

func (fs *FileSplitter) RenderMarkdown(imagePath string) error {
	err := fs.init(false)
	if err != nil {
		return err
	}
	err = fs.nextFile()
	if err != nil {
		return err
	}
	defer fs.Close()

	conv := markdown.New(imagePath, nil, fs.log, func(header *ast.Header) io.Writer {
		if header.Level < fs.level && fs.level != 1 {
			header.Text = SkipHeaderMark
		}
		if header.Level == fs.level && header != fs.firstHeader {
			if fs.skipChapter(header) {

			}
			err := fs.nextFile()
			if err != nil {
				fs.log.Error(context.Background(), err.Error())
				return nil
			}
			fs.decreaseHeader(header)
			return fs.w

		}
		fs.decreaseHeader(header)
		return nil
	})
	conv.RenderMarkdown(fs.doc, fs.w)
	return nil
}

func (fs *FileSplitter) getNextFileName(h *ast.Header) string {
	fs.fileIndex++
	if h == nil {
		return fmt.Sprintf("%s_%v.md", fs.slug, fs.fileIndex)
	}
	name, ok := fs.conf.Headers[fs.doc.Name][h.Text]
	if !ok {
		//fs.log.Info(context.Background(), "no file name for h", slog.F("h", h.Text))
		return fmt.Sprintf("%s_%v.md", fs.slug, fs.fileIndex)
	}
	return name
}

func (fs *FileSplitter) nextFile() error {
	//close previous file
	fs.Close()
	var err error
	fs.fileName = fs.fileNames[fs.fileIndex]
	if fs.fileName == SkipChapterMark {
		fs.w = bufio.NewWriter(ioutil.Discard)
		fs.file = nil
		fs.log.Warn(context.Background(), "discard writer created")
		return nil
	}

	fullName := filepath.Join(fs.path, fs.fileName)
	fs.file, err = os.Create(fullName)
	if err != nil {
		return err
	}

	fs.log.Debug(context.Background(), "output file created", slog.F("file", fullName))
	fs.w = bufio.NewWriter(fs.file)
	fs.fileIndex++

	return nil
}

func (fs *FileSplitter) Close() {
	//close previous files
	if fs.w != nil {
		fs.w.Flush()
	}
	if fs.file != nil {
		fs.file.Close()
	}
}

func (fs *FileSplitter) skipChapter(h *ast.Header) bool {
	m, ko := fs.conf.Headers[fs.doc.Name][h.Text]
	if ko && m == SkipChapterMark {
		fs.log.Warn(context.Background(), "skipping chapter", slog.F("chapter", h.Text))
		return true
	}
	return false
}

func (fs *FileSplitter) findFirstHeader() *ast.Header {
	var hdr *ast.Header
	fs.doc.Walk(func(b ast.Block, doc *ast.Document) bool {
		h, ok := b.(*ast.Header)
		if ok && h.Level == fs.level && !h.Float && !fs.skipChapter(h) {
			hdr = h
			return false
		}
		return true
	}, fs.doc)

	return hdr
}

func (fs *FileSplitter)	findRewriteRule(url string) string {
	for _, elem := range fs.conf.UrlRewrites {
		for k, r := range elem {
			if strings.Contains(url, k) {
				return r
			}
		}
	}
	return ""
}

func (fs *FileSplitter) urlRewrite(url *ast.Link, root *ast.Document) {
	ctx := context.Background()
	var idRef, adocRef string
	var entry *IdMapEntry

	adocRef = url.Url
	idx := strings.Index(url.Url, "#")
	if idx != -1 {
		adocRef = url.Url[:idx]
		idRef = url.Url[idx + 1:]
	}
	switch {
	case idx == -1 && strings.HasSuffix(url.Url, ".adoc"):
		//no #, link to the document ("file.adoc")
		adocRef = url.Url
		idRef = ""
	case idx == -1 && url.Internal:
		//no #, internal link ("apps-publish")
		adocRef = fs.doc.Name
		idRef = url.Url
	case adocRef == "" && url.Internal:
		//internal link with # ("#apps-publish")
		adocRef = fs.doc.Name
	case url.Internal:
		// probably relative file name "../docs/admin.adoc"
		// replace backslashes to slashes for compatibility
		// path package works with slash-separated paths
		_, adocRef = path.Split(strings.ReplaceAll(adocRef, `\`, `/`))
	}

	rule := fs.findRewriteRule(adocRef)
	if rule != "" {
		fs.log.Debug(ctx, "found rewrite rule", slog.F(adocRef, rule))
		adocRef = rule
	}

	if !url.Internal && rule == "" {
		// external link without rewrite rule
		fs.log.Debug(ctx, "external link without rewrite rule", slog.F("link", url))
		return
	}

	entry = fs.findIdMap(adocRef, idRef)
	if entry == nil {
		// let's try fallbacks if any
		fb := fs.conf.IdMapFallbacks[adocRef]
		if entry = fs.findIdMap(fb, idRef); entry != nil {
			adocRef = fb
		}
	}

	old := url.Url
	if entry != nil {
		url.Url = fmt.Sprintf("%v#%v", path.Join(fs.getDocPath(adocRef), entry.FileName), idRef)
		if url.Text == "" {
			url.Text = entry.Caption
		}
		fs.log.Debug(ctx, "successfully rewrote url", slog.F("new", url.Url), slog.F("old", old))
	} else {
		fs.log.Error(ctx, "cannot rewrite url: idmap is not found", slog.F("url", url), slog.F("doc", root.Name))
		//url.Url = fmt.Sprintf("%v#%v", adocRef, idRef)
	}
}

//should be called AFTER fillIdMap
func (fs *FileSplitter) fixUrls() {
	fs.doc.Walk(
		func(b ast.Block, root *ast.Document) bool {
			ctx := context.Background()
			if b == nil || utils.IsNil(b) {
				fs.log.Error(ctx, "walker: nil block")
			}
			switch b.(type) {
			case *ast.Link:
				link := b.(*ast.Link)
				fs.urlRewrite(link, root)
			}
			return true
		},
		nil)

}

func (fs *FileSplitter) writeIdMap(name string, idMap IdMap) error {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer f.Close()
	out, err := yaml.Marshal(idMap)
	_, err = f.Write(out)
	return err
}

func (fs *FileSplitter) getDocPath(doc string) string {
	// we need to address outer document docs/doc2/file2.md from docs/doc1/file1.md
	if doc == fs.doc.Name {
		return ""
	}
	// first get to the root folder "docs" by writing <stepsUp> double dots: ../
	stepsUp := utils.CountDirs(fs.conf.CrossLinks[fs.doc.Name])
	base := strings.Repeat("../", stepsUp)
	// now append relative path from the "docs" to "file2.md" to the path: ../doc2/file2.md
	return path.Join(base, fs.conf.CrossLinks[doc])
}

func (fs *FileSplitter)	fillIdMap(printYaml bool) {
	ctx := context.Background()
	// link can refer to the document in whole, without any id after '#'
	// by this time fs.fileName SHOULD contain the first part's file name
	fs.appendIdMap(fs.doc.Name, fs.doc.Name, fs.fileName, "")
	if fs.firstHeader == nil {
		//no headers in the document
		fs.log.Warn(ctx, "no first header, skip filling idmap")
		fs.fileIndex = 0
		fs.fileName = ""
		return
	}
	docPath := path.Join(fs.conf.CrossLinks[fs.doc.Name], fs.fileName)
	nav := []string{fmt.Sprintf("- %s: %s", fs.firstHeader.Text, docPath)}
	skipCurChapter := false
	fs.doc.Walk(func(b ast.Block, root *ast.Document) bool {

		//fs.log.Debug(ctx, "walker block", slog.F("block", b))
		if b == nil || utils.IsNil(b) {
			fs.log.Error(ctx, "walker: nil block")
		}
		switch b.(type) {
		case *ast.Header:
			hd := b.(*ast.Header)
			//fs.log.Debug(ctx, "walking by header", slog.F("header", hd))
			if hd.Level == fs.level && hd != fs.firstHeader && !hd.Float {
				if fs.skipChapter(hd) {
					skipCurChapter = true
					fs.fileName = SkipChapterMark

				} else {
					skipCurChapter = false
					fs.fileName = fs.getNextFileName(hd)
					docPath = path.Join(fs.conf.CrossLinks[fs.doc.Name], fs.fileName)
					nav = append(nav, fmt.Sprintf("- %s: %s", hd.Text, docPath))
				}
				fs.fileNames = append(fs.fileNames, fs.fileName)
			}
			if hd.Id != "" && !skipCurChapter {
				fs.appendIdMap(fs.doc.Name, hd.Id, fs.fileName, hd.Text)
			}
		case *ast.Bookmark:
			//fs.log.Debug(ctx, "walking by bookmark", slog.F("header", b.(*ast.Bookmark)))
			if  !skipCurChapter {
				fs.appendIdMap(fs.doc.Name, b.(*ast.Bookmark).Literal, fs.fileName, "")
			}
		}
		return true
	}, nil)

	fs.fileIndex = 0
	fs.fileName = ""
	if printYaml {
		fmt.Print(strings.Join(nav, "\n") + "\n")
	}
	if fs.conf.NavFile != "" {
		err := fs.writeNavToFile(fs.conf.NavFile, fs.doc.Name, nav)
		if err != nil {
			panic(err)
		}
	}
}

func (fs *FileSplitter) writeNavToFile(navFile string, docFile string, nav []string) error {
	data, err := ioutil.ReadFile(navFile)
	if err != nil {
		return err
	}
	out, err := writeNav(string(data), docFile, nav)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(navFile, []byte(out), 666)
	return err
}

func (fs *FileSplitter) appendIdMap(doc string, id string, file string, caption string) {
	if id == "" {
		//ignore empty IDs
		return
	}
	if fs.idMaps[fs.doc.Name] == nil {
		fs.idMaps[fs.doc.Name] = make(IdMap)
	}
	fs.idMaps[fs.doc.Name][id] = &IdMapEntry{file, caption}
}

func (fs *FileSplitter) findIdMap(doc string, id string) *IdMapEntry {
	ctx := context.Background()
	if fs.idMaps[doc] == nil {
		//try to read *.idmap file
		data, err := ioutil.ReadFile(filepath.Join(fs.conf.ArtifactsDir, doc) + ".idmap")
		if err != nil {
			fs.log.Error(ctx, "cannot load idmap file", slog.F("err", err))
			//no file, create emtpy map
			fs.idMaps[doc] = make(IdMap)
			return nil
		}
		var idm IdMap
		err = yaml.Unmarshal(data, &idm)
		if err != nil {
			fs.log.Error(ctx, "cannot load idmap file", slog.F("err", err))
			//no file, create emtpy map
			fs.idMaps[doc] = make(IdMap)
			return nil
		}

		fs.idMaps[doc] = idm
	}
	if id == "" {
		//link to the document without id
		return fs.idMaps[doc][doc]
	}

	return fs.idMaps[doc][id]
}

func (fs *FileSplitter) init(fillMapOnly bool) error {
	fs.firstHeader = fs.findFirstHeader()
	fs.fileName = fs.getNextFileName(fs.firstHeader)
	fs.fileNames = append(fs.fileNames, fs.fileName)
	fs.fillIdMap(false)
	if fillMapOnly {
		if len(fs.idMaps) != 1 {
			return errors.New("several id maps found")
		}
		for _, m := range fs.idMaps {
			err := fs.writeIdMap(filepath.Join(fs.conf.ArtifactsDir, fs.doc.Name + ".idmap"), m)
			if err != nil {
				return err
			}
		}
	}
	if !fillMapOnly {
		fs.fixUrls()
	}
	return nil
}
