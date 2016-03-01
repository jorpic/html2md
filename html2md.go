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
	parent    atom.Atom
	tokenizer *html.Tokenizer
	token     html.Token
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

func (cxt *context) GetAttr(name string) (string, bool) {
	for _, attr := range cxt.token.Attr {
		if attr.Key == name {
			return attr.Val, true
		}
	}
	return "", false
}

type parserFunc func(context) error

type dispatcher struct {
	parser      parserFunc
	nestedTable map[atom.Atom]dispatcher
}

var textTable = map[atom.Atom]dispatcher{
	0: {text, nil},
}

var textLinkTable = map[atom.Atom]dispatcher{
	0:      {text, nil},
	atom.A: {anchor, textTable},
}

var bodyTable = map[atom.Atom]dispatcher{
	0:       {text, nil},
	atom.A:  {anchor, textTable},
	atom.H1: {h1_2("="), textLinkTable},
	atom.H2: {h1_2("-"), textLinkTable},
	atom.H3: {h3_5(3), textLinkTable},
	atom.H4: {h3_5(4), textLinkTable},
	atom.H5: {h3_5(5), textLinkTable},
}

func html2md(r io.Reader, w io.Writer) {
	z := html.NewTokenizer(r)
	cxt := context{
		tokenizer: z,
		writer:    w,
		tagTable:  bodyTable}

	for dispatch(cxt) == nil {
	}
}

func dispatch(cxt context) error {
	tt := cxt.tokenizer.Next()
	cxt.token = cxt.tokenizer.Token()
	switch tt {
	case html.ErrorToken:
		return errEndOfStream // FIXME: check tkz.Err()
	case html.StartTagToken:
		tag := cxt.token.DataAtom
		d, ok := cxt.tagTable[tag]
		if ok {
			newCxt := cxt
			newCxt.tagTable = d.nestedTable
			return d.parser(newCxt)
		}
	case html.TextToken:
		d, ok := cxt.tagTable[0]
		if ok {
			newCxt := cxt
			newCxt.tagTable = d.nestedTable
			return d.parser(newCxt)
		}
	case html.EndTagToken:
		if cxt.token.DataAtom == cxt.parent {
			return okEndOfContext
		}
	case html.SelfClosingTagToken:
		// FIXME: br
	default:
	}
	return nil
}

func text(cxt context) error {
	return cxt.WriteStrings(cxt.token.Data)
}

func anchor(cxt context) error {
	href, ok := cxt.GetAttr("href")
	if !ok {
		return errSomeError
	}

	buf, err := goDeeper(&cxt)
	if err != nil {
		return err
	}
	txt := buf.String()
	return cxt.WriteStrings("[", txt, "](", href, ")")
}

func h1_2(subChar string) parserFunc {
	return func(cxt context) error {
		buf, err := goDeeper(&cxt)
		if err != nil {
			return err
		}
		txt := buf.String()
		sub := strings.Repeat(subChar, len(txt))
		return cxt.WriteStrings("\n", txt, "\n", sub, "\n")
	}
}

func h3_5(level int) parserFunc {
	return func(cxt context) error {
		buf, err := goDeeper(&cxt)
		if err != nil {
			return err
		}
		txt := buf.String()
		pre := strings.Repeat("#", level)
		return cxt.WriteStrings("\n", pre, " ", txt, "\n")
	}
}

// FIXME: goDeeper copies `cxt` just before it is copeid in `dispatch`
func goDeeper(cxt *context) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	newCxt := *cxt
	newCxt.writer = buf
	newCxt.parent = cxt.token.DataAtom
	for {
		err := dispatch(newCxt)
		if err == okEndOfContext {
			return buf, nil
		}
		if err != nil {
			return buf, err
		}
	}
}

var errEndOfStream = fmt.Errorf("end of stream")
var errSomeError = fmt.Errorf("some error")
var okEndOfContext = fmt.Errorf("end of context")
