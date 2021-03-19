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

type IdMap map[string]string

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
			header.Text = "<skip>"
		}
		if header.Level == fs.level && header != fs.firstHeader {
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
		fs.log.Info(context.Background(), "no file name for h", slog.F("h", h.Text))
		return fmt.Sprintf("%s_%v.md", fs.slug, fs.fileIndex)
	}
	return name
}

func (fs *FileSplitter) nextFile() error {
	//close previous file
	fs.Close()
	var err error
	fs.fileName = fs.fileNames[fs.fileIndex]
	fullName := filepath.Join(fs.path, fs.fileName)
	fs.file, err = os.Create(fullName)
	if err != nil {
		return err
	}

	fs.log.Info(context.Background(), "output file created", slog.F("file", fullName))
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

//func (fs *FileSplitter)

func (fs *FileSplitter) findFirstHeader() *ast.Header {
	var hdr *ast.Header
	fs.doc.Walk(func(b ast.Block, doc *ast.Document) bool {
		h, ok := b.(*ast.Header)
		if ok && h.Level == fs.level && !h.Float {
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
	var ref, uri, file string

	uri = url.Url
	idx := strings.Index(url.Url, "#")
	if idx != -1 {
		uri = url.Url[:idx]
		ref = url.Url[idx + 1:]
	}
	switch {
	case idx == -1 && strings.HasSuffix(url.Url, ".adoc"):
		//no #, link to the document ("file.adoc")
		uri = url.Url
		ref = ""
	case idx == -1 && url.Internal:
		//no #, internal link ("apps-publish")
		uri = fs.doc.Name
		ref = url.Url
	case uri == "" && url.Internal:
		//internal link with # ("#apps-publish")
		uri = fs.doc.Name
	case url.Internal:
		// probably relative file name "../docs/admin.adoc"
		// replace backslashes to slashes for compatibility
		// path package works with slash-separated paths
		_, uri = path.Split(strings.ReplaceAll(uri, `\`, `/`))
	}

	rule := fs.findRewriteRule(uri)
	if rule != "" {
		fs.log.Debug(ctx, "found rewrite rule", slog.F(file, rule))
		uri = rule
	}

	file = fs.findIdMap(uri, ref)

	if file == "" && rule == "" {
		//no rewrite rule && no file mapping
		fs.log.Error(ctx, "cannot rewrite url: idmap is not found", slog.F("url", url), slog.F("doc", root.Name))
		return
	}
	old := url.Url
	if file != "" {
		url.Url = fmt.Sprintf("%v#%v", path.Join(fs.getDocPath(uri), file), ref)
	} else {
		url.Url = fmt.Sprintf("%v#%v", uri, ref)
	}

	fs.log.Debug(ctx, "successfully rewrote url", slog.F("new", url.Url), slog.F("old", old))
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

func (fs *FileSplitter) writeIdMap(name string, idMap map[string]string) error {
	if name == "" {
		return errors.New("writeIdMap: name is empty")
	}
	f, err := os.Create(name + ".idmap")
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
	// by this time fs.fileName SHOULD contain the first part file name
	fs.appendIdMap(fs.doc.Name, fs.doc.Name, fs.fileName)
	if fs.firstHeader == nil {
		//no headers in the document
		fs.log.Warn(ctx, "no first header, skip filling idmap")
		fs.fileIndex = 0
		fs.fileName = ""
		return
	}
	docPath := path.Join(fs.conf.CrossLinks[fs.doc.Name], fs.fileName)
	str := fmt.Sprintf("    - %s: %s\n", fs.firstHeader.Text, docPath)

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
				fs.fileName = fs.getNextFileName(hd)
				fs.fileNames = append(fs.fileNames, fs.fileName)
				docPath = path.Join(fs.conf.CrossLinks[fs.doc.Name], fs.fileName)
				str += fmt.Sprintf("    - %s: %s\n", hd.Text, docPath)
			}
			if hd.Id != "" {
				fs.appendIdMap(fs.doc.Name, hd.Id, fs.fileName)
			}
		case *ast.Bookmark:
			//fs.log.Debug(ctx, "walking by bookmark", slog.F("header", b.(*ast.Bookmark)))
			fs.appendIdMap(fs.doc.Name, b.(*ast.Bookmark).Literal, fs.fileName)
		}
		return true
	}, nil)

	fs.fileIndex = 0
	fs.fileName = ""
	if printYaml {
		fmt.Print(str)
	}

}

func (fs *FileSplitter) appendIdMap(doc string, id string, file string) {
	if id == "" {
		//ignore empty IDs
		return
	}
	if fs.idMaps[fs.doc.Name] == nil {
		fs.idMaps[fs.doc.Name] = make(IdMap)
	}
	fs.idMaps[fs.doc.Name][id] = file
}

func (fs *FileSplitter) findIdMap(doc string, id string) string {
	ctx := context.Background()
	if fs.idMaps[doc] == nil {
		//try to read *.idmap file
		data, err := ioutil.ReadFile(doc + ".idmap")
		if err != nil {
			fs.log.Error(ctx, "cannot load idmap file", slog.F("err", err))
			//no file, create emtpy map
			fs.idMaps[doc] = make(IdMap)
			return ""
		}
		var idm IdMap
		err = yaml.Unmarshal(data, &idm)
		if err != nil {
			fs.log.Error(ctx, "cannot load idmap file", slog.F("err", err))
			//no file, create emtpy map
			fs.idMaps[doc] = make(IdMap)
			return ""
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
	fs.fillIdMap(fillMapOnly)
	if fillMapOnly {
		for k, m := range fs.idMaps {
			err := fs.writeIdMap(k, m)
			if err != nil {
				return err
			}
		}
	}
	fs.fixUrls()
	return nil
}
