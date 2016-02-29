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

func TestH2(t *t.T) {
	check(t,
		"<h2>Hello!</h2>",
		"\nHello!\n------\n")
}

func TestH5(t *t.T) {
	check(t,
		"<h5>Hello!</h5>",
		"\n##### Hello!\n")
}

func check(t *t.T, in, out string) {
	w := new(bytes.Buffer)
	html2md(strings.NewReader(in), w)
	res := w.String()
	if out != res {
		t.Errorf("\n%s\n != \n%s", res, out)
	}
}
