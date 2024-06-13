package schema

import (
	"io"
	"strconv"

	"github.com/muktihari/xmltokenizer"
)

type SheetData struct {
	Rows []Row `xml:"row,omitempty"`
}

func (s *SheetData) UnmarshalToken(tok *xmltokenizer.Tokenizer, se *xmltokenizer.Token) error {
	for {
		token, err := tok.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if token.IsEndElementOf(se) {
			break
		}
		if token.IsEndElement() {
			continue
		}

		switch string(token.Name.Local) {
		case "row":
			var row Row
			se := xmltokenizer.GetToken().Copy(token)
			err = row.UnmarshalToken(tok, se)
			xmltokenizer.PutToken(se)
			if err != nil {
				return err
			}
			s.Rows = append(s.Rows, row)
		}
	}
	return nil
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
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if token.IsEndElementOf(se) {
			break
		}
		if token.IsEndElement() {
			continue
		}

		switch string(token.Name.Local) {
		case "c":
			var cell Cell
			se := xmltokenizer.GetToken().Copy(token)
			err = cell.UnmarshalToken(tok, se)
			xmltokenizer.PutToken(se)
			if err != nil {
				return err
			}
			r.Cells = append(r.Cells, cell)
		}
	}

	return nil
}

type Cell struct {
	Reference    string `xml:"r,attr"` // E.g. A1
	Style        int    `xml:"s,attr"`
	Type         string `xml:"t,attr,omitempty"`
	Value        string `xml:"v,omitempty"`
	InlineString string `xml:"is>t"`
}

func (c *Cell) UnmarshalToken(tok *xmltokenizer.Tokenizer, se *xmltokenizer.Token) error {
	var err error
	for i := range se.Attrs {
		attr := &se.Attrs[i]
		switch string(attr.Name.Local) {
		case "r":
			c.Reference = string(attr.Value)
		case "s":
			c.Style, err = strconv.Atoi(string(attr.Value))
			if err != nil {
				return err
			}
		case "t":
			c.Type = string(attr.Value)
		}
	}

	// Must check since `c` may contains self-closing tag:
	// <c r="C1" />
	if se.SelfClosing {
		return nil
	}

	for {
		token, err := tok.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if token.IsEndElementOf(se) {
			break
		}
		if token.IsEndElement() {
			continue
		}

		switch string(token.Name.Local) {
		case "v":
			c.Value = string(token.CharData)
		case "t":
			c.InlineString = string(token.CharData)
		}
	}

	return nil
}
