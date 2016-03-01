package main

import (
  "log"
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
	0: {text, nil},
	atom.A:  {anchor, textTable},
}

var bodyTable = map[atom.Atom]dispatcher{
	0:       {text, nil},
	atom.A:  {anchor, textTable},
	atom.H1: {h1, textLinkTable},
	atom.H2: {h2, textLinkTable},
	atom.H5: {h5, textLinkTable},
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
    log.Printf("Tag: %s", tag.String())
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
var errSomeError = fmt.Errorf("some error")
