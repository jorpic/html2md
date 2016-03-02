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

func html2md(r io.Reader, w io.Writer) {
	z := html.NewTokenizer(r)
	cxt := context{
		tokenizer: z,
		writer:    w,
		parserMap: topHTML}

	// To convert HTML we are looping `dispatch` on html tokenizer
	// until there are no more HTML tokens.
	for {
		if err := dispatch(cxt); err != nil {
			if err != errEndOfStream {
				log.Fatal(err)
			}
			return
		}
	}
}

// Dispatch takes current HTML token and, depending on its type,
// applies corresponding parser.
// Parsers use `dispatch` corecursively to scan tokens further (until matching
// closing tag).
func dispatch(cxt context) error {
	tt := cxt.tokenizer.Next()
	cxt.token = cxt.tokenizer.Token()
	switch tt {
	case html.ErrorToken:
		return errEndOfStream // FIXME: check tkz.Err()
	case html.StartTagToken, html.TextToken:
		tag := cxt.token.DataAtom
		// NB. here we are relying on undocumented feature:
		// `token.DataAtom == 0` for `TextToken`.
		d, ok := cxt.parserMap[tag]
		if ok { // skip unknown tokens
			// create *new* context with parsers for child elements
			newCxt := cxt
			newCxt.parserMap = d.parserMap
			return d.parser(newCxt)
		}
	case html.EndTagToken:
		if cxt.token.DataAtom == cxt.parent {
			return errEndOfContext
		}
	case html.SelfClosingTagToken:
		// TODO: br
	}
	return nil
}

// context is a bunch of fields representing current parser state
type context struct {
	tokenizer *html.Tokenizer
	writer    io.Writer
	// current token
	token html.Token
	// map of parsers that we can apply within current token
	parserMap parserMap
	parent    atom.Atom
	lst       struct {
		// track indent-level of ul/ol list
		level int
		// index of current list item in ordered list
		// FIXME?: using slice for single integer
		//         is just an ugly trick to have reference sematics
		order []int
	}
}

// Just a couple of helper functions.
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

// Here we are at the core of our converter.

// elemParser is a main building block of converter.
// It contains `parser` function that is able to parse current `tag`,
// and a `parserMap` that contains parsers for nested elements.
type elemParser struct {
	tag       atom.Atom
	parser    parserFunc
	parserMap parserMap
}

// parserFunc takes tokens out of context converts them and writes them
// to the `context.writer`.
type parserFunc func(context) error
type parserMap map[atom.Atom]elemParser

// topHTML is much like a BNF grammar describing accepted document format.
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
		{atom.I, wrap("_", "_"), formattedText},
		{atom.Em, wrap("*", "*"), formattedText},
		{atom.Span, wrap("", ""), formattedText},
		{atom.Code, wrap("`", "`"), rawText},
		{atom.Pre, wrap("\n```\n", "\n```\n"), rawText}}
	fillMap(formattedText, formattedTextParsers)

	textAndLinks := fillMap(
		make(parserMap),
		append(formattedTextParsers,
			elemParser{atom.A, anchor, formattedText}))

	textAndLinksAndLists := make(parserMap)
	listBullets := make(parserMap)
	fillMap(textAndLinksAndLists,
		append(formattedTextParsers, []elemParser{
			{atom.A, anchor, formattedText},
			{atom.Ul, list, listBullets},
			{atom.Ol, list, listBullets}}...))
	fillMap(listBullets, []elemParser{
		{atom.Ul, list, listBullets},
		{atom.Ol, list, listBullets},
		{atom.Li, listItem, textAndLinksAndLists}})

	topHTMLParsers := append(formattedTextParsers, []elemParser{
		{atom.Script, skip, nil},
		{atom.Head, skip, nil},
		{atom.A, anchor, formattedText},
		{atom.H1, h1_2("="), textAndLinks},
		{atom.H2, h1_2("-"), textAndLinks},
		{atom.H3, wrap("\n\n### ", "\n"), textAndLinks},
		{atom.H4, wrap("\n\n#### ", "\n"), textAndLinks},
		{atom.H5, wrap("\n\n##### ", "\n"), textAndLinks},
		{atom.P, wrap("\n", "\n"), textAndLinksAndLists},
		{atom.Pre, wrap("\n```", "```\n"), rawText},
		{atom.Ul, list, listBullets},
		{atom.Ol, list, listBullets}}...)
	fillMap(topHTML, topHTMLParsers)
}

func fillMap(m parserMap, xs []elemParser) parserMap {
	for _, x := range xs {
		m[x.tag] = x
	}
	return m
}

// Those below are parser combinators.

func skip(cxt context) error {
	_, err := goDeeper(&cxt)
	return err
}

// NB: preserve `nbsp`
var rxTrimSpace = regexp.MustCompile("(?m)[ \\t\\r\\n\\v]+")
var rxEscapeEmph = regexp.MustCompile("(~~|[\\\\*_])")

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
		return cxt.WriteStrings("\n\n", txt, "\n", sub, "\n")
	}
}

// goDeeper calls `dispatch` in a loop until matching closing tag.
// Returns converted nested elements in a bytes.Buffer.
func goDeeper(cxt *context) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	// FIXME: do we really need to copy `cxt` just before it is copeid in
	//        `dispatch`?
	newCxt := *cxt
	newCxt.writer = buf
	newCxt.parent = cxt.token.DataAtom
	for {
		err := dispatch(newCxt)
		if err == errEndOfContext {
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
var errEndOfContext = fmt.Errorf("end of context")
