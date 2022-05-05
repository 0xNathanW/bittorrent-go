package p2p

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	msg "github.com/0xNathanW/bittorrent-go/p2p/message"
	"github.com/rivo/tview"
)

type Peer struct {
	PeerID   [20]byte
	IP       net.IP
	Port     string
	Conn     net.TCPConn
	BitField msg.Bitfield
	Active   bool
	strikes  int
	// Buffer for messages outside peer goroutine
	MsgBuffer chan []byte

	DownloadRate int
	UploadRate   int

	Downloaded  int
	Uploaded    int
	Downloading bool // Should upload to best 4 peers.

	Choked       bool
	Interested   bool
	IsChoking    bool
	IsInterested bool
	// UI elements.
	Activity *tview.TextView
}

type Request struct {
	Peer   *Peer
	Idx    int
	Offset int
	Length int
}

// String sent by tracker is parsed into peer structs.
func ParsePeers(peerString string, bfLength int) []*Peer {

	var peers []*Peer
	// Each peer is a string of length 6.
	numPeers := len(peerString) / 6

	for i := 0; i < numPeers; i++ {

		ip := peerString[i*6 : i*6+4]             // First 4 bytes are IP address.
		port := []byte(peerString[i*6+4 : i*6+6]) // Next 2 bytes are port.

		peer := &Peer{
			IP:        net.IP{ip[0], ip[1], ip[2], ip[3]},
			Port:      strconv.Itoa(int(binary.BigEndian.Uint16(port))),
			BitField:  make(msg.Bitfield, bfLength),
			MsgBuffer: make(chan []byte, 5),

			Active:       false,
			Choked:       true,
			Interested:   false,
			IsChoking:    true,
			IsInterested: false,

			DownloadRate: 0,
			UploadRate:   0,

			Activity: tview.NewTextView().
				SetScrollable(true).
				ScrollToEnd().
				SetDynamicColors(true).
				SetMaxLines(20),
		}

		peer.Activity.
			SetBorder(true).
			SetTitle("Activity").
			SetTitleAlign(tview.AlignLeft).
			SetBorderPadding(1, 1, 2, 2)

		peers = append(peers, peer)
	}
	return peers
}

// Initalises peer connection.
func (p *Peer) connect() error {
	// Connect to IP on TCP.
	address := net.JoinHostPort(p.IP.String(), p.Port)
	conn, err := net.DialTimeout("tcp", address, 15*time.Second)
	if err != nil {
		return fmt.Errorf("failed connection: %v", err)
	}

	p.Conn = conn
	p.Activity.Write([]byte("[green]TCP connection successful.[-]\n\n"))
	return nil
}

// Serialised message is written to peer connection.
func (p *Peer) send(data []byte) error {
	p.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

	_, err := p.Conn.Write(data)
	if err != nil {
		return fmt.Errorf("failed to send data: %w", err)
	}
	// Update activity, blocks will clog feed.
	if data[4] != 6 {
		p.Activity.Write([]byte(fmt.Sprintf("==> %s\n\n", msg.MsgIDmap[data[4]])))
	}
	return nil
}

// Reads single message from peer connection.
func (p *Peer) read() (*msg.Message, error) {

	p.Conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	message := new(msg.Message)

	buf := make([]byte, 4) // Length buffer.
	if _, err := io.ReadFull(p.Conn, buf); err != nil {
		return nil, err
	}
	message.Length = buf

	length := binary.BigEndian.Uint32(message.Length)
	if length == 0 { // Keep-alive message.
		p.Activity.Write([]byte("<== keep-Alive\n\n"))
		return nil, nil
	}

	messageBuf := make([]byte, length)
	if _, err := io.ReadFull(p.Conn, messageBuf); err != nil {
		return nil, fmt.Errorf("failed to read message: %v", err)
	}

	message.ID = messageBuf[0]
	if message.ID > 7 {
		return nil, fmt.Errorf("unknown message ID: %v", message.ID)
	}
	if length > 1 {
		message.Payload = messageBuf[1:]
	}
	if message.ID != 7 { // Update activity, as long as not block, as they will clog feed.
		p.Activity.Write([]byte(fmt.Sprintf("<== %s\n\n", msg.MsgIDmap[message.ID])))
	}

	return message, nil
}

