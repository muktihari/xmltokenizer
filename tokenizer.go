package xmltokenizer

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

type errorString string

func (e errorString) Error() string { return string(e) }

const (
	errAutoGrowBufferExceedMaxLimit = errorString("auto grow buffer exceed max limit")
)

const (
	defaultReadBufferSize      = 4 << 10
	autoGrowBufferMaxLimitSize = 1000 << 10
	defaultAttrsBufferSize     = 16
)

// Tokenizer is a XML tokenizer.
type Tokenizer struct {
	r       io.Reader // reader provided by the client
	n       int64     // the n read bytes counter
	options options   // tokenizer's options
	buf     []byte    // buffer that will grow as needed, large enough to hold a token (default max limit: 1MB)
	cur     int       // cursor byte position
	err     error     // last encountered error
	token   Token     // shared token
}

type options struct {
	readBufferSize             int
	autoGrowBufferMaxLimitSize int
	attrsBufferSize            int
}

func defaultOptions() options {
	return options{
		readBufferSize:             defaultReadBufferSize,
		autoGrowBufferMaxLimitSize: autoGrowBufferMaxLimitSize,
		attrsBufferSize:            defaultAttrsBufferSize,
	}
}

// Option is Tokenizer option.
type Option func(o *options)

// WithReadBufferSize directs XML Tokenizer to this buffer size
// to read from the io.Reader. Default: 4096.
func WithReadBufferSize(size int) Option {
	if size <= 0 {
		size = defaultReadBufferSize
	}
	return func(o *options) { o.readBufferSize = size }
}

// WithAutoGrowBufferMaxLimitSize directs XML Tokenizer to limit
// auto grow buffer to not grow exceed this limit. Default: 1 MB.
func WithAutoGrowBufferMaxLimitSize(size int) Option {
	if size <= 0 {
		size = autoGrowBufferMaxLimitSize
	}
	return func(o *options) { o.autoGrowBufferMaxLimitSize = size }
}

// WithAttrBufferSize directs XML Tokenizer to use this Attrs
// buffer capacity as its initial size. Default: 8.
func WithAttrBufferSize(size int) Option {
	if size <= 0 {
		size = defaultAttrsBufferSize
	}
	return func(o *options) { o.attrsBufferSize = size }
}

// New creates new XML tokenizer.
func New(r io.Reader, opts ...Option) *Tokenizer {
	t := new(Tokenizer)
	t.Reset(r, opts...)
	return t
}

// Reset resets the Tokenizer, maintaining storage for
// future tokenization to reduce memory alloc.
func (t *Tokenizer) Reset(r io.Reader, opts ...Option) {
	t.r, t.err = r, nil
	t.n, t.cur = 0, 0

	t.options = defaultOptions()
	for i := range opts {
		opts[i](&t.options)
	}

	if cap(t.token.Attrs) < t.options.attrsBufferSize {
		t.token.Attrs = make([]Attr, 0, t.options.attrsBufferSize)
	}
	if t.options.readBufferSize > t.options.autoGrowBufferMaxLimitSize {
		t.options.autoGrowBufferMaxLimitSize = t.options.readBufferSize
	}

	switch size := t.options.readBufferSize; {
	case cap(t.buf) >= size+defaultReadBufferSize:
		t.buf = t.buf[:size:cap(t.buf)]
	default:
		// Create buffer with additional cap since we need to memmove remaining bytes
		t.buf = make([]byte, size, size+defaultReadBufferSize)
	}
}

// Token returns either a valid token or an error.
// The returned token is only valid before next
// Token or RawToken method invocation.
func (t *Tokenizer) Token() (token Token, err error) {
	if t.err != nil {
		return token, t.err
	}

	b, err := t.RawToken()
	if err != nil {
		if !errors.Is(err, io.EOF) {
			err = fmt.Errorf("byte pos %d: %w", t.n, err)
		}
		if len(b) == 0 || errors.Is(err, io.ErrUnexpectedEOF) {
			return
		}
		t.err = err
	}

	t.clearToken()

	b = t.consumeNonTagIdentifier(b)
	if len(b) > 0 {
		b = t.consumeTagName(b)
		b = t.consumeAttrs(b)
		t.consumeCharData(b)
	}

	token = t.token
	if len(token.Attrs) == 0 {
		token.Attrs = nil
	}
	if len(token.Data) == 0 {
		token.Data = nil
	}

	return token, nil
}

