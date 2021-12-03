package p2p

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	msg "github.com/0xNathanW/bittorrent-goV2/p2p/message"
)

type Peer struct {
	PeerID        [20]byte
	IP            net.IP
	Port          string
	Conn          net.Conn
	BitField      msg.Bitfield
	Choked        bool
	Interested    bool
	IsChoking     bool
	IsInteresting bool
	Strikes       int
}

func ParsePeers(peerString string, bfLength int) []*Peer {
	var peers []*Peer
	numPeers := len(peerString) / 6
	for i := 0; i < numPeers; i++ {
		ip := peerString[i*6 : i*6+4]
		port := []byte(peerString[i*6+4 : i*6+6])
		peer := &Peer{
			IP:            net.IP{ip[0], ip[1], ip[2], ip[3]},
			Port:          strconv.Itoa(int(binary.BigEndian.Uint16(port))),
			BitField:      make(msg.Bitfield, bfLength),
			Choked:        true,
			Interested:    false,
			IsChoking:     true,
			IsInteresting: false,
			Strikes:       0,
		}
		peers = append(peers, peer)
	}
	return peers
}

func (p *Peer) PrintInfo() {
	fmt.Println("PeerID:", p.PeerID)
	fmt.Println("IP:", p.IP.String())
	fmt.Println("Port:", []byte(p.Port))
	fmt.Println("Choked:", p.Choked)
	fmt.Println("Interested:", p.Interested)
	fmt.Println("IsChoking:", p.IsChoking)
	fmt.Println("IsInteresting:", p.IsInteresting)
	fmt.Println("Strikes:", p.Strikes)
}

// Initalises peer connection.
func (p *Peer) Connect() error {
	// Connect to IP on TCP network.
	addr := net.JoinHostPort(p.IP.String(), p.Port)
	fmt.Println("Connecting to", addr)
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to peer: %v", err)
	}
	p.Conn = conn
	return nil
}

// Serialised message is written to peer connection.
func (p *Peer) Send(data []byte) error {
	p.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	_, err := p.Conn.Write(data)
	if err != nil {
		return fmt.Errorf("failed to send data: %v", err)
	}
	return nil
}

func (p *Peer) Read() []msg.Message {
	p.Conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	rawMessages := make([][]byte, 0)
	for {
		// Message buffer starts with 4 bytes for length.
		message := bytes.NewBuffer(make([]byte, 4))
		// Length is read from connection.
		// If EOF is reached, break.
		_, err := p.Conn.Read(message.Bytes())
		if err == io.EOF {
			break
		}
		// Length is converted to int.
		length := int(binary.BigEndian.Uint32(message.Bytes()))
		// Grow message buffer to fit message.
		message.Grow(length)
		// Message is read from connection.
		// If EOF is reached, break.
		_, err = p.Conn.Read(message.Bytes())
		if err == io.EOF {
			break
		}
		// Message is appended to raw message buffer.
		rawMessages = append(rawMessages, message.Bytes())
	}
	return msg.ParseMsgs(rawMessages)
}

func (p *Peer) ExchangeHandshake(ID, infoHash [20]byte) error {
	// Send handshake message.
	p.Conn.SetDeadline(time.Now().Add(15 * time.Second))
	_, err := p.Conn.Write(msg.Handshake(ID, infoHash))
	if err != nil {
		return fmt.Errorf("failed to send handshake: %v", err)
	}
	// Receive handshake message.
	buf := make([]byte, 68)
	_, err = p.Conn.Read(buf)
	if err != nil {
		return fmt.Errorf("failed to receive handshake: %v", err)
	}
	// Check if handshake is valid, if so return the peer's ID.
	peerID, err := msg.VerifyHandshake(buf, infoHash)
	if err != nil {
		return err
	}
	p.PeerID = peerID
	return nil
}
