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

func ConstructMessage(To, From, Body string) Message {
    var messageXMLName xml.Name = xml.Name{Space: "", Local: "message"}
    var message Message = Message{XMLName: messageXMLName, To: To, Body: Body} 
    return message
}

