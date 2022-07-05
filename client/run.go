package client

import (
	"os"
	"sort"
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

	for _, peer := range c.Peers {
		go c.operatePeer(peer, workQ, dataQ, requestQ)
	}

	go func() {
		c.collectPieces(buf, dataQ)
		// Closing workQ causes peers to switch to seeding.
		close(workQ)
	}()

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
	c.Active.Lock()
	c.Active.int += 1
	c.Active.Unlock()

	p.Run(c.ID, c.Torrent, workQ, dataQ, requestQ)
	// When peer disconnects, it returns from Run().

	c.Active.Lock()
	c.Active.int -= 1
	c.Active.Unlock()
}

func (c *Client) collectPieces(buf []byte, dataQ <-chan *torrent.PieceData) {

	var done int            // Number of pieces downloaded.
	var bytesDownloaded int // Tracks number of bytes downloaded.

	speedTick := time.NewTicker(time.Second / 2)
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
			c.UI.App.QueueUpdateDraw(func() {
				c.UI.UpdateProgress(done)
				c.UI.UpdateTable()
			})

		case <-speedTick.C:
			// Update graph with new mb per second speed.
			mbps := float64(bytesDownloaded) / (1024 * 1024)
			c.UI.App.QueueUpdateDraw(func() { c.UI.Graph.Update(mbps) })
			bytesDownloaded = 0

		case <-sec10.C:
			// Check for all inactive.
			if c.Active.int == 0 {
				c.shutdown()
			}
			go c.chokingAlgo()
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

// Allows uploading to the top 4 peers that provide the most data.
func (c *Client) chokingAlgo() {

	top := make([]struct {
		peer string
		down int
	}, len(c.Peers))

	for _, peer := range c.Peers {

		per10down := peer.Rates.Downloaded - peer.Rates.LastDownloaded
		peer.Rates.LastDownloaded = peer.Rates.Downloaded

		top = append(top, struct {
			peer string
			down int
		}{peer: peer.IP.String(), down: per10down})
	}

	// Sort peers by download rate.
	sort.Slice(top, func(i, j int) bool {
		return top[i].down > top[j].down
	})

	for _, peer := range c.Peers {
		peer.Downloading = false

		for _, t := range top[:4] {
			if peer.IP.String() == t.peer && t.down > 0 {
				peer.Downloading = true
			}
		}
	}

	c.UI.App.QueueUpdateDraw(func() { c.UI.UpdateTable() })
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

func (c *Client) shutdown() {
	panic("No active peers, unable to continue...")
}
