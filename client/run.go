package client

import (
	"os"
	"time"

	"github.com/0xNathanW/bittorrent-go/p2p"
	msg "github.com/0xNathanW/bittorrent-go/p2p/message"
	"github.com/0xNathanW/bittorrent-go/torrent"
)

func (c *Client) Run() {

	workQ := c.Torrent.NewWorkQueue()      // workQ is the queue of pieces we need to download.
	dataQ := make(chan *torrent.PieceData) // dataQ recieves piece data from workers.
	requestQ := make(chan p2p.Request)     // requestQ is the queue of requests we need to send to peers.
	defer close(requestQ)
	buf := make([]byte, c.Torrent.Size)

	for _, peer := range c.Peers.active {
		c.operatePeer(peer, workQ, dataQ, requestQ)
	}

	go c.collectPieces(buf, dataQ)
	go c.serveRequests(buf, requestQ)

	// Run tview event loop.
	if err := c.UI.App.SetFocus(c.UI.PeerTable).Run(); err != nil {
		panic(err)
	}
}

func (c *Client) operatePeer(
	p *p2p.Peer,
	workQ chan torrent.Piece,
	dataQ chan<- *torrent.PieceData,
	requestQ chan<- p2p.Request,
) {

	p.Run(c.ID, c.Torrent, workQ, dataQ, requestQ)
	// When peer disconnects, it returns from Run().
	c.Peers.Unlock()

	delete(c.Peers.active, p.IP.String())
	c.Peers.inactive = append(c.Peers.inactive, p.IP)

	c.Peers.Lock()

	// if len(c.Peers.active) == 0 {
	// 	// Some sort of shutdown procedure.
	// }
}

func (c *Client) collectPieces(buf []byte, dataQ <-chan *torrent.PieceData) {

	var done int            // Number of pieces downloaded.
	var bytesDownloaded int // Tracks number of bytes downloaded.

	sec := time.NewTicker(time.Second)
	sec10 := time.NewTicker(time.Second * 10)

	// Collect downloaded pieces.
	for done < len(c.Torrent.Pieces) {

		select {
		// Piece data received and written to buffer.
		case piece := <-dataQ:

			start, end, err := c.Torrent.PiecePosition(piece.Index)
			if err != nil {
				panic(err) //fix
			}

			n := copy(buf[start:end], piece.Data)
			c.BitField.SetPiece(piece.Index)

			bytesDownloaded += n
			done++
			c.UI.App.QueueUpdateDraw(func() { c.UI.UpdateProgress(done) })

		case <-sec.C:

			mbps := float64(bytesDownloaded) / (1024 * 1024)
			_ = mbps

		case <-sec10.C:
			go c.chokingAlgo()
		}
	}

	c.seed()
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

// Allows uploading to the top 4 peers that provide the most data.
func (c *Client) chokingAlgo() {

	// last := make(map[*p2p.Peer][2]int)
	// ticker := time.NewTicker(time.Second * 10)

	// for range ticker.C {

	// 	for _, peer := range c.Peers {
	// 		if peer.Active {
	// 			peer.DownloadRate = peer.Downloaded - last[peer][0]
	// 			peer.UploadRate = peer.Uploaded - last[peer][1]
	// 			last[peer] = [2]int{peer.Downloaded, peer.Uploaded}
	// 		} else {
	// 			peer.DownloadRate = 0
	// 			peer.UploadRate = 0
	// 		}
	// 	}

	// 	// uploadSort := c.Peers
	// 	// sort.Slice(uploadSort, func(i, j int) bool {
	// 	// 	return uploadSort[i].DownloadRate > uploadSort[j].DownloadRate
	// 	// })

	// 	// for i := 0; i < 4; i++ {
	// 	// 	for _, peer := range c.Peers {

	// 	// 		if peer.IP.String() == uploadSort[i].IP.String() {
	// 	// 			peer.Downloading = true
	// 	// 			peer.Activity.Write([]byte("[green]serving requests from peer.\n\n[-]"))
	// 	// 		} else {
	// 	// 			peer.Downloading = false
	// 	// 		}
	// 	// 	}
	// 	// }
	// }
}

func (c *Client) serveRequests(buf []byte, requestQ <-chan p2p.Request) {
	for request := range requestQ {

		if !c.BitField.HasPiece(request.Idx) {
			// Send bitfield?
			continue
		}

		start, _, err := c.Torrent.PiecePosition(request.Idx)
		if err != nil {
			continue
		}

		// Retrieve piece from buffer.
		var block []byte
		_ = copy(block, buf[start+request.Offset:start+request.Offset+request.Length])

		request.Peer.BlockOut <- msg.Block(request.Idx, request.Offset, block)
	}
}

func (c *Client) seed() {

}
