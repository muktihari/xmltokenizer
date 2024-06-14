# XML Tokenizer

![GitHub Workflow Status](https://github.com/muktihari/xmltokenizer/workflows/CI/badge.svg)
[![Go Reference](https://pkg.go.dev/badge/github.com/muktihari/xmltokenizer.svg)](https://pkg.go.dev/github.com/muktihari/xmltokenizer)
[![CodeCov](https://codecov.io/gh/muktihari/xmltokenizer/branch/master/graph/badge.svg)](https://codecov.io/gh/muktihari/xmltokenizer)
[![Go Report Card](https://goreportcard.com/badge/github.com/muktihari/xmltokenizer)](https://goreportcard.com/report/github.com/muktihari/xmltokenizer)

XML Tokenizer is a low-memory high performance library for parsing simple XML 1.0. This is an alternative option to the standard library's xml when speed is your main concern. This may not cover all XML files, but it can cover typical XML files.

# Motivation

Go provides a standard library for [XML](https://pkg.go.dev/encoding/xml) parsing, however, I've found it to be slow for my use case. I work with a lot of GPX files in my personal project to retrieve my workouts data; GPX is an XML-based file format. When parsing my 14MB GPX file containing 208km ride using the standard library's xml, it takes roughly 600ms which is super slow and it needs 2.8mil alloc!. I need an alternative library for parsing XML that's faster than standard library's `xml`, suitable for typical XML parsing tasks and no code should be made unsafe.

# Usage

Please see [USAGE.md](./docs/USAGE.md).

# Benchmark

```js
goos: darwin; goarch: amd64; pkg: xmltokenizer
cpu: Intel(R) Core(TM) i5-5257U CPU @ 2.70GHz
Benchmark/stdlib.xml:"ride_sembalun.gpx"-4    2  605913816 ns/op  110562568 B/op  2806823 allocs/op
Benchmark/xmltokenizer:"ride_sembalun.gpx"-4  8  141616068 ns/op   17143609 B/op       85 allocs/op
```

Approx. 4 times faster!
