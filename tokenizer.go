package xmltokenizer

import (
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
	defaultAttrsBufferSize     = 8
)

// Tokenizer is a XML tokenizer.
type Tokenizer struct {
	r         io.Reader // reader provided by the client
	options   options   // tokenizer's options
	buf       []byte    // buffer that will grow as needed, large enough to hold a token (default max limit: 1MB)
	cur, last int       // cur and last bytes positions
	err       error     // last encountered error
	token     Token     // shared token
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
	t.cur, t.last = 0, 0

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
			return token, err
		}
		t.err = io.EOF
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
// it may returns last token bytes and an io.EOF error.
// The returned token bytes is only valid before next
// Token or RawToken method invocation.
func (t *Tokenizer) RawToken() (b []byte, err error) {
	if t.err != nil {
		return nil, err
	}

	pos := t.cur
	var off int
	for {
		if pos >= t.last {
			off, pos = t.memmoveRemainingBytes(off)
			if err = t.manageBuffer(); err != nil {
				t.err = err
				return t.buf[off:pos], err
			}
		}
		switch t.buf[pos] {
		case '<':
			off = pos

			// Check if tag represents Document Type Definition (DTD)
			const prefix, _ = "<!DOCTYPE", "]>"
			dtdOff := 0
			var k int = 1
			for i := pos + 1; ; i++ {
				if i >= t.last {
					prevLast := t.last
					off, i = t.memmoveRemainingBytes(off)
					dtdOff = dtdOff - (prevLast - t.last)
					if err = t.manageBuffer(); err != nil {
						t.err = err
						break
					}
				}
				if k < len(prefix) {
					if t.buf[i] != prefix[k] {
						k = 0
						break
					}
					k++
					continue
				}
				switch t.buf[i] {
				case ']':
					dtdOff = i
				case '>':
					if dtdOff == i-1 && t.buf[dtdOff] == ']' {
						buf := trim(t.buf[off : i+1 : cap(t.buf)])
						t.cur = i + 1
						return buf, err
					}
				}
			}
		case '>':
			// If next char represents CharData, include it.
			for i := pos + 1; ; i++ {
				if i >= t.last {
					off, i = t.memmoveRemainingBytes(off)
					pos = i - 1
					if err = t.manageBuffer(); err != nil {
						t.err = err
						break
					}
				}
				if t.buf[i] == '<' {
					pos = i - 1
					// Might be in the form of <![CDATA[ CharData ]]>
					const prefix = "<![CDATA["
					var k int = 1
					for j := i + 1; ; j++ {
						if j >= t.last {
							prevLast := t.last
							off, j = t.memmoveRemainingBytes(off)
							pos = pos - (prevLast - t.last)
							if err = t.manageBuffer(); err != nil {
								t.err = err
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
						if t.buf[j] == '>' {
							pos = j
							break
						}
					}
					break
				}
			}
			buf := trim(t.buf[off : pos+1 : cap(t.buf)])
			t.cur = pos + 1
			return buf, err
		}
		pos++
	}
}

func (t *Tokenizer) clearToken() {
	t.token.Name.Space = nil
	t.token.Name.Local = nil
	t.token.Name.Full = nil
	t.token.Attrs = t.token.Attrs[:0]
	t.token.Data = nil
	t.token.SelfClosing = false
}

func (t *Tokenizer) memmoveRemainingBytes(off int) (cur, last int) {
	if off == 0 {
		return t.cur, t.last
	}
	n := copy(t.buf, t.buf[off:])
	t.buf = t.buf[:n:cap(t.buf)]
	t.cur, t.last = 0, n
	return t.cur, t.last
}

func (t *Tokenizer) manageBuffer() error {
	var start, end int
	switch growSize := t.last + t.options.readBufferSize; {
	case growSize <= cap(t.buf): // Grow by reslice
		t.buf = t.buf[:growSize:cap(t.buf)]
		start, end = t.last, growSize
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
	t.last = len(t.buf)

	return err
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
	for i := range b {
		switch b[i] {
		case '<':
			pos = i + 1
			fullpos = i + 1
		case ':':
			t.token.Name.Space = trim(b[pos:i])
			pos = i + 1
		case '>', ' ': // e.g. <gpx>, <trkpt lat="-7.1872750" lon="110.3450230">
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
	var space, local, full []byte
	var pos, fullpos int
	var inquote bool
	for i := range b {
		switch b[i] {
		case ':':
			if !inquote {
				space = trim(b[pos:i])
				pos = i + 1
			}
		case '=':
			local = trim(b[pos:i])
			full = trim(b[fullpos:i])
			pos = i + 1
		case '"':
			inquote = !inquote
			if !inquote {
				if full == nil {
					continue
				}
				t.token.Attrs = append(t.token.Attrs, Attr{
					Name:  Name{Space: space, Local: local, Full: full},
					Value: trim(b[pos+1 : i]),
				})
				space, local, full = nil, nil, nil
				pos = i + 1
				fullpos = i + 1
			}
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
	for i := range b {
		switch b[i] {
		case '\r':
			if i+1 < len(b) && b[i+1] == '\n' {
				start += 2
			}
		case '\n', ' ':
			start++
		default:
			return b[start:]
		}
	}
	return b
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
		case ' ':
			end--
		default:
			return b[:end]
		}
	}
	return b
}