func (p *Peer) handleHave(m *msg.Message) {

	p.BitField.SetPiece(int(binary.BigEndian.Uint32(m.Payload[0:4])))

	for {
		m, err := p.read()
		if err != nil {
			return
		}

		if m.ID != 4 {
			p.handleOther(m)
			return
		}

		p.BitField.SetPiece(int(binary.BigEndian.Uint32(m.Payload[0:4])))
	}
}

func (p *Peer) handleOther(m *msg.Message) {
	switch m.ID {
	case 0: // Choke
		p.IsChoking = true

	case 1: // Unchoke
		p.IsChoking = false

	case 2: // Interested
		if p.Downloading {
			p.IsInterested = true
			p.send(msg.Unchoke())
		} else {
			p.send(msg.Choke())
		}

	case 3: // Not interested
		p.IsInterested = false

	case 4: // Have
		p.handleHave(m)

	case 5: // Bitfield
		p.BitField = msg.Bitfield(m.Payload)

	default:
		return
	}
}

func (p *Peer) exchangeHandshake(ID, infoHash [20]byte) error {

	p.Conn.SetDeadline(time.Now().Add(20 * time.Second))

	// send handshake message.
	_, err := p.Conn.Write(msg.Handshake(ID, infoHash))
	if err != nil {
		return fmt.Errorf("failed to send handshake: %w", err)
	}

	// Receive handshake message.
	buf := make([]byte, 68)
	if _, err = p.Conn.Read(buf); err != nil {
		return fmt.Errorf("error receiving handshake: %w", err)
	}

	// Check if handshake is valid, if so return the peer's ID.
	peerID, err := msg.VerifyHandshake(buf, infoHash)
	if err != nil {
		return err
	}

	p.Activity.Write([]byte("[green]handshake successful.[-]\n\n"))
	p.PeerID = peerID
	return nil
}

// Establish peer ensures a verified connection to a peer
// and that we have information about what pieces the peer has.
func (p *Peer) establishPeer(ID, infoHash [20]byte) error {
	// Connect to peer and exchange handshake.
	if err := p.connect(); err != nil {
		return err
	}

	if err := p.exchangeHandshake(ID, infoHash); err != nil {
		return err
	}
	// Peers will then send messages about what pieces they have.
	// This can come in many forms, eg bitfield or have msgs.
	// This is where we will parse the message and set the peer's bitfield.
	if err := p.buildBitfield(); err != nil {
		return err
	}
	// send intent to download from peer.
	p.send(msg.Interested())

	// TODO: Maybe add condition for if unchoke already received.
	// Wait for unchoke from peer.
	message, err := p.read()
	if err != nil {
		return err
	}
	// Sometimes peers will annoyingly send have messages after bitfields.
	// However we should be expecting an unchoke message.
	p.handleOther(message)

	p.Activity.Write([]byte("[green]peer established.[-]\n\n"))
	return nil
}

// buildBitfield parses the message and sets the peer's bitfield.
func (p *Peer) buildBitfield() error {

	message, err := p.read()
	if err != nil {
		return err
	}

	if message.ID == 4 || message.ID == 5 {
		p.handleOther(message)

	} else if message.ID == 1 {
		p.handleOther(message)

		if err := p.buildBitfield(); err != nil {
			return err
		}

	} else {
		p.handleOther(message)
		return fmt.Errorf("expected user piece info, got: %v", msg.MsgIDmap[message.ID])
	}
	return nil
}

// Attempts to reconnect to peer 3 times at 30 second intervals.
func (p *Peer) attemptReconnect(ID, infoHash [20]byte) error {

	time.Sleep(time.Second * 30)
	for i := 0; i < 2; i++ {
		if err := p.establishPeer(ID, infoHash); err == nil {
			return nil
		}
		p.Activity.Write([]byte(fmt.Sprintf("[red]reconnection attempt %v failed[-]\n\n", i+1)))
		time.Sleep(time.Second * 30)
	}

	return fmt.Errorf("failed reconnection 3 times, disconnecting")
}
