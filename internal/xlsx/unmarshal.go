package xlsx

import (
	"encoding/xml"
	"io"

	"github.com/muktihari/xmltokenizer"
	"github.com/muktihari/xmltokenizer/internal/xlsx/schema"
)

func UnmarshalWithXMLTokenizer(r io.Reader) (schema.SheetData, error) {
	tok := xmltokenizer.New(r)
	var sheetData schema.SheetData
loop:
	for {
		token, err := tok.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return sheetData, err
		}

		switch string(token.Name.Local) {
		case "sheetData":
			se := xmltokenizer.GetToken().Copy(token)
			err = sheetData.UnmarshalToken(tok, se)
			xmltokenizer.PutToken(se)
			if err != nil {
				return sheetData, err
			}
			break loop
		}
	}

	return sheetData, nil
}

func UnmarshalWithStdlibXML(r io.Reader) (schema.SheetData, error) {
	dec := xml.NewDecoder(r)
	var sheetData schema.SheetData
	for {
		token, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return sheetData, err
		}

		switch elem := token.(type) {
		case xml.StartElement:
			if elem.Name.Local == "sheetData" {
				if err = dec.DecodeElement(&sheetData, &elem); err != nil {
					return sheetData, err
				}
			}
		}
	}
	return sheetData, nil
}
