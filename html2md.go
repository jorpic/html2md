package main

import (
  "os"
  "io"
  "golang.org/x/net/html"
  "golang.org/x/net/html/atom"
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

type parserFunc func(context)

type dispatcher struct {
  parser parserFunc
  nestedTable map[atom.Atom]dispatcher
}

var bodyTable = map[atom.Atom]dispatcher{
    0: {text, nil},
  }


func html2md(r io.Reader, w io.Writer) {
  z := html.NewTokenizer(r)
  cxt := context{
    tokenizer: z,
    writer: w,
    tagTable: bodyTable,}

  for {
    switch z.Next() {
    case html.ErrorToken:
      return
    case html.StartTagToken:
      continue
    case html.TextToken:
      d := cxt.tagTable[0]
      newCxt := cxt
      newCxt.tagTable = d.nestedTable
      d.parser(newCxt)
    case html.SelfClosingTagToken:
      // br
      continue
    default:
      continue
    }
  }
}


func text(cxt context) {
  cxt.writer.Write(cxt.tokenizer.Text())
}

