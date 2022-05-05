package p2p

import (
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"time"

	msg "github.com/0xNathanW/bittorrent-go/p2p/message"
	"github.com/0xNathanW/bittorrent-go/torrent"
)

func (p *Peer) Run(
	ID [20]byte,
	t *torrent.Torrent,
	workQ chan torrent.Piece,
	dataQ chan<- *torrent.PieceData,
	requestQ chan<- Request,
) {

	if err := p.establishPeer(ID, t.InfoHash); err != nil {
		p.Activity.Write([]byte(fmt.Sprintf("[red]failed to establish connection: %v.[-]\n\n", err)))
		p.Conn.Close()
		return
	}
	p.Active = true

	defer p.disconnect()

	for {
		select {

		case piece := <-workQ:

			if p.Downloading && p.Choked { // If a top peer, unchoke them.
				p.send(msg.Unchoke())
			}

			// If peer doesnt have piece, put it back in the queue.
			if !p.BitField.HasPiece(piece.Index) {
				workQ <- piece
				continue
			}

			if err := p.downloadPiece(piece, dataQ, requestQ); err != nil {
				workQ <- piece
				p.Activity.Write([]byte("[red]" + err.Error() + "[-]\n\n"))

				p.strikes++        // Add a strike if download fails.
				if p.strikes > 5 { // 5 strikes and peer gets disconnected.
					p.Activity.Write([]byte("[red]too many strikes, disconnecting...[-]\n\n"))
					return
				}

				continue
			}
		}
	}
}

func (p *Peer) downloadPiece(piece torrent.Piece, dataQ chan<- *torrent.PieceData, requestQ chan<- Request) error {

	p.Conn.SetDeadline(time.Now().Add(30 * time.Second))

	/* Pieces are too long to request in one go.
	 * We will request a piece in chunks of 16384 bytes (16Kb) called blocks.
	 * The last block will likely be smaller.
	 */

	p.Activity.Write([]byte(fmt.Sprintf("downloading piece %d.\n\n", piece.Index)))

	requested, downloaded := 0, 0
	data := make([]byte, piece.Length)

	// Request all blocks in piece.
	for requested < piece.Length {

		var blockSize int = 16384 // 16Kb
		// If last block is smaller, set block size to remaining bytes.
		if requested+blockSize > piece.Length {
			blockSize = piece.Length - requested
		}

		// Request block.
		if err := p.send(msg.Request(piece.Index, requested, blockSize)); err != nil {
			return fmt.Errorf("failed to send request: %v", err)
		}
		requested += blockSize
	}

	// read responses.
	for downloaded < piece.Length {

		// Read next message.
		m, err := p.read()
		if err != nil {
			return fmt.Errorf("failed to read from connection: %v", err)
		}

		if m.ID == 6 {
			// If the peer is allowed, add to the request queue.
			if p.Downloading {
				idx := int(binary.BigEndian.Uint32(m.Payload[0:4]))
				off := int(binary.BigEndian.Uint32(m.Payload[4:8]))
				length := int(binary.BigEndian.Uint32(m.Payload[8:12]))
				requestQ <- Request{p, idx, off, length}
				continue
			} else { // If not allowed, choke.
				p.send(msg.Choke())
				continue
			}
		}

		if m.ID == 7 && m.Payload != nil {

			msgIdx := int(binary.BigEndian.Uint32(m.Payload[0:4]))
			msgBegin := int(binary.BigEndian.Uint32(m.Payload[4:8]))
			msgData := m.Payload[8:]

			// Check piece is the correct index.
			if msgIdx != piece.Index {
				return fmt.Errorf(
					"piece index mismatch, expected: %d, got: %d",
					piece.Index, msgIdx,
				)
			}
			// Check begin is less than length of data.
			if msgBegin >= piece.Length {
				return fmt.Errorf(
					"piece begin index too large, expected: %d, got: %d",
					piece.Length, msgBegin,
				)
			}
			// Check if begin plus length is greater than length of data.
			if msgBegin+len(msgData) > piece.Length {
				return fmt.Errorf(
					"piece length too large, expected: %d, got: %d",
					piece.Length, msgBegin+len(msgData),
				)
			}

			// Copy data to data buffer.
			n := copy(data[downloaded:], msgData)
			downloaded += n

		} else {
			p.handleOther(m)
		}
	}

	// verify piece hash.
	h := sha1.New()
	h.Write(data)
	hash := h.Sum(nil)

	var pieceHash [20]byte
	copy(pieceHash[:], hash)
	if pieceHash != piece.Hash {
		return fmt.Errorf("piece hash mismatch")
	}

	// send piece to dataQ.
	dataQ <- &torrent.PieceData{Index: piece.Index, Data: data}
	p.Activity.Write([]byte(fmt.Sprintf("[blue]downloaded piece %d.[-]\n\n", piece.Index)))
	p.Downloaded += piece.Length

	return nil
}
