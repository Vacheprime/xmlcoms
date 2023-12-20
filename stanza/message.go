package stanza

import (
    "encoding/xml"
)

// A message stanza
type Message struct {
    XMLName xml.Name `xml:"message"`
    To string `xml:"to"`
    From string `xml:"from"`
    Body string `xml:"body"`
}
