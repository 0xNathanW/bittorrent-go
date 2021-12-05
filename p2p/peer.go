package p2p

import (
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
	Reader        io.Reader
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
	fmt.Println("Port:", p.Port)
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
	fmt.Println("\nSending msg: ", msg.MsgIDmap[data[4]])
	_, err := p.Conn.Write(data)
	if err != nil {
		return fmt.Errorf("failed to send data: %v", err)
	}
	return nil
}

// Reads single message from peer connection.
func (p *Peer) Read() (*msg.Message, error) {
	p.Conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	msg := new(msg.Message)
	buf := make([]byte, 4)
	_, err := io.ReadFull(p.Conn, buf)
	if err != nil {
		return nil, err
	}
	msg.Length = buf
	length := binary.BigEndian.Uint32(msg.Length)
	message := make([]byte, length)
	_, err = io.ReadFull(p.Conn, message)
	if err != nil {
		return nil, fmt.Errorf("failed to read message: %v", err)
	}
	msg.ID = message[0]
	if msg.ID > 7 {
		return nil, fmt.Errorf("unknown message ID: %v", msg.ID)
	}
	if length > 1 {
		msg.Payload = message[1:]
	}
	return msg, nil
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
		return fmt.Errorf("error receiving handshake: %v", err)
	}
	// Check if handshake is valid, if so return the peer's ID.
	peerID, err := msg.VerifyHandshake(buf, infoHash)
	if err != nil {
		return err
	}
	p.PeerID = peerID
	return nil
}

func (p *Peer) BuildBitfield() error {
	message, err := p.Read()
	if err != nil {
		return err
	}
	switch msg.MsgIDmap[message.ID] {
	case "Bitfield":
		if len(message.Payload) != len(p.BitField) {
			return fmt.Errorf("invalid bitfield length")
		}
		p.BitField = message.Payload
	case "Have":
		p.BitField.SetPiece(int(binary.BigEndian.Uint32(message.Payload[0:4])))
		for msg.MsgIDmap[message.ID] == "Have" {
			message, err = p.Read()
			if err != nil {
				return err
			}
			p.BitField.SetPiece(int(binary.BigEndian.Uint32(message.Payload[0:4])))
		}
	default:
		return fmt.Errorf("unexpected message: %v", msg.MsgIDmap[message.ID])
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
	fmt.Println("Downloading piece: ", idx)
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
			if err != nil {
				return nil, fmt.Errorf("failed to read message: %v", err)
			}
			if msg.ID == 7 {
				fmt.Println("Received block")
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
				copy(data[downloaded:], msgData)
				downloaded += len(msgData)
			} else {
				return nil, fmt.Errorf("unexpected message: %v", msg.ID)
			}
		}
	}
	return data, nil
}