// RawToken returns token in its raw bytes. At the end,
// it may returns last token bytes and an error.
// The returned token bytes is only valid before next
// Token or RawToken method invocation.
func (t *Tokenizer) RawToken() ([]byte, error) {
	if t.err != nil {
		return nil, t.err
	}
	var pivot int
	for {
		// Find opening <
		p := bytes.IndexByte(t.buf[t.cur:], '<')
		if p == -1 {
			t.memmoveRemainingBytes(t.cur)
			t.err = t.manageBuffer()
			if t.err == nil {
				continue
			}
			return nil, t.err
		}
		pivot = t.cur + p
		break
	}
	for {
		// Find closing >
		pos := t.findTokenEnd(pivot)
		if pos == -1 {
			pivot, pos = t.memmoveRemainingBytes(pivot)
			t.err = t.manageBuffer()
			if t.err == nil {
				continue
			}
			if errors.Is(t.err, io.EOF) {
				t.err = io.ErrUnexpectedEOF
			}
			return t.buf[pivot:pos], t.err
		}
		switch t.buf[pivot+1] {
		default:
			pivot, pos = t.parseCharData(pivot, pos)
			pos++
		case '?', '!':
		}
		buf := trim(t.buf[pivot:pos:cap(t.buf)])
		t.cur = pos
		return buf, nil
	}
}

// findTokenEnd returns the index of the first character after the
// token started at the given position, or -1 if more data needs
// to be buffered.
func (t *Tokenizer) findTokenEnd(pivot int) int {
	left := pivot + 1 // left-hand bound on the search area
	for {
		var right int // the candidate end point
		if p := bytes.IndexByte(t.buf[left:], '>'); p == -1 {
			return -1
		} else {
			right = left + p + 1
		}
		switch t.buf[pivot+1] {
		case '?':
			// is a processing instruction
			if right >= pivot+3 && t.buf[right-2] == '?' {
				return right
			}
			// this > is not part of the closing ?>
			left = right
			continue
		case '!':
			if len(t.buf) > pivot+4 && t.buf[pivot+2] == '-' && t.buf[pivot+3] == '-' {
				// is a comment
				if right >= pivot+6 && t.buf[right-3] == '-' && t.buf[right-2] == '-' {
					return right
				}
				// this > is not part of the closing -->
				left = right
				continue
			}
			// is DOCTYPE, ENTITY etc
			p := bytes.IndexByte(t.buf[left+1:right-1], '<')
			if p != -1 {
				left = t.findTokenEnd(left + p + 1)
				if left == -1 {
					return -1
				}
				// this > is part of a nested tag
				continue
			}
		}
		if bytes.Count(t.buf[left:right], []byte{'"'})%2 == 0 {
			return right
		}
		// this > is within a quoted value, scan to closing quote
		p := bytes.IndexByte(t.buf[right:], '"')
		if p == -1 {
			return -1
		}
		left = right + p + 1
	}
}

// parseCharData parses the next character sequence and if it represents
// CharData or <![CDATA[ CharData ]]>, this method will include it in the previous token.
// It returns the new pivot and new position.
func (t *Tokenizer) parseCharData(pivot, pos int) (newPivot, newPos int) {
	for i := pos; ; i++ {
		if i >= len(t.buf) {
			pivot, i = t.memmoveRemainingBytes(pivot)
			pos = i - 1
			if t.err = t.manageBuffer(); t.err != nil {
				break
			}
		}
		if t.buf[i] != '<' {
			continue
		}

		pos = i - 1
		// Might be in the form of <![CDATA[ CharData ]]>
		const prefix, suffix = "<![CDATA[", "]]>"
		var k int = 1
		for j := i + 1; ; j++ {
			if j >= len(t.buf) {
				prevLast := len(t.buf)
				pivot, j = t.memmoveRemainingBytes(pivot)
				pos = pos - (prevLast - len(t.buf))
				if t.err = t.manageBuffer(); t.err != nil {
					if errors.Is(t.err, io.EOF) {
						t.err = io.ErrUnexpectedEOF
					}
					break
				}
			}
			if k < len(prefix) {
				if t.buf[j] != prefix[k] {
					break
				}
				k++
				continue
			}
			if t.buf[j] == '>' && string(t.buf[j-2:j+1]) == suffix {
				pos = j
				break
			}
		}
		break
	}
	return pivot, pos
}

func (t *Tokenizer) memmoveRemainingBytes(pivot int) (cur, last int) {
	if pivot == 0 {
		return t.cur, len(t.buf)
	}
	n := copy(t.buf, t.buf[pivot:])
	t.buf = t.buf[:n:cap(t.buf)]
	t.cur = 0
	return t.cur, len(t.buf)
}

