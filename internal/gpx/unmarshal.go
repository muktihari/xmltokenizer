package gpx

import (
	"encoding/xml"
	"io"
	"os"

	"github.com/muktihari/xmltokenizer"
	"github.com/muktihari/xmltokenizer/internal/gpx/schema"
)

func UnmarshalWithXMLTokenizer(path string) (schema.GPX, error) {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	tok := xmltokenizer.New(f)
	var gpx schema.GPX
loop:
	for {
		token, err := tok.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return gpx, err
		}

		switch string(token.Name.Local) {
		case "gpx":
			se := xmltokenizer.GetToken().Copy(token)
			err = gpx.UnmarshalToken(tok, se)
			xmltokenizer.PutToken(se)
			if err != nil {
				return gpx, err
			}
			break loop
		}
	}

	return gpx, nil
}

func UnmarshalWithStdlibXML(path string) (schema.GPX, error) {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	dec := xml.NewDecoder(f)
	var gpx schema.GPX
loop:
	for {
		token, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return gpx, err
		}

		se, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		switch se.Name.Local {
		case "gpx":
			if err = gpx.UnmarshalXML(dec, se); err != nil {
				return gpx, err
			}
			break loop
		}
	}

	return gpx, nil
}
