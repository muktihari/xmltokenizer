package xmltokenizer_test

import (
	"encoding/xml"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/muktihari/xmltokenizer"
	"github.com/muktihari/xmltokenizer/internal/gpx"
	"github.com/muktihari/xmltokenizer/internal/xlsx"
)

func BenchmarkToken(b *testing.B) {
	filepath.Walk("testdata", func(path string, info fs.FileInfo, _ error) error {
		if info.IsDir() {
			return nil
		}
		name := strings.TrimPrefix(path, "testdata/")
		b.Run(fmt.Sprintf("stdlib.xml:%q", name), func(b *testing.B) {
			var err error
			for i := 0; i < b.N; i++ {
				if err = unmarshalWithStdlibXML(path); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(fmt.Sprintf("xmltokenizer:%q", name), func(b *testing.B) {
			var err error
			for i := 0; i < b.N; i++ {
				if err = unmarshalWithXMLTokenizer(path); err != nil {
					b.Fatal(err)
				}
			}
		})
		return nil
	})
}

func unmarshalWithXMLTokenizer(path string) error {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	tok := xmltokenizer.New(f)
	for {
		token, err := tok.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		_ = token
	}
	return nil
}

func unmarshalWithStdlibXML(path string) error {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	dec := xml.NewDecoder(f)
	for {
		token, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		_ = token
	}
	return nil
}

func BenchmarkUnmarshalGPX(b *testing.B) {
	filepath.Walk("testdata", func(path string, info fs.FileInfo, _ error) error {
		if info.IsDir() {
			return nil
		}
		if strings.ToLower(filepath.Ext(path)) != ".gpx" {
			return nil
		}

		name := strings.TrimPrefix(path, "testdata/")

		b.Run(fmt.Sprintf("stdlib.xml:%q", name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = gpx.UnmarshalWithStdlibXML(path)
			}
		})
		b.Run(fmt.Sprintf("xmltokenizer:%q", name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = gpx.UnmarshalWithXMLTokenizer(path)
			}
		})

		return nil
	})
}

func BenchmarkUnmarshalXLSX(b *testing.B) {
	path := filepath.Join("testdata", "xlsx_sheet1.xml")
	name := strings.TrimPrefix(path, "testdata/")

	b.Run(fmt.Sprintf("stdlib.xml:%q", name), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = xlsx.UnmarshalWithStdlibXML(path)
		}
	})
	b.Run(fmt.Sprintf("xmltokenizer:%q", name), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = xlsx.UnmarshalWithXMLTokenizer(path)
		}
	})
}
