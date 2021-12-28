package p2p

import (
	"encoding/binary"
	"errors"
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
	Conn     net.Conn
	BitField msg.Bitfield

	Active       bool
	Choked       bool
	Interested   bool
	IsChoking    bool
	IsInterested bool
	// UI elements.
	Page     *tview.Flex
	Info     *tview.TextView
	Activity *tview.TextView
}

// String sent by tracker is parsed into peer structs.
func ParsePeers(peerString string, bfLength int) []*Peer {
	var peers []*Peer
	// Each peer is a string of length 6.
	numPeers := len(peerString) / 6
	for i := 0; i < numPeers; i++ {
		// First 4 bytes are IP address.
		ip := peerString[i*6 : i*6+4]
		// Next 2 bytes are port.
		port := []byte(peerString[i*6+4 : i*6+6])
		peer := &Peer{
			IP:           net.IP{ip[0], ip[1], ip[2], ip[3]},
			Port:         strconv.Itoa(int(binary.BigEndian.Uint16(port))),
			BitField:     make(msg.Bitfield, bfLength),
			Active:       false,
			Choked:       true,
			Interested:   false,
			IsChoking:    true,
			IsInterested: false,
			Page:         tview.NewFlex().SetDirection(tview.FlexRow),
			Info: tview.NewTextView().
				SetDynamicColors(true).
				SetScrollable(false),
			Activity: tview.NewTextView().
				SetScrollable(true).
				ScrollToEnd().
				SetDynamicColors(true),
		}
		peer.UpdateInfo()
		peer.Activity.SetBorder(true).SetTitle("Activity").SetTitleAlign(tview.AlignLeft).SetBorderPadding(1, 1, 2, 2)
		peer.Page.AddItem(peer.Info, 0, 1, false).AddItem(peer.Activity, 0, 3, false)
		peer.Page.SetBorder(true).SetTitle("Peer Info").SetTitleAlign(tview.AlignLeft)
		peers = append(peers, peer)
	}
	return peers
}

// Initalises peer connection.
func (p *Peer) Connect() error {
	// Connect to IP on TCP network.
	addr := net.JoinHostPort(p.IP.String(), p.Port)
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed connection: %v", err)
	}
	p.Conn = conn
	p.Activity.Write([]byte("[green]Successfully connected to peer.[-]\n\n"))
	return nil
}

// Serialised message is written to peer connection.
func (p *Peer) Send(data []byte) error {
	p.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	_, err := p.Conn.Write(data)
	if err != nil {
		return fmt.Errorf("failed to send data: %v", err)
	}
	// Update activity, ID is fourth idx.
	if data[4] != 6 {
		p.Activity.Write([]byte(fmt.Sprintf("==> %s\n\n", msg.MsgIDmap[data[4]])))
	}
	return nil
}

// Reads single message from peer connection.
func (p *Peer) Read() (*msg.Message, error) {
	p.Conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	message := new(msg.Message)
	buf := make([]byte, 4) // Length buffer.
	_, err := io.ReadFull(p.Conn, buf)
	if err != nil {
		return nil, err
	}
	message.Length = buf
	length := binary.BigEndian.Uint32(message.Length)
	if length == 0 {
		p.Activity.Write([]byte("<== Keep-Alive\n\n"))
		return nil, errors.New("keep-alive")
	}
	messageBuf := make([]byte, length)
	_, err = io.ReadFull(p.Conn, messageBuf)
	if err != nil {
		return nil, fmt.Errorf("failed to read message: %v", err)
	}
	message.ID = messageBuf[0]
	if message.ID > 7 {
		return nil, fmt.Errorf("unknown message ID: %v", message.ID)
	}
	if length > 1 {
		message.Payload = messageBuf[1:]
	}
	// Update activity.
	if message.ID != 7 {
		p.Activity.Write([]byte(fmt.Sprintf("<== %s\n\n", msg.MsgIDmap[message.ID])))
	}
	return message, nil
}

func (p *Peer) exchangeHandshake(ID, infoHash [20]byte) error {
	// Send handshake message.
	p.Conn.SetDeadline(time.Now().Add(15 * time.Second))
	_, err := p.Conn.Write(msg.Handshake(ID, infoHash))
	if err != nil {
		return fmt.Errorf("failed to send handshake: %w", err)
	}
	// Receive handshake message.
	buf := make([]byte, 68)
	_, err = p.Conn.Read(buf)
	if err != nil {
		return fmt.Errorf("error receiving handshake: %w", err)
	}
	// Check if handshake is valid, if so return the peer's ID.
	peerID, err := msg.VerifyHandshake(buf, infoHash)
	if err != nil {
		return err
	}
	p.Activity.Write([]byte("[green]Handshake successful.[-]\n\n"))
	p.PeerID = peerID
	return nil
}

