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

	Active  bool
	strikes int

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
func ParsePeers(peerString string, bfLength int) (map[[20]byte]*Peer, []*net.TCPAddr) {

	// Each peer is a string of length 6.
	numPeers := len(peerString) / 6
	peers := make(map[[20]byte]*Peer)
	var inactive []*net.TCPAddr

	for i := 0; i < numPeers; i++ {

		address, err := net.ResolveTCPAddr("tcp", peerString[i*6:(i+1)*6])
		if err != nil {
			print("failed to resolve address %s:", peerString[i*6:(i+1)*6], err)
			continue
		}

		peer, err := NewPeer(address, bfLength)
		if err != nil {
			peers[peer.PeerID] = peer
		} else {
			inactive = append(inactive, address)
		}
	}
	return peers, inactive
}

func NewPeer(address *net.TCPAddr, bitfieldLength int) (*Peer, error) {

	conn, err := net.DialTCP("tcp", nil, address)
	if err != nil {
		return nil, fmt.Errorf("could not connect to peer: %v", err)
	}

	if err := conn.SetKeepAlive(true); err != nil {
		return nil, fmt.Errorf("could not set keep alive: %v", err)
	}

	p := &Peer{
		IP:       address,
		Conn:     conn,
		BitField: make(msg.Bitfield, bitfieldLength),

		Active:       false,
		Choked:       true,
		Interested:   false,
		IsChoking:    true,
		IsInterested: false,

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

	p.Activity.Write([]byte("[green]Connected to peer[-]\n\n"))

	return p, nil
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

func (p *Peer) disconnect() {
	p.Conn.Close()
	p.Active = false
	p.Activity.Write([]byte("[red]peer disconnected.[-]\n\n"))
}
