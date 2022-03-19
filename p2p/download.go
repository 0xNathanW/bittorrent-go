package p2p

import (
	"context"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"time"

	msg "github.com/0xNathanW/bittorrent-go/p2p/message"
	"github.com/0xNathanW/bittorrent-go/torrent"
)

func (p *Peer) Run(
	ctx context.Context,
	ID [20]byte,
	t *torrent.Torrent,
	workQ chan torrent.Piece,
	dataQ chan<- *torrent.PieceData,
	requestQ chan<- [3]int,
) {

	if err := p.establishPeer(ID, t.InfoHash); err != nil {

		p.Activity.Write([]byte("[red]" + err.Error() + "[-]\n\n"))
		p.Activity.Write([]byte("[red]attempting to reconnect...[-]\n\n"))

		if err := p.attemptReconnect(ID, t.InfoHash); err != nil {
			p.Activity.Write([]byte("[red]" + err.Error() + "[-]\n\n"))
			return
		}
	}
	p.Active = true

	defer func() {
		p.Conn.Close()
		p.Active = false
		p.Activity.Write([]byte("[red]disconnected from peer[-]\n\n"))
	}()

	// Go-routine measures download/upload speed.
	measureCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go p.measureSpeeds(measureCtx)

	for {
		select {

		case <-ctx.Done():
			return

		case piece := <-workQ:

			if p.strikes > 5 {
				p.Activity.Write([]byte("[red]too many strikes, disconnecting...[-]\n\n"))
				return
			}

			// If peer doesnt have piece, put it back in the queue.
			if !p.BitField.HasPiece(piece.Index) {
				workQ <- piece
				continue
			}

			if err := p.downloadPiece(piece, dataQ, requestQ); err != nil {
				workQ <- piece
				p.Activity.Write([]byte("[red]" + err.Error() + "[-]\n\n"))
				p.strikes++
				continue
			}
		}
	}
}

func (p *Peer) downloadPiece(
	piece torrent.Piece,
	dataQ chan<- *torrent.PieceData,
	requestQ chan<- [3]int,
) error {

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

		m, err := p.read()
		if err != nil {
			return fmt.Errorf("failed to read response: %v", err)
		}

		// Add requests to queue.
		if m.ID == 6 {
			idx := int(binary.BigEndian.Uint32(m.Payload[0:4]))
			off := int(binary.BigEndian.Uint32(m.Payload[4:8]))
			length := int(binary.BigEndian.Uint32(m.Payload[8:12]))
			requestQ <- [3]int{idx, off, length}
			continue
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
			p.Downloaded += n
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
		p.Downloaded -= piece.Length
		return fmt.Errorf("piece hash mismatch")
	}

	// send piece to dataQ.
	dataQ <- &torrent.PieceData{Index: piece.Index, Data: data}
	p.Activity.Write([]byte(fmt.Sprintf("[blue]downloaded piece %d.[-]\n\n", piece.Index)))

	return nil
}

func (p *Peer) measureSpeeds(ctx context.Context) {

	ticker := time.NewTicker(time.Second)
	var lastDownloaded, lastUploaded int

	for {
		select {

		case <-ticker.C:
			p.DownloadRate = float64(p.Downloaded-lastDownloaded) / float64(1024)
			p.UploadRate = float64(p.Uploaded-lastUploaded) / float64(1024)

			lastDownloaded = p.Downloaded
			lastUploaded = p.Uploaded

		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}
