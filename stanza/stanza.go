package stanza 

import (
    "encoding/xml"
    "io"

)

// A general interface to reprensent any type of stanza
type Stanza any 

// A general XML message for decoding
type BaseXML struct {
    XMLName xml.Name
    Attrs []xml.Attr
    Tks []xml.Token
    i int 
}

// Custom unmarshaling of the xml element
func (b *BaseXML) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
    // Get the name of the received xml element
    b.XMLName = start.Name
    // Copy all attributes
    b.Attrs = start.Copy().Attr
    // Loop through all tokens and save them
    startCopy := start.Copy()
    b.Tks = append(b.Tks, startCopy)
    // Set the token index to 0
    b.i = 0
    for {
        tk, err := d.Token() 
        if err != nil {
            if err == io.EOF {
                break
            } else {
                return err
            }
        }
        // Save a copy to the tokens
        tkCopy := xml.CopyToken(tk)
        b.Tks = append(b.Tks, tkCopy) 
    }

    return nil
}

// Obtain tokens
func (b *BaseXML) Token() (xml.Token, error) {
    // Check if the index has reached the last token
    if b.i < len(b.Tks) {
        tk := b.Tks[b.i]
        b.i++
        return tk, nil
    } else {
        return nil, io.EOF
    }
}
