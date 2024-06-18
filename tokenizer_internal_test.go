package xmltokenizer

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestOptions(t *testing.T) {
	tt := []struct {
		name            string
		options         []Option
		expectedOptions options
	}{
		{
			name:            "defaultOptions",
			expectedOptions: defaultOptions(),
		},
		{
			name: "less than 0",
			options: []Option{
				WithReadBufferSize(-1),
				WithAttrBufferSize(-1),
				WithAutoGrowBufferMaxLimitSize(-1),
			},
			expectedOptions: options{
				readBufferSize:             defaultReadBufferSize,
				autoGrowBufferMaxLimitSize: autoGrowBufferMaxLimitSize,
				attrsBufferSize:            defaultAttrsBufferSize,
			},
		},
		{
			name: "readBufferSize > maxLimitGrowBufferSize",
			options: []Option{
				WithReadBufferSize(4 << 10),
				WithAutoGrowBufferMaxLimitSize(1 << 10),
			},
			expectedOptions: options{
				readBufferSize:             4 << 10,
				autoGrowBufferMaxLimitSize: 4 << 10,
				attrsBufferSize:            defaultAttrsBufferSize,
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			tok := New(nil, tc.options...)
			if diff := cmp.Diff(tok.options, tc.expectedOptions,
				cmp.AllowUnexported(options{}),
			); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestAutoGrowBuffer(t *testing.T) {
	tt := []struct {
		name     string
		filename string
		opts     []Option
		err      error
	}{
		{
			name:     "grow buffer with alloc",
			filename: "long_comment_token.xml",
			opts: []Option{
				WithReadBufferSize(5),
			},
			err: nil,
		},
		{
			name:     "grow buffer exceed max limit",
			filename: "long_comment_token.xml",
			opts: []Option{
				WithReadBufferSize(5),
				WithAutoGrowBufferMaxLimitSize(5),
			},
			err: errAutoGrowBufferExceedMaxLimit,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			f, err := os.Open(filepath.Join("testdata", tc.filename))
			if err != nil {
				panic(err)
			}
			defer f.Close()

			tok := New(f, tc.opts...)
			for {
				_, err = tok.Token()
				if err == io.EOF {
					err = nil
					break
				}
				if err != nil {
					break
				}
			}

			if !errors.Is(err, tc.err) {
				t.Fatalf("expected error: %v, got: %v", tc.err, err)
			}
		})
	}
}

type fnReader func(b []byte) (n int, err error)

func (f fnReader) Read(b []byte) (n int, err error) { return f(b) }

func TestReset(t *testing.T) {
	r := fnReader(func(b []byte) (n int, err error) { return len(b), nil })
	tok := New(r)
	tok.Token() // Trigger make buffer init, cause grow buffer by alloc up to max limit: 1MB

	tok.Reset(r,
		WithReadBufferSize(1024),
		WithAutoGrowBufferMaxLimitSize(4),
	)

	if expected := 1024; len(tok.buf) != expected {
		t.Fatalf("expected len(t.buf): %d, got: %d", expected, len(tok.buf))
	}
	if expected := 1000 << 10; cap(tok.buf) != expected {
		t.Fatalf("expected cap(t.buf): %d, got: %d", expected, cap(tok.buf))
	}

	if tok.cur != 0 {
		t.Fatalf("expected cur: %d, got: cur: %d",
			0, tok.cur)
	}

	newBufferSize := 2000 << 10
	tok.Reset(r,
		WithReadBufferSize(newBufferSize),
		WithAutoGrowBufferMaxLimitSize(4),
	)

	tok.Token() // Trigger manageBuffer

	if expected := newBufferSize; len(tok.buf) != expected {
		t.Fatalf("expected len(t.buf): %d, got: %d", expected, len(tok.buf))
	}
	if expected := newBufferSize + defaultReadBufferSize; cap(tok.buf) != expected {
		t.Fatalf("expected len(t.buf): %d, got: %d", expected, len(tok.buf))
	}
}
