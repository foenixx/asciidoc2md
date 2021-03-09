package main

import (
	"asciidoc2md/ast"
	"asciidoc2md/markdown"
	"bufio"
	"cdr.dev/slog"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type FileSplitter struct {
	doc       *ast.Document
	headerMap map[string]string //header -> file name
	idMap     map[string]string //header or bookmark id -> file name
	log       slog.Logger
	slug      string
	path		string //output path
	level     int //split at the specified level headers
	firstHeader *ast.Header
	fileIndex 	int
	fileName 	string //current fileName
	fileNames	[]string //all the filenames
	file 		*os.File 	//current file
	w 			*bufio.Writer  	//current writer
}

func NewFileSplitter(doc *ast.Document, nameSlug string, headerMap map[string]string, path string, log slog.Logger) *FileSplitter {
	return &FileSplitter{
		doc: doc,
		headerMap: headerMap,
		idMap: make(map[string]string),
		level: 2,
		log: log,
		slug: nameSlug,
		path: path}
}

func (fs *FileSplitter) RenderMarkdown(imagePath string) error {
	fs.init()
	err := fs.nextFile()
	if err != nil {
		return err
	}
	defer fs.Close()

	conv := markdown.New(imagePath, fs.idMap, fs.log, func(header *ast.Header) io.Writer {
		if header.Level < fs.level { //former 1 level
			header.Text = "<skip>"
		}
		if header.Level == fs.level && header != fs.firstHeader {
			err := fs.nextFile()
			if err != nil {
				fs.log.Error(context.Background(), err.Error())
				return nil
			}
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
	name, ok := fs.headerMap[h.Text]
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
	fs.file, err = os.Create(filepath.Join(fs.path, fs.fileName))
	if err != nil {
		return err
	}

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
	for _, b := range fs.doc.Blocks {
		h, ok := b.(*ast.Header)
		if ok && h.Level == fs.level {
			return h
		}
	}
	return nil
}

func (fs *FileSplitter) init() {
	fs.firstHeader = fs.findFirstHeader()
	fs.fileName = fs.getNextFileName(fs.firstHeader)
	fs.fileNames = append(fs.fileNames, fs.fileName)
	str := fmt.Sprintf("    - %s: %s\n", fs.firstHeader.Text, fs.fileName)

	fs.doc.Walk(func(b ast.Block) {
		switch b.(type) {
		case *ast.Header:
			hd := b.(*ast.Header)
			fs.log.Debug(context.Background(), "walking by header", slog.F("header", hd))
			if hd.Id != "" {
				fs.idMap[hd.Id] = fs.fileName
			}
			if hd.Level == fs.level && hd != fs.firstHeader {
				fs.fileName = fs.getNextFileName(hd)
				fs.fileNames = append(fs.fileNames, fs.fileName)
				str += fmt.Sprintf("    - %s: %s\n", hd.Text, fs.fileName)
			}
		case *ast.Bookmark:
			fs.log.Debug(context.Background(), "walking by bookmark", slog.F("header", b.(*ast.Bookmark)))
			fs.idMap[b.(*ast.Bookmark).Literal] = fs.fileName
		}
	})

	fs.fileIndex = 0
	fmt.Print(str)
}
