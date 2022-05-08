package p2p

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"

	msg "github.com/0xNathanW/bittorrent-go/p2p/message"
	"github.com/rivo/tview"
)

type Peer struct {
	PeerID   [20]byte
	IP       *net.TCPAddr
	Conn     *net.TCPConn
	BitField msg.Bitfield
	Start    time.Time

	Active  bool
	strikes int

	Rates *Rates

	Downloading bool // Should upload to best 4 peers.
	BlockOut    chan []byte

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

type Rates struct {
	Downloaded int
	Uploaded   int

	LastDownloaded int
	LastUploaded   int
}

func NewPeer(address *net.TCPAddr, bitfieldLength int) *Peer {

	p := &Peer{
		IP:       address,
		BitField: make(msg.Bitfield, bitfieldLength),

		Active:       false,
		Choked:       true,
		Interested:   false,
		IsChoking:    true,
		IsInterested: false,

		Rates: &Rates{},

		Activity: tview.NewTextView().
			SetScrollable(true).
			ScrollToEnd().
			SetDynamicColors(true).
			SetMaxLines(20),
	}

	p.Activity.
		SetBorder(true).
		SetTitle("Activity").
		SetTitleAlign(tview.AlignLeft).
		SetBorderPadding(1, 1, 2, 2)

	return p
}

// Serialised message is written to peer connection.
func (p *Peer) send(data []byte) error {
	p.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

	_, err := p.Conn.Write(data)
	if err != nil {
		return fmt.Errorf("failed to send msg: %w", err)
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

// Have messages are received semi-randomly.
// We read consecutive have msgs, stopping once we retrieve a non-have msg.
func (p *Peer) handleHave(m *msg.Message) {
	p.BitField.SetPiece(int(binary.BigEndian.Uint32(m.Payload[0:4])))
	for {
		m, err := p.read()
		if err != nil {
			return
		}
		if m.ID != 4 { // Not have.
			p.handle(m)
			return
		}
		p.BitField.SetPiece(int(binary.BigEndian.Uint32(m.Payload[0:4])))
	}
}

// Generic message handler.
func (p *Peer) handle(m *msg.Message) {
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

	// Connect to peer.
	conn, err := net.DialTimeout("tcp", p.IP.String(), 10*time.Second)
	if err != nil {
		return err
	}

	tcpConn := conn.(*net.TCPConn)
	if err := tcpConn.SetKeepAlive(true); err != nil {
		return err
	}
	p.Conn = tcpConn

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
	p.handle(message)

	p.Activity.Write([]byte("[green]peer established.[-]\n\n"))
	return nil
}

// buildBitfield parses the message and sets the peer's bitfield.
func (p *Peer) buildBitfield() error {

	message, err := p.read()
	if err != nil {
		return err
	}
	// Case of have or bitfield.
	if message.ID == 4 || message.ID == 5 {
		p.handle(message)

	} else if message.ID == 1 { // If peer is unchoking, try again.
		p.handle(message)
		if err := p.buildBitfield(); err != nil {
			return err
		}

	} else {
		p.handle(message)
		return fmt.Errorf("expected user piece info, got: %v", msg.MsgIDmap[message.ID])
	}
	return nil
}

func (p *Peer) disconnect() {
	p.Conn.Close()
	p.Active = false
	p.Activity.Write([]byte("[red]peer disconnected.[-]\n\n"))
}
