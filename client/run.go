package client

import (
	"context"
	"os"
	"time"

	"github.com/0xNathanW/bittorrent-go/torrent"
)

func (c *Client) Run() {
	// workQ is the queue of pieces we need to download.
	// If a worker is available, it will be given a piece from the queue.
	// If a worker fails to download a piece, it will be put back on the queue.
	workQ := c.Torrent.NewWorkQueue()
	defer close(workQ)

	// dataQ is a buffer of downloaded pieces.
	dataQ := make(chan *torrent.PieceData)
	defer close(dataQ)

	// requestQ is a buffer of requests for pieces.
	// requests consist of the idx, begin, and length of the piece.
	requestQ := make(chan [3]int)
	defer close(requestQ)

	ctx, cancel := context.WithCancel(context.Background())
	_ = cancel

	// Start workers, each in a goroutine.
	for _, peer := range c.Peers {
		go peer.Run(ctx, c.ID, c.Torrent, workQ, dataQ, requestQ)
	}

	// Collect downloaded pieces.
	go c.collectPieces(dataQ)

	// Run tview event loop.
	if err := c.UI.App.SetFocus(c.UI.PeerTable).Run(); err != nil {
		panic(err)
	}
}

func (c *Client) collectPieces(dataQ <-chan *torrent.PieceData) {

	buf := make([]byte, c.Torrent.Size) // Output buffer.

	var done int            // Tracks number of pieces downloaded.
	var bytesDownloaded int // Tracks number of megabytes downloaded.
	var kbps float64        // Kilobytes per second.

	//start := time.Now()
	sec := time.NewTicker(time.Second)

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
			bytesDownloaded += n
			kbps += float64(n) / 1024
			done++

		case <-sec.C:

			c.UI.App.QueueUpdateDraw(
				func() {
					c.UI.Graph.Update(kbps)
				},
			)

			kbps = 0
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
// At the moment this breaks peer pages.

// TODO:  REIMPLEMENT

// func (c *Client) resetUploadPeers() {
// 	counts := [][]int{}
// 	for i, peer := range c.Peers {
// 		if peer.Active && !peer.IsChoking {
// 			counts = append(counts, []int{i, peer.Downloaded})
// 		}
// 		peer.Downloaded = 0
// 	}
// 	if len(counts) < 4 {
// 		for i := 0; i < len(counts); i++ {
// 			c.Peers[counts[i][0]].Upload = true
// 		}
// 	} else {
// 		sort.Slice(counts, func(i, j int) bool {
// 			return counts[i][1] > counts[j][1]
// 		})
// 		for i := 0; i < 4; i++ {
// 			c.Peers[counts[i][0]].Upload = true
// 		}
// 	}

// }
