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
	"strconv"
	"strings"
)

func main() {
	html2md(os.Stdin, os.Stdout)
}

type context struct {
	tokenizer *html.Tokenizer
	token     html.Token
	writer    io.Writer
	parserMap parserMap
	parent    atom.Atom
	lst       struct { // those two are ul/ol related
		level int
		order []int // ugly trick to have reference sematics
	}
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

	formattedText := make(parserMap)
	formattedTextParsers := []elemParser{
		{0, text, nil},
		{atom.B, wrap("**", "**"), formattedText},
		{atom.S, wrap("~~", "~~"), formattedText},
		{atom.Em, wrap("*", "*"), formattedText},
		{atom.Span, wrap("", ""), formattedText},
		{atom.Code, wrap("`", "`"), rawText}}
	fillMap(formattedText, formattedTextParsers)

	textAndLinks := fillMap(
		make(parserMap),
		append(formattedTextParsers,
			elemParser{atom.A, anchor, formattedText}))

	textAndLinksAndLists := make(parserMap)
	listBullets := make(parserMap)
	fillMap(textAndLinksAndLists,
		append(formattedTextParsers,
			elemParser{atom.A, anchor, formattedText},
			elemParser{atom.Ul, list, listBullets},
			elemParser{atom.Ol, list, listBullets}))
	fillMap(listBullets, []elemParser{
		elemParser{atom.Ul, list, listBullets},
		elemParser{atom.Ol, list, listBullets},
		elemParser{atom.Li, listItem, textAndLinksAndLists}})

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

// NB: preserve `nbsp`
var rxTrimSpace = regexp.MustCompile("(?m)[ \\t\\r\\n\\v]+")
var rxEscapeEmph = regexp.MustCompile("(~~|[\\\\*])")

func text(cxt context) error {
	txt := cxt.token.Data
	txt = rxTrimSpace.ReplaceAllLiteralString(txt, " ")
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
	href, ok := cxt.GetAttr("href") // FIXME: inline?
	if !ok {
		return nil
	}
	return wrap("[", "]("+href+")")(cxt)
}

func list(cxt context) error {
	cxt.lst.level++
	cxt.lst.order = []int{0}
	return wrap("", "")(cxt)
}

func listItem(cxt context) error {
	cxt.lst.order[0]++
	indent := strings.Repeat(" ", cxt.lst.level*2)
	if cxt.parent == atom.Ul {
		return wrap("\n"+indent+"- ", "")(cxt)
	}
	order := strconv.Itoa(cxt.lst.order[0]) + ". "
	return wrap("\n"+indent+order, "")(cxt)
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
