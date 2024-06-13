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

func TestTokenOnGPXFiles(t *testing.T) {
	err := filepath.Walk("testdata", func(path string, info fs.FileInfo, _ error) error {
		if info.IsDir() {
			return nil
		}
		if strings.ToLower(filepath.Ext(path)) != ".gpx" {
			return nil
		}

		gpx1, err := gpx.UnmarshalWithXMLTokenizer(path)
		if err != nil {
			return fmt.Errorf("xmltokenizer: %w", err)
		}

		gpx2, err := gpx.UnmarshalWithStdlibXML(path)
		if err != nil {
			return fmt.Errorf("xml: %w", err)
		}

		if diff := cmp.Diff(gpx1, gpx2,
			cmp.Transformer("float64", func(x float64) uint64 {
				return math.Float64bits(x)
			}),
		); diff != "" {
			t.Fatal(diff)
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
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

func TestAutoGrowBuffer(t *testing.T) {
	path := filepath.Join("testdata", "xlsx_sheet1.xml")
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	tok := xmltokenizer.New(f,
		xmltokenizer.WithReadBufferSize(5),
		xmltokenizer.WithAttrBufferSize(0),
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
