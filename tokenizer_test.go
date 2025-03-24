package xmltokenizer_test

import (
	"bytes"
	"errors"
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

func TestTokenWithInmemXML(t *testing.T) {
	tt := []struct {
		name      string
		xml       string
		expecteds []xmltokenizer.Token
		err       error
	}{
		{
			name: "dtd without entity",
			xml: `
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN"
	"http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<body xmlns:foo="ns1" xmlns="ns2" xmlns:tag="ns3" ` +
				"\r\n\t" + `  >
	<hello lang="en">World &lt;&gt;&apos;&quot; &#x767d;&#40300;翔</hello>
	<query>&何; &is-it;</query>
	<goodbye />
	<outer foo:attr="value" xmlns:tag="ns4">
	<inner/>
	</outer>
	<tag:name>
	<![CDATA[Some text here.]]>
	</tag:name>
</body><!-- missing final newline -->`, // Note: retrieved from stdlib xml test.
			expecteds: []xmltokenizer.Token{
				{
					Data:        []byte(`<?xml version="1.0" encoding="UTF-8"?>`),
					SelfClosing: true,
				},
				{
					Data: []byte("<!DOCTYPE html PUBLIC \"-//W3C//DTD XHTML 1.0 Transitional//EN\"\n" +
						"	\"http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd\">"),
					SelfClosing: true,
				},
				{
					Name: xmltokenizer.Name{Local: []byte("body"), Full: []byte("body")},
					Attrs: []xmltokenizer.Attr{
						{Name: xmltokenizer.Name{Prefix: []byte("xmlns"), Local: []byte("foo"), Full: []byte("xmlns:foo")}, Value: []byte("ns1")},
						{Name: xmltokenizer.Name{Local: []byte("xmlns"), Full: []byte("xmlns")}, Value: []byte("ns2")},
						{Name: xmltokenizer.Name{Prefix: []byte("xmlns"), Local: []byte("tag"), Full: []byte("xmlns:tag")}, Value: []byte("ns3")},
					},
				},
				{
					Name: xmltokenizer.Name{Local: []byte("hello"), Full: []byte("hello")},
					Attrs: []xmltokenizer.Attr{
						{Name: xmltokenizer.Name{Local: []byte("lang"), Full: []byte("lang")}, Value: []byte("en")},
					},
					Data: []byte("World &lt;&gt;&apos;&quot; &#x767d;&#40300;翔"),
				},
				{
					Name:         xmltokenizer.Name{Local: []byte("hello"), Full: []byte("hello")},
					IsEndElement: true,
				},
				{
					Name: xmltokenizer.Name{Local: []byte("query"), Full: []byte("query")},
					Data: []byte("&何; &is-it;"),
				},
				{
					Name:         xmltokenizer.Name{Local: []byte("query"), Full: []byte("query")},
					IsEndElement: true,
				},
				{
					Name:        xmltokenizer.Name{Local: []byte("goodbye"), Full: []byte("goodbye")},
					SelfClosing: true,
				},
				{
					Name: xmltokenizer.Name{Local: []byte("outer"), Full: []byte("outer")},
					Attrs: []xmltokenizer.Attr{
						{Name: xmltokenizer.Name{Prefix: []byte("foo"), Local: []byte("attr"), Full: []byte("foo:attr")}, Value: []byte("value")},
						{Name: xmltokenizer.Name{Prefix: []byte("xmlns"), Local: []byte("tag"), Full: []byte("xmlns:tag")}, Value: []byte("ns4")},
					},
				},
				{
					Name:        xmltokenizer.Name{Local: []byte("inner"), Full: []byte("inner")},
					SelfClosing: true,
				},
				{
					Name:         xmltokenizer.Name{Local: []byte("outer"), Full: []byte("outer")},
					IsEndElement: true,
				},
				{
					Name: xmltokenizer.Name{Prefix: []byte("tag"), Local: []byte("name"), Full: []byte("tag:name")},
					Data: []byte("Some text here."),
				},
				{
					Name:         xmltokenizer.Name{Prefix: []byte("tag"), Local: []byte("name"), Full: []byte("tag:name")},
					IsEndElement: true,
				},
				{
					Name:         xmltokenizer.Name{Local: []byte("body"), Full: []byte("body")},
					IsEndElement: true,
				},
				{
					Data:        []byte("<!-- missing final newline -->"),
					SelfClosing: true,
				},
			},
		},
		{
			name: "unexpected EOF truncated XML after `<!`",
			xml:  "<?xml version=\"1.0\" encoding=\"UTF-8\"?><!",
			expecteds: []xmltokenizer.Token{
				{
					Data:        []byte(`<?xml version="1.0" encoding="UTF-8"?>`),
					SelfClosing: true,
				},
			},
			err: io.ErrUnexpectedEOF,
		},
		{
			name: "unexpected quote before attr name",
			xml:  "<?xml version=\"1.0\" encoding=\"UTF-8\"?><a =\"ns2\"></a>",
			expecteds: []xmltokenizer.Token{
				{
					Data:        []byte(`<?xml version="1.0" encoding="UTF-8"?>`),
					SelfClosing: true,
				},
				{Name: xmltokenizer.Name{Local: []byte("a"), Full: []byte("a")}},
				{Name: xmltokenizer.Name{Local: []byte("a"), Full: []byte("a")}, IsEndElement: true},
			},
		},
		{
			name: "unexpected equals in attr name",
			xml:  "<?xml version=\"1.0\" encoding=\"UTF-8\"?><Image URL=\"https://test.com/my-url-ending-in-=\" URL2=\"https://ok.com\"/>",
			expecteds: []xmltokenizer.Token{
				{
					Data:         []byte(`<?xml version="1.0" encoding="UTF-8"?>`),
					SelfClosing:  true,
					IsEndElement: false,
				},
				{Name: xmltokenizer.Name{Local: []byte("Image"), Full: []byte("Image")},
					Attrs: []xmltokenizer.Attr{
						{
							Name:  xmltokenizer.Name{Local: []uint8("URL"), Full: []uint8("URL")},
							Value: []uint8("https://test.com/my-url-ending-in-="),
						},
						{
							Name:  xmltokenizer.Name{Local: []uint8("URL2"), Full: []uint8("URL2")},
							Value: []uint8("https://ok.com"),
						},
					},
					SelfClosing: true,
				},
			},
		},
		{
			name: "tab after node name",
			xml:  `<sample	foo="bar"/>`,
			expecteds: []xmltokenizer.Token{
				{
					Name: xmltokenizer.Name{
						Local: []uint8("sample"),
						Full:  []uint8("sample"),
					},
					Attrs: []xmltokenizer.Attr{
						{
							Name: xmltokenizer.Name{
								Local: []uint8("foo"),
								Full:  []uint8("foo")},
							Value: []uint8("bar"),
						},
					},
					SelfClosing: true,
				},
			},
		},
		{
			name: "tab after attribute value",
			xml:  `<sample foo="bar"	/>`,
			expecteds: []xmltokenizer.Token{
				{
					Name: xmltokenizer.Name{
						Local: []uint8("sample"),
						Full:  []uint8("sample"),
					},
					Attrs: []xmltokenizer.Attr{
						{
							Name: xmltokenizer.Name{
								Local: []uint8("foo"),
								Full:  []uint8("foo")},
							Value: []uint8("bar"),
						},
					},
					SelfClosing: true,
				},
			},
		},
		{
			name: "tab between attributes",
			xml:  `<sample foo="bar"	baz="quux"/>`,
			expecteds: []xmltokenizer.Token{
				{
					Name: xmltokenizer.Name{
						Local: []uint8("sample"),
						Full:  []uint8("sample"),
					},
					Attrs: []xmltokenizer.Attr{
						{
							Name: xmltokenizer.Name{
								Local: []uint8("foo"),
								Full:  []uint8("foo")},
							Value: []uint8("bar"),
						},
						{
							Name: xmltokenizer.Name{
								Local: []uint8("baz"),
								Full:  []uint8("baz")},
							Value: []uint8("quux"),
						},
					},
					SelfClosing: true,
				},
			},
		},
		{
			name: "slash inside attribute value",
			xml:  `<sample path="foo/bar/baz">`,
			expecteds: []xmltokenizer.Token{
				{
					Name: xmltokenizer.Name{Local: []byte("sample"), Full: []byte("sample")},
					Attrs: []xmltokenizer.Attr{
						{
							Name:  xmltokenizer.Name{Local: []uint8("path"), Full: []uint8("path")},
							Value: []uint8("foo/bar/baz"),
						},
					},
				},
			},
		},
		{
			name: "right angle bracket inside attribute value",
			xml:  `<sample path="foo>bar>baz">`,
			expecteds: []xmltokenizer.Token{
				{
					Name: xmltokenizer.Name{Local: []byte("sample"), Full: []byte("sample")},
					Attrs: []xmltokenizer.Attr{
						{
							Name:  xmltokenizer.Name{Local: []uint8("path"), Full: []uint8("path")},
							Value: []uint8("foo>bar>baz"),
						},
					},
				},
			},
		},
		{
			name: "right angle bracket inside comment",
			xml:  `<!-->--><!-- foo>bar>baz -->`,
			expecteds: []xmltokenizer.Token{
				{
					Data:        []byte(`<!-->-->`),
					SelfClosing: true,
				},
				{
					Data:        []byte(`<!-- foo>bar>baz -->`),
					SelfClosing: true,
				},
			},
		},
		{
			name: "left angle bracket inside comment",
			xml:  `<!--<--><!-- foo<bar<baz -->`,
			expecteds: []xmltokenizer.Token{
				{
					Data:        []byte(`<!--<-->`),
					SelfClosing: true,
				},
				{
					Data:        []byte(`<!-- foo<bar<baz -->`),
					SelfClosing: true,
				},
			},
		},
		{
			name: "angle brackets in processing instruction",
			xml:  `<?sample <foo> ?>`,
			expecteds: []xmltokenizer.Token{
				{
					Data:        []uint8("<?sample <foo> ?>"),
					SelfClosing: true,
				},
			},
		},
		{
			name: "quote in processing instruction",
			xml:  `<?sample " ?>`,
			expecteds: []xmltokenizer.Token{
				{
					Data:        []uint8(`<?sample " ?>`),
					SelfClosing: true,
				},
			},
		},
		{
			name: "quoted right angle bracket in doctype",
			xml:  `<!DOCTYPE ">" >`,
			expecteds: []xmltokenizer.Token{
				{
					Data:        []uint8(`<!DOCTYPE ">" >`),
					SelfClosing: true,
				},
			},
		},
		{
			name: "commented angle brackets in doctype",
			xml:  `<!DOCTYPE [ <!-- <foo> --> ] ><!DOCTYPE [ <!-- > --> ] ><!DOCTYPE [ <!-- < --> ] >`,
			expecteds: []xmltokenizer.Token{
				{
					Data:        []uint8(`<!DOCTYPE [ <!-- <foo> --> ] >`),
					SelfClosing: true,
				},
				{
					Data:        []uint8(`<!DOCTYPE [ <!-- > --> ] >`),
					SelfClosing: true,
				},
				{
					Data:        []uint8(`<!DOCTYPE [ <!-- < --> ] >`),
					SelfClosing: true,
				},
			},
		},
		{
			name: "commented quote in doctype",
			xml:  `<!DOCTYPE [ <!-- " --> ] >`,
			expecteds: []xmltokenizer.Token{
				{
					Data:        []uint8(`<!DOCTYPE [ <!-- " --> ] >`),
					SelfClosing: true,
				},
			},
		},
	}

	for i, tc := range tt {
		t.Run(fmt.Sprintf("[%d]: %s", i, tc.name), func(t *testing.T) {
			tok := xmltokenizer.New(
				bytes.NewReader([]byte(tc.xml)),
				xmltokenizer.WithReadBufferSize(1), // Read per char so we can cover more code paths
			)

			for i := 0; ; i++ {
				token, err := tok.Token()
				if err == io.EOF {
					if i != len(tc.expecteds) {
						t.Fatalf("expected %d tokens, got %d", len(tc.expecteds), i)
					}
					break
				}
				if err != nil {
					if !errors.Is(err, tc.err) {
						t.Fatalf("expected error: %v, got: %v", tc.err, err)
					}
					return
				}
				if diff := cmp.Diff(token, tc.expecteds[i]); diff != "" {
					t.Fatalf("%d: %s", i, diff)
				}
			}
		})
	}
}

func TestTokenWithSmallXMLFiles(t *testing.T) {
	tt := []struct {
		filename  string
		expecteds []xmltokenizer.Token
		err       error
	}{
		{filename: "cdata.xml", expecteds: []xmltokenizer.Token{
			tokenHeader,
			{Name: xmltokenizer.Name{Local: []byte("content"), Full: []byte("content")}},
			{
				Name: xmltokenizer.Name{Local: []byte("data"), Full: []byte("data")},
				Data: []byte("text"),
			},
			{
				Name:         xmltokenizer.Name{Local: []byte("data"), Full: []byte("data")},
				IsEndElement: true,
			},
			{
				Name: xmltokenizer.Name{Local: []byte("data"), Full: []byte("data")},
				Data: []byte("<element>text</element>"),
			},
			{
				Name:         xmltokenizer.Name{Local: []byte("data"), Full: []byte("data")},
				IsEndElement: true,
			},
			{
				Name: xmltokenizer.Name{Local: []byte("data"), Full: []byte("data")},
				Data: []byte("<element>text</element>"),
			},
			{
				Name:         xmltokenizer.Name{Local: []byte("data"), Full: []byte("data")},
				IsEndElement: true,
			},
			{
				Name:         xmltokenizer.Name{Local: []byte("content"), Full: []byte("content")},
				IsEndElement: true,
			},
		}},
		{filename: "cdata_clrf.xml", expecteds: []xmltokenizer.Token{
			tokenHeader,
			{Name: xmltokenizer.Name{Local: []byte("content"), Full: []byte("content")}},
			{
				Name: xmltokenizer.Name{Local: []byte("data"), Full: []byte("data")},
				Data: []byte("text"),
			},
			{
				Name:         xmltokenizer.Name{Local: []byte("data"), Full: []byte("data")},
				IsEndElement: true,
			},
			{
				Name: xmltokenizer.Name{Local: []byte("data"), Full: []byte("data")},
				Data: []byte("<element>text</element>"),
			},
			{
				Name:         xmltokenizer.Name{Local: []byte("data"), Full: []byte("data")},
				IsEndElement: true,
			},
			{
				Name: xmltokenizer.Name{Local: []byte("data"), Full: []byte("data")},
				Data: []byte("<element>text</element>"),
			},
			{
				Name:         xmltokenizer.Name{Local: []byte("data"), Full: []byte("data")},
				IsEndElement: true,
			},
			{
				Name:         xmltokenizer.Name{Local: []byte("content"), Full: []byte("content")},
				IsEndElement: true,
			},
		}},
		{filename: filepath.Join("corrupted", "cdata_truncated.xml"), expecteds: []xmltokenizer.Token{
			tokenHeader,
			{Name: xmltokenizer.Name{Local: []byte("content"), Full: []byte("content")}},
			{
				Name: xmltokenizer.Name{Local: []byte("data"), Full: []byte("data")},
			},
		},
			err: io.ErrUnexpectedEOF,
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
			{Name: xmltokenizer.Name{Local: []byte("to"), Full: []byte("to")}, IsEndElement: true},
			{Name: xmltokenizer.Name{Local: []byte("from"), Full: []byte("from")}, Data: []byte("Jani")},
			{Name: xmltokenizer.Name{Local: []byte("from"), Full: []byte("from")}, IsEndElement: true},
			{Name: xmltokenizer.Name{Local: []byte("heading"), Full: []byte("heading")}, Data: []byte("Reminder")},
			{Name: xmltokenizer.Name{Local: []byte("heading"), Full: []byte("heading")}, IsEndElement: true},
			{Name: xmltokenizer.Name{Local: []byte("body"), Full: []byte("body")}, Data: []byte("Don't forget me this weekend!")},
			{Name: xmltokenizer.Name{Local: []byte("body"), Full: []byte("body")}, IsEndElement: true},
			{Name: xmltokenizer.Name{Local: []byte("footer"), Full: []byte("footer")}, Data: []byte("&writer;&nbsp;&copyright;")},
			{Name: xmltokenizer.Name{Local: []byte("footer"), Full: []byte("footer")}, IsEndElement: true},
			{Name: xmltokenizer.Name{Local: []byte("note"), Full: []byte("note")}, IsEndElement: true},
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
					if !errors.Is(err, tc.err) {
						t.Fatalf("expected error: %v, got: %v", tc.err, err)
					}
					return
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

			data, err := os.ReadFile(path)
			if err != nil {
				t.Skip(err)
			}

			gpx1, err := gpx.UnmarshalWithXMLTokenizer(bytes.NewReader(data))
			if err != nil {
				t.Fatalf("xmltokenizer: %v", err)
			}

			gpx2, err := gpx.UnmarshalWithStdlibXML(bytes.NewReader(data))
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

	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip(err)
	}

	sheet1, err := xlsx.UnmarshalWithXMLTokenizer(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("xmltokenizer: %v", err)
	}
	sheet2, err := xlsx.UnmarshalWithStdlibXML(bytes.NewReader(data))
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
		xmltokenizer.WithReadBufferSize(1),
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

	f2, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f2.Close()

	sheetData2, err := xlsx.UnmarshalWithStdlibXML(f2)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(sheetData1, sheetData2); diff != "" {
		t.Fatal(err)
	}
}

func TestRawTokenWithInmemXML(t *testing.T) {
	tt := []struct {
		name      string
		xml       string
		expecteds []string
		err       error
	}{
		{
			name: "simple xml happy flow",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN"
	"http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<body xmlns:foo="ns1" xmlns="ns2" xmlns:tag="ns3" ` +
				"\r\n\t" + `  >
	<hello lang="en">World &lt;&gt;&apos;&quot; &#x767d;&#40300;翔</hello>
	<query>&何; &is-it;</query>
	<goodbye />
	<outer foo:attr="value" xmlns:tag="ns4">
	<inner/>
	</outer>
	<tag:name>
	<![CDATA[Some text here.]]>
	</tag:name>
</body><!-- missing final newline -->`, // Note: retrieved from stdlib xml test.
			expecteds: []string{
				"<?xml version=\"1.0\" encoding=\"UTF-8\"?>",
				"<!DOCTYPE html PUBLIC \"-//W3C//DTD XHTML 1.0 Transitional//EN\"\n" +
					"	\"http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd\">",
				"<body xmlns:foo=\"ns1\" xmlns=\"ns2\" xmlns:tag=\"ns3\" " +
					"\r\n\t" + "  >",
				"<hello lang=\"en\">World &lt;&gt;&apos;&quot; &#x767d;&#40300;翔",
				"</hello>",
				"<query>&何; &is-it;",
				"</query>",
				"<goodbye />",
				"<outer foo:attr=\"value\" xmlns:tag=\"ns4\">",
				"<inner/>",
				"</outer>",
				"<tag:name>\n	<![CDATA[Some text here.]]>",
				"</tag:name>",
				"</body>",
				"<!-- missing final newline -->",
			},
		},
		{
			name: "unexpected EOF truncated XML after `<!`",
			xml:  "<?xml version=\"1.0\" encoding=\"UTF-8\"?><!",
			expecteds: []string{
				"<?xml version=\"1.0\" encoding=\"UTF-8\"?>",
				"<!",
			},
			err: io.ErrUnexpectedEOF,
		},
	}

	for i, tc := range tt {
		t.Run(fmt.Sprintf("[%d]: %s", i, tc.name), func(t *testing.T) {
			tok := xmltokenizer.New(
				bytes.NewReader([]byte(tc.xml)),
				xmltokenizer.WithReadBufferSize(1), // Read per char so we can cover more code paths
			)

			for i := 0; ; i++ {
				token, err := tok.RawToken()
				if err == io.EOF {
					break
				}
				if err != nil {
					if !errors.Is(err, tc.err) {
						t.Fatalf("expected error: %v, got: %v", tc.err, err)
					}
					return
				}
				if diff := cmp.Diff(string(token), tc.expecteds[i]); diff != "" {
					t.Fatal(diff)
				}
			}
		})
	}

	t.Run("with prior error", func(t *testing.T) {
		// Test in case RawToken() is reinvoked when there is prior error.
		tok := xmltokenizer.New(bytes.NewReader([]byte{}))
		token, err := tok.RawToken()
		if err != io.EOF {
			t.Fatalf("expected error: %v, got: %v", io.EOF, err)
		}
		_ = token
		token, err = tok.RawToken() // Reinvoke
		if err != io.EOF {
			t.Fatalf("expected error: %v, got: %v", io.EOF, err)
		}
		_ = token
	})
}
