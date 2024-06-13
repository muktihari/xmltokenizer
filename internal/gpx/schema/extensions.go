package schema

import (
	"encoding/xml"
	"io"
	"math"
	"strconv"

	"github.com/muktihari/xmltokenizer"
)

// TrackPointExtension is a GPX extension for health-related data.
type TrackPointExtension struct {
	Cadence     uint8
	Distance    float64
	HeartRate   uint8
	Temperature int8
	Power       uint16
}

func (t *TrackPointExtension) reset() {
	t.Cadence = math.MaxUint8
	t.Distance = math.NaN()
	t.HeartRate = math.MaxUint8
	t.Temperature = math.MaxInt8
	t.Power = math.MaxUint16
}

func (t *TrackPointExtension) UnmarshalToken(tok *xmltokenizer.Tokenizer, se *xmltokenizer.Token) error {
	t.reset()

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
		case "cad", "cadence":
			val, err := strconv.ParseUint(string(token.CharData), 10, 8)
			if err != nil {
				return err
			}
			t.Cadence = uint8(val)
		case "distance":
			val, err := strconv.ParseFloat(string(token.CharData), 64)
			if err != nil {
				return err
			}
			t.Distance = val
		case "hr", "heartrate":
			val, err := strconv.ParseUint(string(token.CharData), 10, 8)
			if err != nil {
				return err
			}
			t.HeartRate = uint8(val)
		case "atemp", "temp", "temperature":
			val, err := strconv.ParseInt(string(token.CharData), 10, 8)
			if err != nil {
				return err
			}
			t.Temperature = int8(val)
		case "power":
			val, err := strconv.ParseUint(string(token.CharData), 10, 16)
			if err != nil {
				return err
			}
			t.Power = uint16(val)
		}
	}

	return nil
}

func (t *TrackPointExtension) UnmarshalXML(dec *xml.Decoder, se xml.StartElement) error {
	t.reset()

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
			case "cad", "cadence":
				val, err := strconv.ParseUint(string(elem), 10, 8)
				if err != nil {
					return err
				}
				t.Cadence = uint8(val)
			case "distance":
				val, err := strconv.ParseFloat(string(elem), 64)
				if err != nil {
					return err
				}
				t.Distance = val
			case "hr", "heartrate":
				val, err := strconv.ParseUint(string(elem), 10, 8)
				if err != nil {
					return err
				}
				t.HeartRate = uint8(val)
			case "atemp", "temp", "temperature":
				val, err := strconv.ParseInt(string(elem), 10, 8)
				if err != nil {
					return err
				}
				t.Temperature = int8(val)
			case "power":
				val, err := strconv.ParseUint(string(elem), 10, 16)
				if err != nil {
					return err
				}
				t.Power = uint16(val)
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
