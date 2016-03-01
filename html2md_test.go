package main

import (
	"bytes"
	"strings"
	t "testing"
)

func TestText(t *t.T) {
	check(t, "some text", "some text")
}

func TestH1(t *t.T) {
	check(t,
		"<h1>Hello!</h1>",
		"\nHello!\n======\n")
}

func TestH1_withLink(t *t.T) {
	check(t,
		"<h1><a href='http://ya.ru'>Hello!</a></h1>",
		"\n[Hello!](http://ya.ru)\n======================\n")
}

func TestH1_withLinkAndText(t *t.T) {
	check(t,
		"<h1>Hello <a href='http://ya.ru'>there</a>!</h1>",
		"\nHello [there](http://ya.ru)!\n============================\n")
}

func TestH2(t *t.T) {
	check(t,
		"<h2>Hello!</h2>",
		"\nHello!\n------\n")
}

func TestH4_H5(t *t.T) {
	check(t,
		"<h4>Section</h4><h5>Subsection</h5>",
		"\n#### Section\n\n##### Subsection\n")
}

func TestH5(t *t.T) {
	check(t,
		"<h5>Hello!</h5>",
		"\n##### Hello!\n")
}

func TestLink(t *t.T) {
	check(t,
		"<a href='http://ya.ru'>ya.ru</a>",
		"[ya.ru](http://ya.ru)")
}

func TestLink_withText(t *t.T) {
	check(t,
		"Click <a href='http://ya.ru'>here</a> please.",
		"Click [here](http://ya.ru) please.")
}

func check(t *t.T, in, out string) {
	w := new(bytes.Buffer)
	html2md(strings.NewReader(in), w)
	res := w.String()
	if out != res {
		t.Errorf("\n%s\n != \n%s", res, out)
	}
}
