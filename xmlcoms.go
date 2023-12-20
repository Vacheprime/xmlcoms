package xmlcoms

import (
    "bufio"
    "io"
    "net"
    "encoding/xml"
    "errors"
    
    "github.com/Vacheprime/xmlcoms/stanza"
)

// Struct used for managing streams of xml elements
type XMLCommunicator struct {
    conn *net.TCPConn
    d *xml.Decoder
    l *io.LimitedReader
}

// Create a blank communicator
func NewXMLCommunicator() *XMLCommunicator {
    return &XMLCommunicator{conn: nil, d: nil, l: nil}
}

// Initialize an XMLCommunicator from an existing connection. Useful
// for initializing a communicator from a client connection (server side).
func NewCommunicatorFromConn(TCPConn *net.TCPConn) *XMLCommunicator {
    communicator := &XMLCommunicator{conn: TCPConn}
    communicator.l = io.LimitReader(TCPConn, 1024 * 10).(*io.LimitedReader)
    communicator.d = xml.NewDecoder(bufio.NewReaderSize(communicator.l, 1024))
    return communicator
    
}

// Connect to the specified address 
func (c *XMLCommunicator) Connect(laddr, raddr, proto string) error {
    // Connect if not already connected
    if c.conn != nil {
        return errors.New("The communicator already possesses a connection!")
    }
    // Generate both local addr and remote addr
    local, err := net.ResolveTCPAddr(proto, laddr)
    if err != nil {
        return err
    }
    remote, err := net.ResolveTCPAddr(proto, raddr)
    if err != nil {
        return err
    }
    // Attempt to connect
    c.conn, err = net.DialTCP(proto, local, remote)
    if err != nil {
        return err
    }
    // Initialize the xml decoder
    c.l = io.LimitReader(c.conn, 1024 * 10).(*io.LimitedReader)
    c.d = xml.NewDecoder(bufio.NewReaderSize(c.l, 1024)) 

    return nil
} 

// Receive the next incoming stanza from the server
func (c *XMLCommunicator) ReceiveStanza() (stanza.Stanza, error) { 
    if c.conn == nil {
        return nil, errors.New("The communicator does not possess a connection!")
    }
    var xmlElement stanza.BaseXML = stanza.BaseXML{} 
    // Attempt to obtain the next XML element
    err := c.d.Decode(&xmlElement)
    if err != nil {
        return nil, err
    }
    var stanzaName string = xmlElement.XMLName.Local
    tokenDecoder := xml.NewTokenDecoder(&xmlElement)
    // Determine the type of stanza
    switch stanzaName {
    // Decode the base XML to the specific stanza
    case "message":
        var msg stanza.Message = stanza.Message{}  
        err := tokenDecoder.Decode(&msg)
        if err != nil {
            return nil, err
        }
        return msg, nil
    }
    // Reset the decoder and limitedReader
    c.l = io.LimitReader(c.conn, 1024 * 10).(*io.LimitedReader)
    c.d = xml.NewDecoder(bufio.NewReaderSize(c.l, 1024))
    return nil, nil
}
