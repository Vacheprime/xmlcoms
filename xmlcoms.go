package xmlcoms

import (
    "bufio"
    "io"
    "net"
    "encoding/xml"
    "errors"
    
    "github.com/Vacheprime/xmlcoms/stanza"
    "github.com/Vacheprime/xmlcoms/stream_elements"
)

// Struct used for managing streams of xml elements
type XMLCommunicator struct {
    conn *net.TCPConn
    d *xml.Decoder
    e *xml.Encoder
    l *io.LimitedReader
}

//func resetDecoder

// Create a blank communicator
func NewXMLCommunicator() *XMLCommunicator {
    return &XMLCommunicator{conn: nil, d: nil, e: nil, l: nil}
}

// Initialize an XMLCommunicator from an existing connection. Useful
// for initializing a communicator from a client connection (server side).
func NewCommunicatorFromConn(TCPConn *net.TCPConn) *XMLCommunicator {
    communicator := &XMLCommunicator{conn: TCPConn}
    communicator.l = io.LimitReader(TCPConn, 1024 * 10).(*io.LimitedReader)
    communicator.d = xml.NewDecoder(bufio.NewReaderSize(communicator.l, 1024))
    communicator.e = xml.NewEncoder(TCPConn)
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
    // Initialize the xml decoder and encoder
    c.l = io.LimitReader(c.conn, 1024 * 10).(*io.LimitedReader)
    c.d = xml.NewDecoder(bufio.NewReaderSize(c.l, 1024)) 
    c.e = xml.NewEncoder(c.conn)
    
    // Attempt to negotiate an xml stream
    err = c.NegotiateStream()
    if err != nil {
        return err
    }
    return nil
} 

// Negotiate an xml stream
func (c *XMLCommunicator) NegotiateStream() error {
    // Send an opening element
    _, err := c.conn.Write([]byte(stream_elements.OpenStream))
    if err != nil {
        return err
    }
    // Receive the incoming opening element
    opening, err := c.d.Token()
    if err != nil {
        return err
    }
    var name string = opening.(*xml.StartElement).Name.Local
    if name != stream_elements.OpenStream {
        return errors.New("Invalid opening header!")
    } 
    return nil 
}

// Close the connection
func (c *XMLCommunicator) Close() error {
    if c.conn != nil {
        err := c.conn.Close()
        if err != nil {
            return err
        } else {
            return nil
        }
    } else {
        return errors.New("The communicator does not possess a connection!")
    }
}

// Receive the next incoming stanza from the server
func (c *XMLCommunicator) ReceiveStanza() (stanza.Stanza, error) { 
    // Only attempt to receive if that is possible
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
    var msg stanza.Stanza 
    switch stanzaName {
    // Decode the base XML to the specific stanza
    case "message":
        msg = stanza.Message{}
        err := tokenDecoder.Decode(&msg)
        if err != nil {
            return nil, err
        }
    }
    // Reset the decoder and limitedReader
    //c.l = io.LimitReader(c.conn, 1024 * 10).(*io.LimitedReader)
    c.l.N = 1024 * 10
    c.d = xml.NewDecoder(bufio.NewReaderSize(c.l, 1024))
    return msg, nil
}

func (c *XMLCommunicator) SendStanza(msg stanza.Stanza) error {
    c.e.Encode(msg)
    return nil
}
