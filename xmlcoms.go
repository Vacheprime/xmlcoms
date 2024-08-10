package xmlcoms

import (
	"bufio"
	"encoding/xml"
	"errors"
	"io"
	"net"

	"github.com/Vacheprime/xmlcoms/stanza"
	"github.com/Vacheprime/xmlcoms/stream_elements"
)

const (
    DEFAULT_MAXBUFFERSIZE int64 = 10240
    DEFAULT_BUFIOBUFFERSIZE int = 1024
    DEFAULT_CHANNELBUFFERS int = 50
)

// Struct used for managing streams of xml elements
type XMLCommunicator struct {
    conn *net.TCPConn
    d *xml.Decoder
    e *xml.Encoder
    l *io.LimitedReader
    maxBufferSize int64
    isClosingStream bool
    StanzaChannel chan stanza.Stanza
    ErrorChannel chan error
}

// Reset the limited reader's remaining amount of bytes
func (c *XMLCommunicator) resetBufferLimit() {
    c.l.N = c.maxBufferSize
}

// Create a blank communicator
func NewXMLCommunicator() *XMLCommunicator {
    return &XMLCommunicator{conn: nil, d: nil, e: nil, l: nil, maxBufferSize: DEFAULT_MAXBUFFERSIZE, isClosingStream: false, StanzaChannel: nil, ErrorChannel: nil}
}

// Initialize an XMLCommunicator from an existing connection. Useful
// for initializing a communicator from a client connection (server side).
func NewCommunicatorFromConn(TCPConn *net.TCPConn) *XMLCommunicator {
    communicator := &XMLCommunicator{conn: TCPConn, maxBufferSize: DEFAULT_MAXBUFFERSIZE, isClosingStream: false}
    communicator.l = io.LimitReader(TCPConn, communicator.maxBufferSize).(*io.LimitedReader)
    communicator.d = xml.NewDecoder(bufio.NewReaderSize(communicator.l, DEFAULT_BUFIOBUFFERSIZE))
    communicator.e = xml.NewEncoder(TCPConn)

    // Initialize the channels
    errorChannel := make(chan error, DEFAULT_CHANNELBUFFERS)
    stanzaChannel := make(chan stanza.Stanza, DEFAULT_CHANNELBUFFERS)
    communicator.ErrorChannel = errorChannel
    communicator.StanzaChannel = stanzaChannel

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
    // Initialize the channels
    errorChannel := make(chan error, DEFAULT_CHANNELBUFFERS)
    stanzaChannel := make(chan stanza.Stanza, DEFAULT_CHANNELBUFFERS)
    c.ErrorChannel = errorChannel
    c.StanzaChannel = stanzaChannel

    // Initialize the xml decoder and encoder
    c.l = io.LimitReader(c.conn, c.maxBufferSize).(*io.LimitedReader)
    c.d = xml.NewDecoder(bufio.NewReaderSize(c.l, DEFAULT_BUFIOBUFFERSIZE)) 
    c.e = xml.NewEncoder(c.conn)
    
    return nil
} 

// Open an xml stream with the server
func (c *XMLCommunicator) OpenStream() error {
    // Send an opening element
    _, err := c.conn.Write([]byte(stream_elements.OpenStreamTag))
    if err != nil {
        return err
    }
    // Receive the incoming opening element
    opening, err := c.d.Token()
    if err != nil {
        return err
    }
    var name string = opening.(xml.StartElement).Name.Local

    if name != stream_elements.OpenStreamName {
        return errors.New("Invalid opening header!")
    } 
    go c.ReceiveStanzas()
    return nil 
}

// Accept an incoming stream request
func (c *XMLCommunicator) AcceptStreamOpen() error {
    // Receive an opening stream and accept it
    opening, err := c.d.Token()
    if err != nil {
        return err
    }
    var name string = opening.(xml.StartElement).Name.Local
    if name != stream_elements.OpenStreamName {
        return errors.New("Invalid opening header!")
    }
    // Send an opening element
    _, err = c.conn.Write([]byte(stream_elements.OpenStreamTag))
    if err != nil {
        return err
    }
    go c.ReceiveStanzas()
    return nil
}

// Request closure of the connection
func (c *XMLCommunicator) RequestCloseStream() error {
    // Close the connection at the end
    defer c.conn.Close()
    // Send a closing element
    _, err := c.conn.Write([]byte(stream_elements.CloseStreamTag))
    if err != nil {
        return err
    }

    c.isClosingStream = true
    return nil
}

// Accept closure of the connection
func (c *XMLCommunicator) AcceptCloseStream() error {
    // Close the connection at the end
    defer c.conn.Close()

    // Send a closing element
    _, err := c.conn.Write([]byte(stream_elements.CloseStreamTag))
    if err != nil {
        return err
    }
    return nil
}

// Take a BaseXML and decode it into its appropriate struct
func decodeBaseXML(xmlElement stanza.BaseXML) (stanza.Stanza, error) {

    var stanzaName string = xmlElement.XMLName.Local
    tokenDecoder := xml.NewTokenDecoder(&xmlElement)

    // Determine the type of stanza
    var stz stanza.Stanza 
    switch stanzaName {

    // Decode the base XML to the specific stanza
    case "message":
        var msg stanza.Message = stanza.Message(stanza.Message{})
        err := tokenDecoder.Decode(&msg)
        if err != nil {
            return nil, err
        }
        stz = stanza.Stanza(msg)

    default:
	return nil, errors.New("Unknown xml element.")
    }

    return stz, nil
}

// Process the next token received to handle cases such as the closure
// of the XML stream.
func (c *XMLCommunicator) processNextToken() (xml.Token, error) {
    // Attempt to obtain the next XML token
    token, err := c.d.Token()
    if err != nil {
	if errors.Is(err, net.ErrClosed) {
	    return nil, io.EOF
	}
        return nil, err
    }

    // Determine if the token is an end stream element
    switch token.(type) {
    case xml.EndElement:
        if token.(xml.EndElement).Name.Local == stream_elements.CloseStreamName {
	    // Determine whether the server is requesting stream
	    // closure or whether it is the client
	    if !c.isClosingStream {
		err := c.AcceptCloseStream()
		if err != nil {
		    return nil, err
		}
	    }
            return nil, io.EOF
        }
    }
    return token, nil

}

// Receive the next incoming stanza from the server
func (c *XMLCommunicator) ReceiveStanzas() { 
    for {
	// Process the next token
	token, err := c.processNextToken()
	if err != nil {
	    c.ErrorChannel <- err
	    break
	}
	
	// Initiate a new BaseXML struct
	startElement := token.(xml.StartElement)
	var xmlElement stanza.BaseXML = stanza.BaseXML{} 

	// Attempt to obtain the next XML element
	err = c.d.DecodeElement(&xmlElement, &startElement)
	if err != nil {
	    c.ErrorChannel <- err
	    break
	}

	// Decode the BaseXML into its appropriate stanza
	stz, err := decodeBaseXML(xmlElement)
	if err != nil {
	    c.ErrorChannel <- err
	    break
	}
	
	// Reset the limitedReader
	c.resetBufferLimit()
	c.StanzaChannel <- stz
    }
}

func (c *XMLCommunicator) SendStanza(msg stanza.Stanza) error {
    if c.isClosingStream {
	return errors.New("Cannot send stanza: the stream is being closed by client's request.")
    }
    err := c.e.Encode(msg)
    if err != nil {
        return err
    }
    return nil
}
