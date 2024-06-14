package xmltokenizer_test

import (
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

func TestXMLContainsCopyrightHeader(t *testing.T) {
	path := filepath.Join("testdata", "copyright_header.xml")
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	tok := xmltokenizer.New(f)
	token, err := tok.Token()
	if err != nil {
		t.Fatal(err)
	}

	if string(token.Name.Local) != "<!--" {
		t.Fatalf("expected name.Local: %q, got: %q", "<!--", string(token.Name.Local))
	}
	if string(token.Name.Full) != "<!--" {
		t.Fatalf("expected name.Full: %q, got: %q", "<!--", string(token.Name.Full))
	}
	if string(token.Data) != "Copyright 2024 Example Licence Authors." {
		t.Fatalf("expected CharData: %q, got: %q", "Copyright 2024 Example Licence Authors.",
			string(token.Data))
	}
}

func TestXMLContainsCDATA(t *testing.T) {
	path := filepath.Join("testdata", "cdata.xml")
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	tok := xmltokenizer.New(f)
	for {
		token, err := tok.Token()
		if err != nil {
			t.Fatal(err)
		}

		if string(token.Name.Local) == "data" {
			token, err = tok.Token()
			if err != nil {
				t.Fatal(err)
			}
			const cdataIdent = "<![CDATA["
			if string(token.Name.Local) != cdataIdent {
				t.Fatalf("expected name.Local: %q, got: %q", cdataIdent, string(token.Name.Local))
			}
			if string(token.Name.Full) != cdataIdent {
				t.Fatalf("expected name.Full: %q, got: %q", cdataIdent, string(token.Name.Full))
			}
			if string(token.Data) != "text" {
				t.Fatalf("expected name.Full: %q, got: %q", "text", string(token.Name.Full))
			}
			break
		}
	}

}
