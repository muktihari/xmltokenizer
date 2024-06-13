package schema

import (
	"encoding/xml"
	"fmt"
	"io"
	"time"

	"github.com/muktihari/xmltokenizer"
)

// Metadata is GPX's Metadata schema (simplified).
type Metadata struct {
	Name   string    `xml:"name,omitempty"`
	Desc   string    `xml:"desc,omitempty"`
	Author *Author   `xml:"author,omitempty"`
	Link   *Link     `xml:"link,omitempty"`
	Time   time.Time `xml:"time,omitempty"`
}

func (m *Metadata) UnmarshalToken(tok *xmltokenizer.Tokenizer, se *xmltokenizer.Token) error {
	for {
		token, err := tok.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if token.IsEndElementOf(se) {
			return nil
		}
		if token.IsEndElement() {
			continue
		}

		switch string(token.Name.Local) {
		case "name":
			m.Name = string(token.CharData)
		case "desc":
			m.Desc = string(token.CharData)
		case "author":
			m.Author = new(Author)
			se := xmltokenizer.GetToken().Copy(token)
			err = m.Author.UnmarshalToken(tok, se)
			xmltokenizer.PutToken(se)
			if err != nil {
				return fmt.Errorf("author: %w", err)
			}
		case "link":
			m.Link = new(Link)
			se := xmltokenizer.GetToken().Copy(token)
			err = m.Link.UnmarshalToken(tok, se)
			xmltokenizer.PutToken(se)
			if err != nil {
				return fmt.Errorf("link: %w", err)
			}
		case "time":
			m.Time, err = time.Parse(time.RFC3339, string(token.CharData))
			if err != nil {
				return fmt.Errorf("time: %w", err)
			}
		}
	}

	return nil
}

func (m *Metadata) UnmarshalXML(dec *xml.Decoder, se xml.StartElement) error {
	var targetCharData string
	for {
		token, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch elem := token.(type) {
		case xml.StartElement:
			switch elem.Name.Local {
			case "author":
				m.Author = new(Author)
				if err := m.Author.UnmarshalXML(dec, elem); err != nil {
					return fmt.Errorf("author: %w", err)
				}
			case "link":
				m.Link = new(Link)
				if err := m.Link.UnmarshalXML(dec, elem); err != nil {
					return fmt.Errorf("link: %w", err)
				}
			default:
				targetCharData = elem.Name.Local
			}
		case xml.CharData:
			switch targetCharData {
			case "name":
				m.Name = string(elem)
			case "desc":
				m.Desc = string(elem)
			case "time":
				m.Time, err = time.Parse(time.RFC3339, string(elem))
				if err != nil {
					return fmt.Errorf("time: %w", err)
				}
			}
			targetCharData = ""
		case xml.EndElement:
			if elem == se.End() {
				return nil
			}
		}
	}

	return nil
}

// Author is Author schema (simplified).
type Author struct {
	Name string `xml:"name"`
	Link *Link  `xml:"link"`
}

func (a *Author) UnmarshalToken(tok *xmltokenizer.Tokenizer, se *xmltokenizer.Token) error {
	for {
		token, err := tok.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("author: %w", err)
		}

		if token.IsEndElementOf(se) {
			return nil
		}
		if token.IsEndElement() {
			continue
		}

		switch string(token.Name.Local) {
		case "name":
			a.Name = string(token.CharData)
		case "link":
			a.Link = new(Link)
			se := xmltokenizer.GetToken().Copy(token)
			err := a.Link.UnmarshalToken(tok, se)
			xmltokenizer.PutToken(se)
			if err != nil {
				return fmt.Errorf("link: %w", err)
			}
		}
	}

	return nil
}

func (a *Author) UnmarshalXML(dec *xml.Decoder, se xml.StartElement) error {
	var targetCharData string
	for {
		token, err := dec.Token()
		if err != nil {
			return err
		}

		switch elem := token.(type) {
		case xml.StartElement:
			switch elem.Name.Local {
			case "link":
				a.Link = new(Link)
				if err := a.Link.UnmarshalXML(dec, elem); err != nil {
					return fmt.Errorf("link: %w", err)
				}
			default:
				targetCharData = elem.Name.Local
			}
		case xml.CharData:
			switch targetCharData {
			case "name":
				a.Name = string(elem)
			}
			targetCharData = ""
		case xml.EndElement:
			if elem == se.End() {
				return nil
			}
		}
	}
}

// Link is Link schema.
type Link struct {
	XMLName xml.Name `xml:"link"`
	Href    string   `xml:"href,attr"`

	Text string `xml:"text,omitempty"`
	Type string `xml:"type,omitempty"`
}

func (a *Link) UnmarshalToken(tok *xmltokenizer.Tokenizer, se *xmltokenizer.Token) error {
	for i := range se.Attrs {
		attr := &se.Attrs[i]
		switch string(attr.Name.Local) {
		case "href":
			a.Href = string(attr.Value)
		}
	}

	for {
		token, err := tok.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("link: %w", err)
		}

		if token.IsEndElementOf(se) {
			return nil
		}
		if token.IsEndElement() {
			continue
		}

		switch string(token.Name.Local) {
		case "text":
			a.Text = string(token.CharData)
		case "type":
			a.Type = string(token.CharData)
		}
	}

	return nil
}

func (a *Link) UnmarshalXML(dec *xml.Decoder, se xml.StartElement) error {
	for i := range se.Attr {
		attr := &se.Attr[i]
		switch attr.Name.Local {
		case "href":
			a.Href = attr.Value
		}
	}

	var targetCharData string
	for {
		token, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch elem := token.(type) {
		case xml.StartElement:
			targetCharData = elem.Name.Local
		case xml.CharData:
			switch targetCharData {
			case "text":
				a.Text = string(elem)
			case "type":
				a.Type = string(elem)
			}
			targetCharData = ""
		case xml.EndElement:
			if elem == se.End() {
				return nil
			}
		}
	}

	return nil
}
