package xlsx

import (
	"encoding/xml"
	"io"
	"os"

	"github.com/muktihari/xmltokenizer"
	"github.com/muktihari/xmltokenizer/internal/xlsx/schema"
)

func UnmarshalWithXMLTokenizer(path string) (schema.SheetData, error) {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	tok := xmltokenizer.New(f)
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

func UnmarshalWithStdlibXML(path string) (schema.SheetData, error) {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	dec := xml.NewDecoder(f)
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
