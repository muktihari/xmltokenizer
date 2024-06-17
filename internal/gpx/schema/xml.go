package schema

import (
	"encoding/xml"
	"fmt"
)

func getCharData(dec *xml.Decoder) (xml.CharData, error) {
	token, err := dec.Token()
	if err != nil {
		return nil, err
	}
	v, ok := token.(xml.CharData)
	if !ok {
		return nil, fmt.Errorf("not a chardata")
	}
	return v, nil
}
