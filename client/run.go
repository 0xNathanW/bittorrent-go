package client

import (
	"crypto/sha1"
	"fmt"
	"os"
	"time"

	"github.com/0xNathanW/bittorrent-go/p2p"
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

	// Start workers, each in a goroutine.
	for _, peer := range c.Peers {
		go c.operatePeer(peer, workQ, dataQ)
	}

	// Collect downloaded pieces.
	go c.collectPieces(dataQ)

	// Run tview event loop.
	if err := c.UI.App.SetFocus(c.UI.PeerList).Run(); err != nil {
		panic(err)
	}
}

// operatePeer is a goroutine that handles communication with a single peer.
// If an error occurs, the peer is disconnected and we return from function.
func (c *Client) operatePeer(peer *p2p.Peer, workQ chan Piece, dataQ chan<- *PieceData) {
	// Establish connection with peer.
	err := peer.EstablishPeer(c.ID, c.Torrent.InfoHash)
	if err != nil {
		peer.Activity.Write([]byte("[red]" + err.Error() + "[-]\n\n"))
		return
	}
	defer peer.Conn.Close()

	// Begin downloading pieces.
	for piece := range workQ {
		peer.UpdateInfo()
		// If peer doesnt have piece, put it back in the queue.
		if !peer.BitField.HasPiece(piece.Index) {
			workQ <- piece
			continue
		}
		// Attempt to download piece.
		data, err := peer.DownloadPiece(piece.Index, piece.Length)
		if err != nil {
			workQ <- piece
			peer.Activity.Write([]byte("[red]" + err.Error() + "[-]\n\n"))
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
			peer.Activity.Write([]byte("[red]Invalid piece hash.[-]\n\n"))
			continue
		}
		// Send piece to dataQ.
		dataQ <- &PieceData{piece.Index, data}
		peer.Activity.Write([]byte(fmt.Sprintf("[blue]Downloaded piece %d.[-]\n\n", piece.Index)))
	}
}

func (c *Client) collectPieces(dataQ <-chan *PieceData) {

	buf := make([]byte, c.Torrent.Size) // Output buffer.
	var done int                        // Tracks number of pieces downloaded.
	var mbDownloaded int                // Tracks number of megabytes downloaded.
	sec := time.NewTicker(time.Second)
	var mbps float64 // Megabytes per second.

	// Collect downloaded pieces.
	for done < len(c.Torrent.Pieces) {
		select {
		// When a piece is pulled from the data queue,
		// It is written to the output buffer.
		case piece := <-dataQ:
			start, end, err := c.Torrent.PiecePosition(piece.Index)
			if err != nil {
				panic(err) //fix
			}
			n := copy(buf[start:end], piece.Data)
			mbDownloaded += n
			// Add megabytes to mbps.
			mbps += float64(n) / 1024 / 1024
			done++
		// Every second, UI graph and progress bar is updated.
		case <-sec.C:
			// Queue UI update and draw.
			c.UI.App.QueueUpdateDraw(
				func() {
					c.UI.Graph.Update(mbps)
					c.UI.UpdateProgress(done, len(c.Torrent.Pieces))
				},
			)
			mbps = 0 // Reset mbps.
		}
	}
	// Write output buffer to file.
	c.writeToFile(buf)
}

func (c *Client) writeToFile(buf []byte) error {
	// If torrent is single file.
	if len(c.Torrent.Files) == 0 {
		f, err := os.Create(c.Torrent.Name)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.Write(buf)
		if err != nil {
			return err
		}
		return nil
	} else {
		// If torrent is multi file.
		start := 0
		for _, file := range c.Torrent.Files {
			f, err := os.Create(file.Path)
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = f.Write(buf[start : start+file.Length])
			if err != nil {
				return err
			}
			start += file.Length
		}
	}
	return nil
}
