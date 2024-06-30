# Usage

The usage of this library is similar to the standard library's xml manual implementation of `xml.Unmarshaler` interface, with a slightly different code.

Let's say we have this xml schema, a simplified version of `xlsx's sheet1.xml`.

```xml
<?xml version="1.0" encoding="UTF-8"?>
<row r="1">
  <c r="A1">
    <v>0</v>
  </c>
  <c r="B1">
    <v>4</v>
  </c>
  <c r="C1" />
</row>
```

We can write the Go implementation like following:

```go
package main

import (
    "bytes"
    "fmt"
    "io"
    "strconv"

    "github.com/muktihari/xmltokenizer"
)

const sample = `<?xml version="1.0" encoding="UTF-8"?>
<row r="1">
  <c r="A1">
    <v>0</v>
  </c>
  <c r="B1">
    <v>4</v>
  </c>
  <c r="C1" />
</row>`

func main() {
    f := bytes.NewReader([]byte(sample))

    tok := xmltokenizer.New(f)
    var row Row
loop:
    for {
        token, err := tok.Token() // Token is only valid until next tok.Token() invocation (short-lived object).
        if err == io.EOF {
            break
        }
        if err != nil {
            panic(err)
        }
        switch string(token.Name.Local) { // This do not allocate 🥳👍
        case "row":
            // Reuse Token object in the sync.Pool since we only use it temporarily.
            se := xmltokenizer.GetToken().Copy(token) // se: StartElement, we should copy it since token is a short-lived object.
            err = row.UnmarshalToken(tok, se)
            xmltokenizer.PutToken(se) // Put back to sync.Pool.
            if err != nil {
                panic(err)
            }
            break loop
        }
    }
    fmt.Printf("row: %+v\n", row)
    // Output:
    // row: {Index:1 Cells:[{Reference:A1 Value:0} {Reference:B1 Value:4} {Reference:C1 Value:}]}
}

type Row struct {
    Index int    `xml:"r,attr,omitempty"`
    Cells []Cell `xml:"c"`
}

func (r *Row) UnmarshalToken(tok *xmltokenizer.Tokenizer, se *xmltokenizer.Token) error {
    var err error
    for i := range se.Attrs {
        attr := &se.Attrs[i]
        switch string(attr.Name.Local) {
        case "r":
            r.Index, err = strconv.Atoi(string(attr.Value))
            if err != nil {
                return err
            }
        }
    }

    for {
        token, err := tok.Token()
        if err != nil {
            return err
        }
        if token.IsEndElementOf(se) { // Reach desired EndElement
            return nil
        }
        if token.IsEndElement { // Ignore child's EndElements
            continue
        }
        switch string(token.Name.Local) {
        case "c":
            var cell Cell
            // Reuse Token object in the sync.Pool since we only use it temporarily.
            se := xmltokenizer.GetToken().Copy(token)
            err = cell.UnmarshalToken(tok, se)
            xmltokenizer.PutToken(se) // Put back to sync.Pool.
            if err != nil {
                return err
            }
            r.Cells = append(r.Cells, cell)
        }
    }
}

type Cell struct {
    Reference string `xml:"r,attr"`
    Value     string `xml:"v,omitempty"`
}

func (c *Cell) UnmarshalToken(tok *xmltokenizer.Tokenizer, se *xmltokenizer.Token) error {
    for i := range se.Attrs {
        attr := &se.Attrs[i]
        switch string(attr.Name.Local) {
        case "r":
            c.Reference = string(attr.Value)
        }
    }

    // Must check since `c` may contains self-closing tag:
    // <c r="C1" />
    if se.SelfClosing {
        return nil
    }

    for {
        token, err := tok.Token()
        if err != nil {
            return err
        }
        if token.IsEndElementOf(se) { // Reach desired EndElement
            return nil
        }
        if token.IsEndElement { // Ignore child's EndElements
            continue
        }
        switch string(token.Name.Local) {
        case "v":
            c.Value = string(token.Data)
        }
    }
}

```

You can find more examples in [internal](../internal/README.md) package.
