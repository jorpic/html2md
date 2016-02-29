package main

import (
	"bytes"
	"fmt"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"io"
	"os"
	"strings"
)

func main() {
	html2md(os.Stdin, os.Stdout)
}

type context struct {
	stack     []atom.Atom
	tokenizer *html.Tokenizer
	writer    io.Writer
	tagTable  map[atom.Atom]dispatcher
}

func (cxt *context) WriteStrings(ss ...string) error {
	for _, s := range ss {
		if _, err := io.WriteString(cxt.writer, s); err != nil {
			return err
		}
	}
	return nil
}

type parserFunc func(context) error

type dispatcher struct {
	parser      parserFunc
	nestedTable map[atom.Atom]dispatcher
}

var textTable = map[atom.Atom]dispatcher{
	0: {text, nil},
}

var bodyTable = map[atom.Atom]dispatcher{
	0:       {text, nil},
	atom.H1: {h1, textTable},
	atom.H2: {h2, textTable},
	atom.H5: {h5, textTable},
}

func html2md(r io.Reader, w io.Writer) {
	z := html.NewTokenizer(r)
	cxt := context{
		tokenizer: z,
		writer:    w,
		tagTable:  bodyTable}

	for dispatch(cxt) != nil {
	}
}

func dispatch(cxt context) error {
	switch cxt.tokenizer.Next() {
	case html.ErrorToken:
		return errEndOfStream // FIXME: check tkz.Err()
	case html.StartTagToken:
		tag := cxt.tokenizer.Token().DataAtom
		d, ok := cxt.tagTable[tag]
		if ok {
			newCxt := cxt
			newCxt.tagTable = d.nestedTable
			return d.parser(newCxt)
		}
	case html.TextToken:
		d := cxt.tagTable[0]
		newCxt := cxt
		newCxt.tagTable = d.nestedTable
		d.parser(newCxt)
	case html.SelfClosingTagToken:
		// br
	default:
	}
	return nil
}

func text(cxt context) error {
	_, err := cxt.writer.Write(cxt.tokenizer.Text())
	return err
}

func h1(cxt context) error {
	buf, err := goDeeper(&cxt)
	if err != nil {
		return err
	}
	txt := buf.String()
	sub := strings.Repeat("=", len(txt))
	return cxt.WriteStrings("\n", txt, "\n", sub, "\n")
}

func h2(cxt context) error {
	buf, err := goDeeper(&cxt)
	if err != nil {
		return err
	}
	txt := buf.String()
	sub := strings.Repeat("-", len(txt))
	return cxt.WriteStrings("\n", txt, "\n", sub, "\n")
}

func h5(cxt context) error {
	buf, err := goDeeper(&cxt)
	if err != nil {
		return err
	}
	txt := buf.String()
	return cxt.WriteStrings("\n##### ", txt, "\n")
}

func goDeeper(cxt *context) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	newCxt := *cxt
	newCxt.writer = buf
	return buf, dispatch(newCxt)
}

var errEndOfStream = fmt.Errorf("end of stream")
