package main

import (
	"bytes"
	"fmt"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
)

func main() {
	html2md(os.Stdin, os.Stdout)
}

type context struct {
	parent    atom.Atom
	level     int
	tokenizer *html.Tokenizer
	token     html.Token
	writer    io.Writer
	parserMap parserMap
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
type parserMap map[atom.Atom]elemParser

type elemParser struct {
	tag       atom.Atom
	parser    parserFunc
	parserMap parserMap
}

// topHTML is like a BNF grammar of accepted document format.
// It is not a tree but a graph, hence it is initialized in `init()` is some
// obscure manner.
var topHTML = make(parserMap)

func init() {
	rawText := parserMap{0: {0, rawText, nil}}

	var formattedText = make(parserMap)
	formattedTextParsers := []elemParser{
		{0, text, nil},
		{atom.B, wrap("**", "**"), formattedText},
		{atom.S, wrap("~~", "~~"), formattedText},
		{atom.Em, wrap("*", "*"), formattedText},
		{atom.Span, wrap("", ""), formattedText},
		{atom.Code, wrap("`", "`"), rawText}}
	fillMap(formattedText, formattedTextParsers)

	var textAndLinks = fillMap(
		make(parserMap),
		append(formattedTextParsers,
			elemParser{atom.A, anchor, formattedText}))

	var textAndLinksAndLists = make(parserMap)
	var listBullets = make(parserMap)
	fillMap(textAndLinksAndLists,
		append(formattedTextParsers,
			elemParser{atom.A, anchor, formattedText},
			elemParser{atom.Ul, list, listBullets},
			elemParser{atom.Ol, list, listBullets}))
	fillMap(listBullets,
		append(formattedTextParsers,
			elemParser{atom.A, anchor, formattedText},
			elemParser{atom.Li, listItem, textAndLinksAndLists}))

	topHTMLParsers := append(formattedTextParsers,
		elemParser{atom.Script, skip, nil},
		elemParser{atom.Head, skip, nil},
		elemParser{atom.A, anchor, formattedText},
		elemParser{atom.H1, h1_2("="), textAndLinks},
		elemParser{atom.H2, h1_2("-"), textAndLinks},
		elemParser{atom.H3, wrap("\n### ", "\n"), textAndLinks},
		elemParser{atom.H4, wrap("\n#### ", "\n"), textAndLinks},
		elemParser{atom.H5, wrap("\n##### ", "\n"), textAndLinks},
		elemParser{atom.P, wrap("\n", "\n"), textAndLinksAndLists},
		elemParser{atom.Ul, list, listBullets},
		elemParser{atom.Ol, list, listBullets})
	fillMap(topHTML, topHTMLParsers)
}

func fillMap(m parserMap, xs []elemParser) parserMap {
	for _, x := range xs {
		m[x.tag] = x
	}
	return m
}

func html2md(r io.Reader, w io.Writer) {
	z := html.NewTokenizer(r)
	cxt := context{
		tokenizer: z,
		writer:    w,
		parserMap: topHTML}

	for {
		if err := dispatch(cxt); err != nil {
			if err != errEndOfStream {
				log.Fatal(err)
			}
			return
		}
	}
}

func dispatch(cxt context) error {
	tt := cxt.tokenizer.Next()
	cxt.token = cxt.tokenizer.Token()
	switch tt {
	case html.ErrorToken:
		return errEndOfStream // FIXME: check tkz.Err()
	case html.StartTagToken, html.TextToken:
		tag := cxt.token.DataAtom
		d, ok := cxt.parserMap[tag]
		if ok {
			newCxt := cxt
			newCxt.parserMap = d.parserMap
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

func skip(cxt context) error {
	_, err := goDeeper(&cxt)
	return err
}

var rxEscapeEmph = regexp.MustCompile("(~~|[\\\\*])")

func text(cxt context) error {
	txt := cxt.token.Data
	txt = rxEscapeEmph.ReplaceAllString(txt, "\\$1")
	return cxt.WriteStrings(txt)
}

func rawText(cxt context) error {
	return cxt.WriteStrings(cxt.token.Data)
}

func wrap(xx string, yy string) parserFunc {
	return func(cxt context) error {
		buf, err := goDeeper(&cxt)
		if err != nil {
			return err
		}
		return cxt.WriteStrings(xx, buf.String(), yy)
	}
}

func anchor(cxt context) error {
	href, ok := cxt.GetAttr("href")
	if !ok {
		return nil
	}

	buf, err := goDeeper(&cxt)
	if err != nil {
		return err
	}
	txt := buf.String()
	return cxt.WriteStrings("[", txt, "](", href, ")")
}

func list(cxt context) error {
	return nil
}

func listItem(cxt context) error {
	return nil
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
		// FIXME: catch unexpected end of file
	}
}

var errEndOfStream = fmt.Errorf("end of stream")
var errSomeError = fmt.Errorf("some error")
var okEndOfContext = fmt.Errorf("end of context")