func (t *Tokenizer) manageBuffer() error {
	growSize := len(t.buf) + t.options.readBufferSize
	start, end := len(t.buf), growSize
	switch {
	case growSize <= cap(t.buf): // Grow by reslice
		t.buf = t.buf[:growSize:cap(t.buf)]
	default: // Grow by make new alloc
		if growSize > t.options.autoGrowBufferMaxLimitSize {
			return fmt.Errorf("could not grow buffer to %d, max limit is set to %d: %w",
				growSize, t.options.autoGrowBufferMaxLimitSize, errAutoGrowBufferExceedMaxLimit)
		}
		buf := make([]byte, growSize)
		n := copy(buf, t.buf)
		t.buf = buf
		start, end = n, cap(t.buf)
	}

	n, err := io.ReadAtLeast(t.r, t.buf[start:end], 1)
	t.buf = t.buf[: start+n : cap(t.buf)]
	t.n += int64(n)

	return err
}

func (t *Tokenizer) clearToken() {
	t.token.Name.Prefix = nil
	t.token.Name.Local = nil
	t.token.Name.Full = nil
	t.token.Attrs = t.token.Attrs[:0]
	t.token.Data = nil
	t.token.SelfClosing = false
	t.token.IsEndElement = false
}

// consumeNonTagIdentifier consumes identifier starts with "<?" or "<!", make it raw data.
func (t *Tokenizer) consumeNonTagIdentifier(b []byte) []byte {
	if len(b) < 2 || (string(b[:2]) != "<?" && string(b[:2]) != "<!") {
		return b
	}
	t.token.Data = b
	t.token.SelfClosing = true
	return nil
}

func (t *Tokenizer) consumeTagName(b []byte) []byte {
	var pos, fullpos int
	for i := 0; i < len(b); i++ {
		switch b[i] {
		case '<':
			if b[i+1] == '/' {
				t.token.IsEndElement = true
				i++
			}
			pos = i + 1
			fullpos = i + 1
		case ':':
			t.token.Name.Prefix = trim(b[pos:i])
			pos = i + 1
		case '>', ' ', '\t', '\r', '\n': // e.g. <gpx>, <trkpt lat="-7.1872750" lon="110.3450230">
			if b[i] == '>' && b[i-1] == '/' { // In case we encounter <name/>
				i--
			}
			t.token.Name.Local = trim(b[pos:i])
			t.token.Name.Full = trim(b[fullpos:i])
			return b[i:]
		}
	}
	return b
}

func (t *Tokenizer) consumeAttrs(b []byte) []byte {
	var prefix, local, full []byte
	var pos, fullpos int
	for i := 0; i < len(b); i++ {
		switch b[i] {
		case ':':
			prefix = trim(b[pos:i])
			pos = i + 1
		case '=':
			local = trim(b[pos:i])
			full = trim(b[fullpos:i])
			pos = i + 1
		case '"':
			for {
				i++
				if i+1 == len(b) {
					return nil
				}
				if b[i] == '"' {
					break
				}
			}
			if len(full) == 0 { // Ignore malformed attr
				continue
			}
			t.token.Attrs = append(t.token.Attrs, Attr{
				Name:  Name{Prefix: prefix, Local: local, Full: full},
				Value: trim(b[pos+1 : i]),
			})
			prefix, local, full = nil, nil, nil
			pos = i + 1
			fullpos = i + 1
		case '/':
			t.token.SelfClosing = true
		case '>':
			return b[i+1:]
		}
	}
	return b
}

func (t *Tokenizer) consumeCharData(b []byte) {
	const prefix, suffix = "<![CDATA[", "]]>"
	b = trimPrefix(b)
	if len(b) >= len(prefix) && string(b[:len(prefix)]) == prefix {
		b = b[len(prefix):]
	}
	if end := len(b) - len(suffix); end >= 0 && string(b[end:]) == suffix {
		b = b[:end]
	}
	t.token.Data = trim(b)
}

func trim(b []byte) []byte {
	b = trimPrefix(b)
	b = trimSuffix(b)
	return b
}

func trimPrefix(b []byte) []byte {
	var start int
	for i := 0; i < len(b); i++ {
		switch b[i] {
		case '\r':
			if i+1 < len(b) && b[i+1] == '\n' {
				start += 2
				i++
			}
		case '\n', ' ', '\t':
			start++
		default:
			return b[start:]
		}
	}
	return b[start:]
}

func trimSuffix(b []byte) []byte {
	var end int = len(b)
	for i := len(b) - 1; i >= 0; i-- {
		switch b[i] {
		case '\n':
			end--
			if i-1 > 0 && b[i-1] == '\r' {
				end--
			}
		case ' ', '\t':
			end--
		default:
			return b[:end]
		}
	}
	return b[:end]
}
