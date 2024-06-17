package schema

import (
	"encoding/xml"
	"fmt"
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
		if err != nil {
			return fmt.Errorf("metadata: %w", err)
		}

		if token.IsEndElementOf(se) {
			return nil
		}
		if token.IsEndElement() {
			continue
		}

		switch string(token.Name.Local) {
		case "name":
			m.Name = string(token.Data)
		case "desc":
			m.Desc = string(token.Data)
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
			m.Time, err = time.Parse(time.RFC3339, string(token.Data))
			if err != nil {
				return fmt.Errorf("time: %w", err)
			}
		}
	}
}

func (m *Metadata) UnmarshalXML(dec *xml.Decoder, se xml.StartElement) error {
	for {
		token, err := dec.Token()
		if err != nil {
			return fmt.Errorf("metadata: %w", err)
		}

		switch elem := token.(type) {
		case xml.StartElement:
			switch elem.Name.Local {
			case "author":
				m.Author = new(Author)
				if err := m.Author.UnmarshalXML(dec, elem); err != nil {
					return fmt.Errorf("author: %w", err)
				}
				continue
			case "link":
				m.Link = new(Link)
				if err := m.Link.UnmarshalXML(dec, elem); err != nil {
					return fmt.Errorf("link: %w", err)
				}
				continue
			}
			charData, err := getCharData(dec)
			if err != nil {
				return err
			}
			switch elem.Name.Local {
			case "name":
				m.Name = string(charData)
			case "desc":
				m.Desc = string(charData)
			case "time":
				m.Time, err = time.Parse(time.RFC3339, string(charData))
				if err != nil {
					return fmt.Errorf("time: %w", err)
				}
			}
		case xml.EndElement:
			if elem == se.End() {
				return nil
			}
		}
	}
}

// Author is Author schema (simplified).
type Author struct {
	Name string `xml:"name"`
	Link *Link  `xml:"link"`
}

func (a *Author) UnmarshalToken(tok *xmltokenizer.Tokenizer, se *xmltokenizer.Token) error {
	for {
		token, err := tok.Token()
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
			a.Name = string(token.Data)
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
}

func (a *Author) UnmarshalXML(dec *xml.Decoder, se xml.StartElement) error {
	for {
		token, err := dec.Token()
		if err != nil {
			return fmt.Errorf("author: %w", err)
		}

		switch elem := token.(type) {
		case xml.StartElement:
			switch elem.Name.Local {
			case "link":
				a.Link = new(Link)
				if err := a.Link.UnmarshalXML(dec, elem); err != nil {
					return fmt.Errorf("link: %w", err)
				}
			case "name":
				charData, err := getCharData(dec)
				if err != nil {
					return fmt.Errorf("name: %w", err)
				}
				a.Name = string(charData)
			}
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
			a.Text = string(token.Data)
		case "type":
			a.Type = string(token.Data)
		}
	}
}

func (a *Link) UnmarshalXML(dec *xml.Decoder, se xml.StartElement) error {
	for i := range se.Attr {
		attr := &se.Attr[i]
		switch attr.Name.Local {
		case "href":
			a.Href = attr.Value
		}
	}

	for {
		token, err := dec.Token()
		if err != nil {
			return fmt.Errorf("link: %w", err)
		}

		switch elem := token.(type) {
		case xml.StartElement:
			charData, err := getCharData(dec)
			if err != nil {
				return fmt.Errorf("%s: %w", elem.Name.Local, err)
			}
			switch elem.Name.Local {
			case "text":
				a.Text = string(charData)
			case "type":
				a.Type = string(charData)
			}
		case xml.EndElement:
			if elem == se.End() {
				return nil
			}
		}
	}
}
