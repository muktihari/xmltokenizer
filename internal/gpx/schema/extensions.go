package schema

import (
	"encoding/xml"
	"fmt"
	"math"
	"strconv"

	"github.com/muktihari/xmltokenizer"
)

// TrackpointExtension is a GPX extension for health-related data.
type TrackpointExtension struct {
	Cadence     uint8
	Distance    float64
	HeartRate   uint8
	Temperature int8
	Power       uint16
}

func (t *TrackpointExtension) reset() {
	t.Cadence = math.MaxUint8
	t.Distance = math.NaN()
	t.HeartRate = math.MaxUint8
	t.Temperature = math.MaxInt8
	t.Power = math.MaxUint16
}

func (t *TrackpointExtension) UnmarshalToken(tok *xmltokenizer.Tokenizer, se *xmltokenizer.Token) error {
	t.reset()

	for {
		token, err := tok.Token()
		if err != nil {
			return fmt.Errorf("trackpointExtension: %w", err)
		}

		if token.IsEndElementOf(se) {
			return nil
		}
		if token.IsEndElement {
			continue
		}

		switch string(token.Name.Local) {
		case "cad", "cadence":
			val, err := strconv.ParseUint(string(token.Data), 10, 8)
			if err != nil {
				return err
			}
			t.Cadence = uint8(val)
		case "distance":
			val, err := strconv.ParseFloat(string(token.Data), 64)
			if err != nil {
				return err
			}
			t.Distance = val
		case "hr", "heartrate":
			val, err := strconv.ParseUint(string(token.Data), 10, 8)
			if err != nil {
				return err
			}
			t.HeartRate = uint8(val)
		case "atemp", "temp", "temperature":
			val, err := strconv.ParseInt(string(token.Data), 10, 8)
			if err != nil {
				return err
			}
			t.Temperature = int8(val)
		case "power":
			val, err := strconv.ParseUint(string(token.Data), 10, 16)
			if err != nil {
				return err
			}
			t.Power = uint16(val)
		}
	}
}

func (t *TrackpointExtension) UnmarshalXML(dec *xml.Decoder, se xml.StartElement) error {
	t.reset()

	for {
		token, err := dec.Token()
		if err != nil {
			return fmt.Errorf("trackpointExtension: %w", err)
		}

		switch elem := token.(type) {
		case xml.StartElement:
			charData, err := getCharData(dec)
			if err != nil {
				return err
			}
			switch elem.Name.Local {
			case "cad", "cadence":
				val, err := strconv.ParseUint(string(charData), 10, 8)
				if err != nil {
					return err
				}
				t.Cadence = uint8(val)
			case "distance":
				val, err := strconv.ParseFloat(string(charData), 64)
				if err != nil {
					return err
				}
				t.Distance = val
			case "hr", "heartrate":
				val, err := strconv.ParseUint(string(charData), 10, 8)
				if err != nil {
					return err
				}
				t.HeartRate = uint8(val)
			case "atemp", "temp", "temperature":
				val, err := strconv.ParseInt(string(charData), 10, 8)
				if err != nil {
					return err
				}
				t.Temperature = int8(val)
			case "power":
				val, err := strconv.ParseUint(string(charData), 10, 16)
				if err != nil {
					return err
				}
				t.Power = uint16(val)
			}
		case xml.EndElement:
			if elem == se.End() {
				return nil
			}
		}
	}
}