// Establish peer ensures a verified connection to a peer
// and that we have information about what pieces the peer has.
func (p *Peer) EstablishPeer(ID, infoHash [20]byte) error {
	// Connect to peer.
	err := p.Connect()
	if err != nil {
		return err
	}
	err = p.exchangeHandshake(ID, infoHash)
	if err != nil {
		return err
	}
	// Peers will then send messages about what pieces they have.
	// This can come in many forms, eg bitfield or have msgs.
	// This is where we will parse the message and set the peer's bitfield.
	err = p.buildBitfield()
	if err != nil {
		return err
	}
	// Send intent to download from peer.
	p.Send(msg.Unchoke())
	p.Send(msg.Interested())
	// Wait for response from peer.
	message, err := p.Read()
	if err != nil {
		return err
	}
	if message.ID == 1 {
		p.IsChoking = false
	}
	return nil
}

// buildBitfield parses the message and sets the peer's bitfield.
func (p *Peer) buildBitfield() error {
	message, err := p.Read()
	if err != nil {
		return err
	}
	switch msg.MsgIDmap[message.ID] {
	case "Bitfield":
		if len(message.Payload) != len(p.BitField) { // Wrong sized bitfield.
			return fmt.Errorf("invalid bitfield length")
		}
		p.BitField = message.Payload
	case "Have": // If "have" need to keep reading until full bitfield sent.
		p.BitField.SetPiece(int(binary.BigEndian.Uint32(message.Payload[0:4])))
		for msg.MsgIDmap[message.ID] == "Have" {
			message, err = p.Read()
			if err != nil {
				return err
			}
			p.BitField.SetPiece(int(binary.BigEndian.Uint32(message.Payload[0:4])))
		}
	default:
		return fmt.Errorf("unexpected message: %s", msg.MsgIDmap[message.ID])
	}
	return nil
}

func (p *Peer) DownloadPiece(idx, length int) ([]byte, error) {
	p.Conn.SetDeadline(time.Now().Add(30 * time.Second))
	/* Pieces are too long to request in one go.
	 * We will request a piece in chunks of 16384 bytes (16Kb) called blocks.
	 * The last block will likely be smaller.
	 */
	// requested and downloaded keep track of progress.
	p.Activity.Write([]byte(fmt.Sprintf("Downloading piece %d.\n\n", idx)))
	requested := 0
	downloaded := 0
	data := make([]byte, length)
	for downloaded < length {
		if p.IsChoking {
			return nil, fmt.Errorf("peer is choking")
		}
		// Request all blocks in piece.
		for requested < length {
			var blockSize int = 16384 // 16Kb
			// If last block is smaller, set block size to remaining bytes.
			if requested+blockSize > length {
				blockSize = length - requested
			}
			// Request block.
			err := p.Send(msg.Request(idx, requested, blockSize))
			if err != nil {
				return nil, fmt.Errorf("failed to send request: %v", err)
			}
			requested += blockSize
		}

		// Read responses.
		for downloaded < length {
			msg, err := p.Read()
			if errors.Is(err, errors.New("keep-alive")) {
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("failed to read response: %v", err)
			}
			if msg.ID == 7 {
				msgIdx := int(binary.BigEndian.Uint32(msg.Payload[0:4]))
				msgBegin := int(binary.BigEndian.Uint32(msg.Payload[4:8]))
				msgData := msg.Payload[8:]
				// Check piece is the correct index.
				if msgIdx != idx {
					return nil, fmt.Errorf(
						"piece index mismatch, expected: %d, got: %d",
						idx, msgIdx,
					)
				}
				// Check begin is less than length of data.
				if msgBegin >= length {
					return nil, fmt.Errorf(
						"piece begin index too large, expected: %d, got: %d",
						length, msgBegin,
					)
				}
				// Check if begin plus length is greater than length of data.
				if msgBegin+len(msgData) > length {
					return nil, fmt.Errorf(
						"piece length too large, expected: %d, got: %d",
						length, msgBegin+len(msgData),
					)
				}
				// Copy data to data buffer.
				n := copy(data[downloaded:], msgData)
				downloaded += n
			} else {
				return nil, fmt.Errorf("unexpected message: %v", msg.ID)
			}
		}
	}
	return data, nil
}

func (p *Peer) UpdateInfo() {
	p.Info.SetText(
		fmt.Sprintf("\tID: %s\n\n"+
			"\tIP: %s\n\n"+
			"\tPort: %s\n\n"+
			"\tActive: %t\n\n"+
			"\tIs Choking: %t\n\n"+
			"\tIs Interested: %t\n",
			p.PeerID, p.IP.String(), p.Port,
			p.Active, p.IsChoking, p.IsInterested),
	)
}
