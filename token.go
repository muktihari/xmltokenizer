package xmltokenizer

import "sync"

var pool = sync.Pool{New: func() any { return new(Token) }}

// GetToken gets token from the pool, don't forget to put it back.
func GetToken() *Token { return pool.Get().(*Token) }

// PutToken puts token back to the pool.
func PutToken(t *Token) { pool.Put(t) }

// Token represent a single token, one of these following:
//   - <?xml version="1.0" encoding="UTF-8"?>
//   - <name attr="value" attr="value">
//   - <name attr="value" attr="value">CharData
//   - <name attr="value" attr="value"><![CDATA[ CharData ]]>
//   - <name attr="value" attr="value"/>
//   - </name>
//   - <!-- a comment -->
//   - <!DOCTYPE library [
//     <!ELEMENT library (book+)>
//     <!ELEMENT book (title, author, year)>
//     ]>
//
// Token includes CharData or CDATA in Data field when it appears right after the start element.
type Token struct {
	Name         Name   // Name is an XML name, empty when a tag starts with "<?" or "<!".
	Attrs        []Attr // Attrs exist when len(Attrs) > 0.
	Data         []byte // Data could be a CharData or a CDATA, or maybe a RawToken if a tag starts with "<?" or "<!" (except "<![CDATA").
	SelfClosing  bool   // True when a tag ends with "/>" e.g. <c r="E3" s="1" />. Also true when a tag starts with "<?" or "<!" (except "<![CDATA").
	IsEndElement bool   // True when a tag start with "</" e.g. </gpx> or </gpxtpx:atemp>.
}

// IsEndElementOf checks whether the given token represent a
// n end element (closing tag) of given StartElement.
func (t *Token) IsEndElementOf(se *Token) bool {
	if t.IsEndElement &&
		string(t.Name.Full) == string(se.Name.Full) {
		return true
	}
	return false
}

// Copy copies src Token into t, returning t. Attrs should be
// consumed immediately since it's only being shallow copied.
func (t *Token) Copy(src Token) *Token {
	t.Name.Prefix = append(t.Name.Prefix[:0], src.Name.Prefix...)
	t.Name.Local = append(t.Name.Local[:0], src.Name.Local...)
	t.Name.Full = append(t.Name.Full[:0], src.Name.Full...)
	t.Attrs = append(t.Attrs[:0], src.Attrs...) // shallow copy
	t.Data = append(t.Data[:0], src.Data...)
	t.SelfClosing = src.SelfClosing
	t.IsEndElement = src.IsEndElement
	return t
}

// Attr represents an XML attribute.
type Attr struct {
	Name  Name
	Value []byte
}

// Name represents an XML name <prefix:local>,
// we don't manage the bookkeeping of namespaces.
type Name struct {
	Prefix []byte
	Local  []byte
	Full   []byte // Full is combination of "prefix:local"
}
