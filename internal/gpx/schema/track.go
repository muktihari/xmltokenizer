package schema

import (
	"encoding/xml"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/muktihari/xmltokenizer"
)

type Track struct {
	Name          string         `xml:"name,omitempty"`
	Type          string         `xml:"type,omitempty"`
	TrackSegments []TrackSegment `xml:"trkseg,omitempty"`
}

func (t *Track) UnmarshalToken(tok *xmltokenizer.Tokenizer, se *xmltokenizer.Token) error {
	for {
		token, err := tok.Token()
		if err != nil {
			return fmt.Errorf("track: %w", err)
		}

		if token.IsEndElementOf(se) {
			return nil
		}
		if token.IsEndElement() {
			continue
		}

		switch string(token.Name.Local) {
		case "name":
			t.Name = string(token.Data)
		case "type":
			t.Type = string(token.Data)
		case "trkseg":
			var trkseg TrackSegment
			se := xmltokenizer.GetToken().Copy(token)
			err = trkseg.UnmarshalToken(tok, se)
			xmltokenizer.PutToken(se)
			if err != nil {
				return fmt.Errorf("trkseg: %w", err)
			}
			t.TrackSegments = append(t.TrackSegments, trkseg)
		}
	}
}

func (t *Track) UnmarshalXML(dec *xml.Decoder, se xml.StartElement) error {
	for {
		token, err := dec.Token()
		if err != nil {
			return fmt.Errorf("track: %w", err)
		}

		switch elem := token.(type) {
		case xml.StartElement:
			switch elem.Name.Local {
			case "trkseg":
				var trkseg TrackSegment
				if err := trkseg.UnmarshalXML(dec, elem); err != nil {
					return fmt.Errorf("trkseg: %w", err)
				}
				t.TrackSegments = append(t.TrackSegments, trkseg)
				continue
			}
			charData, err := getCharData(dec)
			if err != nil {
				return fmt.Errorf("%s: %w", elem.Name.Local, err)
			}
			switch elem.Name.Local {
			case "name":
				t.Name = string(charData)
			case "type":
				t.Type = string(charData)
			}
		case xml.EndElement:
			if elem == se.End() {
				return nil
			}
		}
	}
}

type TrackSegment struct {
	Trackpoints []Waypoint `xml:"trkpt,omitempty"`
}

func (t *TrackSegment) UnmarshalToken(tok *xmltokenizer.Tokenizer, se *xmltokenizer.Token) error {
	for {
		token, err := tok.Token()
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
		case "trkpt":
			var trkpt Waypoint
			se := xmltokenizer.GetToken().Copy(token)
			err = trkpt.UnmarshalToken(tok, se)
			xmltokenizer.PutToken(se)
			if err != nil {
				return fmt.Errorf("trkpt: %w", err)
			}
			t.Trackpoints = append(t.Trackpoints, trkpt)
		}
	}
}

func (t *TrackSegment) UnmarshalXML(dec *xml.Decoder, se xml.StartElement) error {
	for {
		token, err := dec.Token()
		if err != nil {
			return err
		}

		switch elem := token.(type) {
		case xml.StartElement:
			switch elem.Name.Local {
			case "trkpt":
				var trkpt Waypoint
				if err := trkpt.UnmarshalXML(dec, elem); err != nil {
					return fmt.Errorf("trkpt: %w", err)
				}
				t.Trackpoints = append(t.Trackpoints, trkpt)
			}
		case xml.EndElement:
			if elem == se.End() {
				return nil
			}
		}
	}
}

type Waypoint struct {
	Lat                 float64             `xml:"lat,attr,omitempty"`
	Lon                 float64             `xml:"lon,attr,omitempty"`
	Ele                 float64             `xml:"ele,omitempty"`
	Time                time.Time           `xml:"time,omitempty"`
	TrackpointExtension TrackpointExtension `xml:"extensions>TrackPointExtension,omitempty"`
}

func (w *Waypoint) reset() {
	w.Lat = math.NaN()
	w.Lon = math.NaN()
	w.Ele = math.NaN()
	w.Time = time.Time{}
	w.TrackpointExtension.reset()
}

func (w *Waypoint) UnmarshalToken(tok *xmltokenizer.Tokenizer, se *xmltokenizer.Token) error {
	w.reset()

	var err error
	for i := range se.Attrs {
		attr := &se.Attrs[i]
		switch string(attr.Name.Local) {
		case "lat":
			w.Lat, err = strconv.ParseFloat(string(attr.Value), 64)
			if err != nil {
				return fmt.Errorf("lat: %w", err)
			}
		case "lon":
			w.Lon, err = strconv.ParseFloat(string(attr.Value), 64)
			if err != nil {
				return fmt.Errorf("lon: %w", err)
			}
		}
	}

	for {
		token, err := tok.Token()
		if err != nil {
			return fmt.Errorf("waypoint: %w", err)
		}

		if token.IsEndElementOf(se) {
			return nil
		}
		if token.IsEndElement() {
			continue
		}

		switch string(token.Name.Local) {
		case "ele":
			w.Ele, err = strconv.ParseFloat(string(token.Data), 64)
			if err != nil {
				return fmt.Errorf("ele: %w", err)
			}
		case "time":
			w.Time, err = time.Parse(time.RFC3339, string(token.Data))
			if err != nil {
				return fmt.Errorf("time: %w", err)
			}
		case "extensions":
			se := xmltokenizer.GetToken().Copy(token)
			err = w.TrackpointExtension.UnmarshalToken(tok, se)
			xmltokenizer.PutToken(se)
			if err != nil {
				return fmt.Errorf("extensions: %w", err)
			}
		}
	}
}

func (w *Waypoint) UnmarshalXML(dec *xml.Decoder, se xml.StartElement) error {
	w.reset()

	var err error
	for i := range se.Attr {
		attr := &se.Attr[i]
		switch attr.Name.Local {
		case "lat":
			w.Lat, err = strconv.ParseFloat(attr.Value, 64)
			if err != nil {
				return fmt.Errorf("lat: %w", err)
			}
		case "lon":
			w.Lon, err = strconv.ParseFloat(attr.Value, 64)
			if err != nil {
				return fmt.Errorf("lon: %w", err)
			}
		}
	}

	for {
		token, err := dec.Token()
		if err != nil {
			return fmt.Errorf("waypoint: %w", err)
		}

		switch elem := token.(type) {
		case xml.StartElement:
			switch elem.Name.Local {
			case "extensions":
				if err := w.TrackpointExtension.UnmarshalXML(dec, elem); err != nil {
					return fmt.Errorf("extensions: %w", err)
				}
				continue
			}
			charData, err := getCharData(dec)
			if err != nil {
				return fmt.Errorf("%s: %w", elem.Name.Local, err)
			}
			switch elem.Name.Local {
			case "ele":
				w.Ele, err = strconv.ParseFloat(string(charData), 64)
				if err != nil {
					return fmt.Errorf("ele:  %w", err)
				}
			case "time":
				w.Time, err = time.Parse(time.RFC3339, string(charData))
				if err != nil {
					return fmt.Errorf("time:  %w", err)
				}
			}
		case xml.EndElement:
			if elem == se.End() {
				return nil
			}
		}
	}
}
