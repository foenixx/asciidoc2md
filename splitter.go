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

func NewFileSplitter(doc *ast.Document, nameSlug string, conf *settings.Config, path string, log slog.Logger) *FileSplitter {
	return &FileSplitter{
		doc:    doc,
		conf: 	conf,
		idMaps:	make(map[string]IdMap),
		level:  2,
		log:    log,
		slug:   nameSlug,
		path:   path}
}

func (fs *FileSplitter) RenderMarkdown(imagePath string) error {
	err := fs.init()
	if err != nil {
		return err
	}
	err = fs.nextFile()
	if err != nil {
		return err
	}
	defer fs.Close()

	conv := markdown.New(imagePath, nil, fs.log, func(header *ast.Header) io.Writer {
		if header.Level < fs.level {
			header.Text = "<skip>"
		}
		if header.Level == fs.level && header != fs.firstHeader {
			err := fs.nextFile()
			if err != nil {
				fs.log.Error(context.Background(), err.Error())
				return nil
			}
			header.Level--
			return fs.w
		}
		header.Level--
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

func (fs *FileSplitter) findFirstHeader() *ast.Header {
	var hdr *ast.Header
	fs.doc.Walk(func(b ast.Block, doc *ast.Document) bool {
		h, ok := b.(*ast.Header)
		if ok && h.Level == fs.level {
			hdr = h
			return false
		}
		return true
	}, fs.doc)

	return hdr
}

func (fs *FileSplitter) urlRewrite(url *ast.Link, root *ast.Document) {
	ctx := context.Background()
	var id, file, doc string
	if url.Internal {
		//link to the same file: "some_file.adoc#id"
		parts := strings.Split(url.Url, "#")
		switch len(parts) {
		case 1:
			//link to the current file: "<<id>>"
			id = url.Url
			doc = root.Name
		case 2:
			id = parts[1]
			_, doc = filepath.Split(parts[0])
		default:
			fs.log.Error(ctx, "cannot rewrite url", slog.F("url", url))
			return
		}
		file = fs.findIdMap(doc, id)
		if file == "" {
			fs.log.Error(ctx, "cannot rewrite url: idmap is not found", slog.F("url", url), slog.F("doc", root.Name))
			return
		}
		url.Url = fmt.Sprintf("%v#%v", file, id)
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
				if strings.Contains(link.Url, "adoc") {
					fs.log.Error(ctx, "incorrect!!")
				}
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


func (fs *FileSplitter)	fillIdMap(printYaml bool) {
	str := fmt.Sprintf("    - %s: %s\n", fs.firstHeader.Text, fs.fileName)

	fs.doc.Walk(func(b ast.Block, root *ast.Document) bool {
		ctx := context.Background()
		fs.log.Debug(ctx, "walker block", slog.F("block", b))
		if b == nil || utils.IsNil(b) {
			fs.log.Error(ctx, "walker: nil block")
		}
		switch b.(type) {
		case *ast.Header:
			hd := b.(*ast.Header)
			//fs.log.Debug(ctx, "walking by header", slog.F("header", hd))
			if hd.Id != "" {
				fs.appendIdMap(root.Name, hd.Id, fs.fileName)
			}
			if hd.Level == fs.level && hd != fs.firstHeader {
				fs.fileName = fs.getNextFileName(hd)
				fs.fileNames = append(fs.fileNames, fs.fileName)
				str += fmt.Sprintf("    - %s: %s\n", hd.Text, fs.fileName)
			}
		case *ast.Bookmark:
			//fs.log.Debug(ctx, "walking by bookmark", slog.F("header", b.(*ast.Bookmark)))
			fs.appendIdMap(root.Name, b.(*ast.Bookmark).Literal, fs.fileName)
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
	if fs.idMaps[doc] == nil {
		fs.idMaps[doc] = make(IdMap)
	}
	fs.idMaps[doc][id] = file
	if doc != fs.doc.Name {
		//included document, need to duplicate record in the main document idmap table
		fs.appendIdMap(fs.doc.Name, id, file)
	}
}

func (fs *FileSplitter) findIdMap(doc string, id string) string {
	ctx := context.Background()
	if fs.idMaps[doc] == nil {
		//try to read *.idmap file
		data, err := ioutil.ReadFile(doc)
		if err != nil {
			fs.log.Error(ctx, "cannot load idmap file", slog.F("err", err))
			//no file, create emtpy map
			fs.idMaps[doc] = make(IdMap)
			return ""
		}
		var idm IdMap
		err = yaml.Unmarshal(data, idm)
		if err != nil {
			fs.log.Error(ctx, "cannot load idmap file", slog.F("err", err))
			//no file, create emtpy map
			fs.idMaps[doc] = make(IdMap)
			return ""
		}

		fs.idMaps[doc] = idm
	}
	file := fs.idMaps[doc][id]
	if file == "" && doc != fs.doc.Name {
		//included document could reference the main one, let's check it
		return fs.findIdMap(fs.doc.Name, id)
	}
	return fs.idMaps[doc][id]
}

func (fs *FileSplitter) init() error {
	fs.firstHeader = fs.findFirstHeader()
	fs.fileName = fs.getNextFileName(fs.firstHeader)
	fs.fileNames = append(fs.fileNames, fs.fileName)
	fs.fillIdMap(true)
	fs.fixUrls()
	for k, m := range fs.idMaps {
		err := fs.writeIdMap(k, m)
		if err != nil {
			return err
		}
	}
	return nil
}
