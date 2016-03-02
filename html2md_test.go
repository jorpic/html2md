package main

import (
	"bytes"
	"strings"
	t "testing"
)

func TestText(t *t.T) {
	check(t, "some text", "some text")
}

func TestText_escape(t *t.T) {
	check(t, "n\\a ~ *text*", "n\\\\a ~ \\*text\\*")
}

func TestEmph(t *t.T) {
	check(t,
		"<b>bold</b> <s>strikeout</s> <em>emph</em>",
		"**bold** ~~strikeout~~ *emph*")
}

func TestEmph_nested(t *t.T) {
	check(t,
		"<b>bold & <s>strikeout</s></b> <em><b>em</b>phasis</em>",
		"**bold & ~~strikeout~~** ***em**phasis*")
}

func TestEmph_code(t *t.T) {
	check(t,
		"<code>~~this *is* code~~</code>",
		"`~~this *is* code~~`")
}

func TestP(t *t.T) {
	check(t,
		"<p>First paragraph</p><p>Second <span>para</span>graph</p>",
		"\nFirst paragraph\n\nSecond paragraph\n")
}

func TestUl(t *t.T) {
	check(t,
		`<ul>
  <li>first <b>item</b></li>
  <li>second item with a <a href="http://example.com">link</a></li>
</ul>
`,
		`
  - first **item**
  - second item with a [link](http://example.com) `)
}

func TestOl(t *t.T) {
	check(t,
		`<ol>
  <li>first <b>item</b></li>
  <li>second item with a <a href="http://example.com">link</a></li>
</ol>
`,
		`
  1. first **item**
  2. second item with a [link](http://example.com) `)
}

func TestOl_Ul(t *t.T) {
	check(t,
		`<ol>
  <li>first <b>item</b></li>
  <li>second</li>
  <ul><li>sub one</li><li>sub two</li></ul>
  <li>third item
    <ul>
      <li>with a <a href="http://example.com">link</a></li>
      <li>and a profit</li>
    </ul>
  </li>
</ol>
`,
		`
  1. first **item**
  2. second
    - sub one
    - sub two
  3. third item 
    - with a [link](http://example.com)
    - and a profit  `)
}

func TestH1(t *t.T) {
	check(t,
		"<h1>Hello!</h1>",
		"\n\nHello!\n======\n")
}

func TestH1_withLink(t *t.T) {
	check(t,
		"<h1><a href='http://ya.ru'>Hello!</a></h1>",
		"\n\n[Hello!](http://ya.ru)\n======================\n")
}

func TestH1_withLinkAndText(t *t.T) {
	check(t,
		"<h1>Hello <a href='http://ya.ru'>there</a>!</h1>",
		"\n\nHello [there](http://ya.ru)!\n============================\n")
}

func TestH2(t *t.T) {
	check(t,
		"<h2>Hello!</h2>",
		"\n\nHello!\n------\n")
}

func TestH4_H5(t *t.T) {
	check(t,
		"<h4>Section</h4><h5>Subsection</h5>",
		"\n\n#### Section\n\n\n##### Subsection\n")
}

func TestH5(t *t.T) {
	check(t,
		"<h5>Hello!</h5>",
		"\n\n##### Hello!\n")
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
		t.Errorf("\n%s\n!=\n%s", res, out)
	}
}
