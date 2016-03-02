
[![build status](https://travis-ci.org/jorpic/html2md.svg)](https://travis-ci.org/jorpic/html2md)

**html2md** is a tool to convert some unspecified subset of HTML
to some unspecified subset of Markdown.

It is small (~300 LOC) and easily extensible.  
Due to it's streaming nature **html2md** can cope with huge HTML files.


Usage
-------

This command will install html2md binary into `$GOPATH/bin`:

```
go get -u github.com/jorpic/html2md
```

Now you can use it like this:

```
curl -s -L "https://en.wikipedia.org/wiki/Special:Random" | html2md
curl -s -L http://membrana.ru | iconv -f windows-1251 | html2md
curl -s -L https://golang.org/doc |  html2md
```


Disclaimer
----------

At the current point of time the tool is quite fragile.  It is the users's
responsibility to ensure that the provided HTML is well-formed and UTF-8 encoded.

Just don't be evil and assume "Garbage In Garbage Out":
some corner cases or malformed HTML may lead to ridiculous results.

The project is in pre-alpha state and still work in progress, please don't
hesitate to [report](https://github.com/jorpic/html2md/issues/new) bugs and feature requests.



The MIT License
---------------

Copyright (c) 2016 Max Taldykin

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
