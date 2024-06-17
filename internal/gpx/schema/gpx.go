package schema

import (
	"encoding/xml"
	"fmt"

	"github.com/muktihari/xmltokenizer"
)

// GPX is GPX schema (simplified).
type GPX struct {
	Creator  string   `xml:"creator,attr"`
	Version  string   `xml:"version,attr"`
	Metadata Metadata `xml:"metadata,omitempty"`
	Tracks   []Track  `xml:"trk,omitempty"`
}

func (g *GPX) UnmarshalToken(tok *xmltokenizer.Tokenizer, se *xmltokenizer.Token) error {
	for i := range se.Attrs {
		attr := &se.Attrs[i]
		switch string(attr.Name.Local) {
		case "creator":
			g.Creator = string(attr.Value)
		case "version":
			g.Version = string(attr.Value)
		}
	}

	for {
		token, err := tok.Token()
		if err != nil {
			return fmt.Errorf("gpx: %w", err)
		}

		if token.IsEndElementOf(se) {
			return nil
		}
		if token.IsEndElement() {
			continue
		}

		switch string(token.Name.Local) {
		case "metadata":
			se := xmltokenizer.GetToken().Copy(token)
			err = g.Metadata.UnmarshalToken(tok, se)
			xmltokenizer.PutToken(se)
			if err != nil {
				return fmt.Errorf("metadata: %w", err)
			}
		case "trk":
			var track Track
			se := xmltokenizer.GetToken().Copy(token)
			err = track.UnmarshalToken(tok, se)
			xmltokenizer.PutToken(se)
			if err != nil {
				return fmt.Errorf("track: %w", err)
			}
			g.Tracks = append(g.Tracks, track)
		}
	}
}

func (g *GPX) UnmarshalXML(dec *xml.Decoder, se xml.StartElement) error {
	for i := range se.Attr {
		attr := &se.Attr[i]
		switch attr.Name.Local {
		case "creator":
			g.Creator = attr.Value
		case "version":
			g.Version = attr.Value
		}
	}

	for {
		token, err := dec.Token()
		if err != nil {
			return fmt.Errorf("gpx: %w", err)
		}

		switch elem := token.(type) {
		case xml.StartElement:
			switch elem.Name.Local {
			case "metadata":
				if err := g.Metadata.UnmarshalXML(dec, elem); err != nil {
					return fmt.Errorf("metadata: %w", err)
				}
			case "trk":
				var track Track
				if err := track.UnmarshalXML(dec, elem); err != nil {
					return fmt.Errorf("track: %w", err)
				}
				g.Tracks = append(g.Tracks, track)
			}

		case xml.EndElement:
			if elem == se.End() {
				return nil
			}
		}
	}
}
