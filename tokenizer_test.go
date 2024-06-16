package xmltokenizer_test

import (
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/muktihari/xmltokenizer"
	"github.com/muktihari/xmltokenizer/internal/gpx"
	"github.com/muktihari/xmltokenizer/internal/xlsx"
	"github.com/muktihari/xmltokenizer/internal/xlsx/schema"
)

var tokenHeader = xmltokenizer.Token{Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>`), SelfClosing: true}

func TestSmallXML(t *testing.T) {
	tt := []struct {
		filename  string
		expecteds []xmltokenizer.Token
	}{
		{filename: "cdata.xml", expecteds: []xmltokenizer.Token{
			tokenHeader,
			{
				Name: xmltokenizer.Name{Local: []byte("data"), Full: []byte("data")},
				Data: []byte("text"),
			},
			{Name: xmltokenizer.Name{Local: []byte("/data"), Full: []byte("/data")}}},
		},
		{filename: "self_closing.xml", expecteds: []xmltokenizer.Token{
			tokenHeader,
			{Name: xmltokenizer.Name{Local: []byte("a"), Full: []byte("a")}, SelfClosing: true},
			{Name: xmltokenizer.Name{Local: []byte("b"), Full: []byte("b")}, SelfClosing: true},
		}},
		{filename: "copyright_header.xml", expecteds: []xmltokenizer.Token{
			{Data: []byte("<!--\n  Copyright 2024 Example Licence Authors.\n-->"), SelfClosing: true},
			tokenHeader,
		}},
		{filename: "dtd.xml", expecteds: []xmltokenizer.Token{
			tokenHeader,
			{
				Data: []byte("<!DOCTYPE note [\n" +
					"  <!ENTITY nbsp \"&#xA0;\">\n" +
					"  <!ENTITY writer \"Writer: Donald Duck.\">\n" +
					"  <!ENTITY copyright \"Copyright: W3Schools.\">\n" +
					"]>"),
				SelfClosing: true,
			},
			{Name: xmltokenizer.Name{Local: []byte("note"), Full: []byte("note")}},
			{Name: xmltokenizer.Name{Local: []byte("to"), Full: []byte("to")}, Data: []byte("Tove")},
			{Name: xmltokenizer.Name{Local: []byte("/to"), Full: []byte("/to")}},
			{Name: xmltokenizer.Name{Local: []byte("from"), Full: []byte("from")}, Data: []byte("Jani")},
			{Name: xmltokenizer.Name{Local: []byte("/from"), Full: []byte("/from")}},
			{Name: xmltokenizer.Name{Local: []byte("heading"), Full: []byte("heading")}, Data: []byte("Reminder")},
			{Name: xmltokenizer.Name{Local: []byte("/heading"), Full: []byte("/heading")}},
			{Name: xmltokenizer.Name{Local: []byte("body"), Full: []byte("body")}, Data: []byte("Don't forget me this weekend!")},
			{Name: xmltokenizer.Name{Local: []byte("/body"), Full: []byte("/body")}},
			{Name: xmltokenizer.Name{Local: []byte("footer"), Full: []byte("footer")}, Data: []byte("&writer;&nbsp;&copyright;")},
			{Name: xmltokenizer.Name{Local: []byte("/footer"), Full: []byte("/footer")}},
			{Name: xmltokenizer.Name{Local: []byte("/note"), Full: []byte("/note")}},
		}},
	}

	for i, tc := range tt {
		t.Run(fmt.Sprintf("[%d], %s", i, tc.filename), func(t *testing.T) {
			path := filepath.Join("testdata", tc.filename)
			f, err := os.Open(path)
			if err != nil {
				panic(err)
			}
			defer f.Close()

			tok := xmltokenizer.New(f, xmltokenizer.WithReadBufferSize(1))
			for i := 0; ; i++ {
				token, err := tok.Token()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				if diff := cmp.Diff(token, tc.expecteds[i]); diff != "" {
					t.Fatal(diff)
				}
			}
		})
	}
}

func TestTokenOnGPXFiles(t *testing.T) {
	filepath.Walk("testdata", func(path string, info fs.FileInfo, _ error) error {
		t.Run(path, func(t *testing.T) {
			if info.IsDir() {
				return
			}
			if strings.ToLower(filepath.Ext(path)) != ".gpx" {
				return
			}

			gpx1, err := gpx.UnmarshalWithXMLTokenizer(path)
			if err != nil {
				t.Fatalf("xmltokenizer: %v", err)
			}

			gpx2, err := gpx.UnmarshalWithStdlibXML(path)
			if err != nil {
				t.Fatalf("xml: %v", err)
			}

			if diff := cmp.Diff(gpx1, gpx2,
				cmp.Transformer("float64", func(x float64) uint64 {
					return math.Float64bits(x)
				}),
			); diff != "" {
				t.Fatal(diff)
			}
		})

		return nil
	})
}

func TestTokenOnXLSXFiles(t *testing.T) {
	path := filepath.Join("testdata", "xlsx_sheet1.xml")

	sheet1, err := xlsx.UnmarshalWithXMLTokenizer(path)
	if err != nil {
		t.Fatalf("xmltokenizer: %v", err)
	}
	sheet2, err := xlsx.UnmarshalWithStdlibXML(path)
	if err != nil {
		t.Fatalf("xml: %v", err)
	}

	if diff := cmp.Diff(sheet1, sheet2); diff != "" {
		t.Fatal(diff)
	}
}

func TestAutoGrowBufferCorrectness(t *testing.T) {
	path := filepath.Join("testdata", "xlsx_sheet1.xml")
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	tok := xmltokenizer.New(f,
		xmltokenizer.WithReadBufferSize(5),
	)

	var token xmltokenizer.Token
	var sheetData1 schema.SheetData
loop:
	for {
		token, err = tok.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}

		switch string(token.Name.Local) {
		case "sheetData":
			se := xmltokenizer.GetToken().Copy(token)
			err = sheetData1.UnmarshalToken(tok, se)
			xmltokenizer.PutToken(se)
			if err != nil {
				t.Fatal(err)
			}
			break loop
		}
	}

	sheetData2, err := xlsx.UnmarshalWithStdlibXML(path)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(sheetData1, sheetData2); diff != "" {
		t.Fatal(err)
	}
}
