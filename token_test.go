package xmltokenizer_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/muktihari/xmltokenizer"
)

func TestGetToken(t *testing.T) {
	alloc := testing.AllocsPerRun(10, func() {
		token := xmltokenizer.GetToken()
		xmltokenizer.PutToken(token)
	})
	if alloc != 0 {
		t.Fatalf("expected alloc: 0, got: %g", alloc)
	}
}

func TestIsEndElement(t *testing.T) {
	tt := []struct {
		name     string
		token    xmltokenizer.Token
		expected bool
	}{
		{
			name: "an end element",
			token: xmltokenizer.Token{
				Name: xmltokenizer.Name{
					Full: []byte("worksheet"),
				},
				IsEndElement: true,
			},
			expected: true,
		},
		{
			name: "a start element",
			token: xmltokenizer.Token{
				Name: xmltokenizer.Name{
					Full: []byte("worksheet"),
				},
			},
			expected: false,
		},
		{
			name: "a procinst",
			token: xmltokenizer.Token{
				Name: xmltokenizer.Name{
					Full: []byte("?xml"),
				},
			},
			expected: false,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if r := tc.token.IsEndElement; r != tc.expected {
				t.Fatalf("expected: %t, got: %t", tc.expected, r)
			}
		})
	}
}

func TestIsEndElementOf(t *testing.T) {
	tt := []struct {
		name     string
		t1, t2   xmltokenizer.Token
		expected bool
	}{
		{
			name: "correct end element",
			t1: xmltokenizer.Token{
				Name: xmltokenizer.Name{
					Full: []byte("worksheet"),
				},
				IsEndElement: true,
			},
			t2: xmltokenizer.Token{
				Name: xmltokenizer.Name{
					Full: []byte("worksheet"),
				},
			},
			expected: true,
		},
		{
			name: "incorrect end element",
			t1: xmltokenizer.Token{
				Name: xmltokenizer.Name{
					Full: []byte("/gpx"),
				},
			},
			t2: xmltokenizer.Token{
				Name: xmltokenizer.Name{
					Full: []byte("worksheet"),
				},
			},
			expected: false,
		},
		{
			name: "not even an end element",
			t2: xmltokenizer.Token{
				Name: xmltokenizer.Name{
					Full: []byte("worksheet"),
				},
			},
			t1: xmltokenizer.Token{
				Name: xmltokenizer.Name{
					Full: []byte("worksheet"),
				},
			},
			expected: false,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if r := tc.t1.IsEndElementOf(&tc.t2); r != tc.expected {
				t.Fatalf("expected: %t, got: %t", tc.expected, r)
			}
		})
	}
}

func TestCopy(t *testing.T) {
	t1 := xmltokenizer.Token{
		Name: xmltokenizer.Name{
			Prefix: []byte("gpxtpx"),
			Local:  []byte("hr"),
			Full:   []byte("gpxtpx:hr"),
		},
		Attrs: []xmltokenizer.Attr{{
			Name: xmltokenizer.Name{
				Prefix: nil,
				Local:  []byte("units"),
				Full:   []byte("units"),
			},
			Value: []byte("bpm"),
		}},
		Data: []byte("70"),
	}

	var t2 xmltokenizer.Token
	t2.Copy(t1)

	if diff := cmp.Diff(t2, t1); diff != "" {
		t.Fatal(diff)
	}

	t2.Name.Full = append(t2.Name.Full[:0], "asd"...)
	t2.Data = append(t2.Data[:0], "60"...)
	if diff := cmp.Diff(t2, t1); diff == "" {
		t.Fatalf("expected different, got same")
	}

	// Test shallow copy, it should change the original
	t2.Attrs[0].Name.Full[0] = 'i'
	if diff := cmp.Diff(t2.Attrs, t1.Attrs); diff != "" {
		t.Fatal(diff)
	}
}
