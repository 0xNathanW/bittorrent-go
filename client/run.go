package client

import (
	"crypto/sha1"
	"fmt"
	"log"
	"time"

	"github.com/0xNathanW/bittorrent-goV2/p2p"
	msg "github.com/0xNathanW/bittorrent-goV2/p2p/message"
)

type Piece struct {
	Index  int
	Length int
	Hash   [20]byte
}

type PieceData struct {
	Index int
	Data  []byte
}

func (c *Client) Run() {

	// workQ is the queue of pieces we need to download.
	// If a worker is available, it will be given a piece from the queue.
	// If a worker fails to download a piece, it will be put back on the queue.
	workQ := make(chan Piece, len(c.Torrent.Pieces))
	defer close(workQ)
	// Fill queue with all pieces.
	for idx, hash := range c.Torrent.Pieces {
		workQ <- Piece{idx, c.Torrent.PieceSize(idx), hash}
	}
	// dataQ is a buffer of downloaded pieces.
	dataQ := make(chan *PieceData)

	// Start workers.
	for _, peer := range c.Peers {
		go c.operatePeer(peer, workQ, dataQ)
	}

	go c.collectPieces(dataQ)
	log.Print("Downloading...")
	if err := c.UI.App.SetRoot(c.UI.Layout, true).Run(); err != nil {
		fmt.Println(err)
		panic(err)
	}
	time.Sleep(time.Second * 100)
}

// operatePeer is a goroutine that handles communication with a single peer.
// If an error occurs, the peer is disconnected and we return from function.
func (c *Client) operatePeer(peer *p2p.Peer, workQ chan Piece, dataQ chan<- *PieceData) {

	// Establish connection with peer.
	err := peer.EstablishPeer(c.ID, c.Torrent.InfoHash)
	if err != nil {
		return
	}
	defer peer.Conn.Close()

	// Send intent to download from peer.
	peer.Send(msg.Unchoke())
	peer.Send(msg.Interested())

	// Wait for response from peer.
	message, err := peer.Read()
	if err != nil {
		return
	}
	if message.ID == 1 {
		peer.IsChoking = false
	}

	// Begin downloading pieces.
	for piece := range workQ {

		// If peer doesnt have piece, put it back in the queue.
		if !peer.BitField.HasPiece(piece.Index) {
			workQ <- piece
			continue
		}
		// Attempt to download piece.
		data, err := peer.DownloadPiece(piece.Index, piece.Length)
		if err != nil {
			workQ <- piece
			return
		}
		// Verify integrity of piece.
		h := sha1.New()
		h.Write(data)
		hashSlice := h.Sum(nil)
		// Convert to byte array.
		var hash [20]byte
		copy(hash[:], hashSlice)
		if hash != piece.Hash {
			workQ <- piece
			continue
		}
		// Send piece to dataQ.
		dataQ <- &PieceData{piece.Index, data}
	}
}

func (c *Client) collectPieces(dataQ <-chan *PieceData) {
	// Output buffer.
	buf := make([]byte, c.Torrent.Size)

	var done int             // Tracks number of pieces downloaded.
	var mbDownloaded float64 // Tracks number of bytes downloaded.
	sec := time.NewTicker(time.Second / 60)
	startTime := time.Now()
	var mbps float64

	// Collect downloaded pieces.
	for done < len(c.Torrent.Pieces) {
		select {
		case piece := <-dataQ:
			start, end, err := c.Torrent.PiecePosition(piece.Index)
			if err != nil {
				panic(err) //fix
			}
			n := copy(buf[start:end], piece.Data)
			mbDownloaded += float64(n) / 1024 / 1024
			done++
		case <-sec.C:
			// calculate seconds elapsed
			elapsed := time.Since(startTime)
			mbps = float64(mbDownloaded) / float64(elapsed.Seconds())
			c.UI.Graph.Update(mbps)

		}
	}

}
