package main

import (
  "os"
  "io"
  "golang.org/x/net/html"
  "golang.org/x/net/html/atom"
  "strings"
  "bytes"
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
    writer: w,
    tagTable: bodyTable,}

  for dispatch(cxt) {
  }
}

func dispatch(cxt context) bool {
  switch cxt.tokenizer.Next() {
  case html.ErrorToken:
    return false
  case html.StartTagToken:
    tag := cxt.tokenizer.Token().DataAtom
    d, ok := cxt.tagTable[tag]
    if ok {
      newCxt := cxt
      newCxt.tagTable = d.nestedTable
      d.parser(newCxt)
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
  return true
}


func text(cxt context) {
  cxt.writer.Write(cxt.tokenizer.Text())
}


func h1(cxt context) {
  w   := cxt.writer

  buf := new(bytes.Buffer)
  newCxt := cxt
  newCxt.writer = buf
  dispatch(newCxt)
  txt := buf.String()

  io.WriteString(w, "\n")
  io.WriteString(w, txt)
  io.WriteString(w, "\n")
  io.WriteString(w, strings.Repeat("=", len(txt)))
  io.WriteString(w, "\n")
}

func h2(cxt context) {
  buf := new(bytes.Buffer)
  newCxt := cxt
  newCxt.writer = buf
  dispatch(newCxt)
  txt := buf.String()

  w   := cxt.writer
  io.WriteString(w, "\n")
  io.WriteString(w, txt)
  io.WriteString(w, "\n")
  io.WriteString(w, strings.Repeat("-", len(txt)))
  io.WriteString(w, "\n")
}

func h5(cxt context) {
  buf := new(bytes.Buffer)
  newCxt := cxt
  newCxt.writer = buf
  dispatch(newCxt)
  txt := buf.String()

  w := cxt.writer
  io.WriteString(w, "\n##### ")
  io.WriteString(w, txt)
  io.WriteString(w, "\n")
}

