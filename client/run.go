package client

import (
	"context"
	"os"
	"sort"
	"time"

	"github.com/0xNathanW/bittorrent-go/p2p"
	"github.com/0xNathanW/bittorrent-go/torrent"
)

func (c *Client) Run() {
	// workQ is the queue of pieces we need to download.
	// If a worker is available, it will be given a piece from the queue.
	// If a worker fails to download a piece, it will be put back on the queue.
	workQ := c.Torrent.NewWorkQueue()
	defer close(workQ)

	// dataQ is a recieves piece data from workers.
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
	go c.collectPieces(dataQ, requestQ)

	// Run tview event loop.
	if err := c.UI.App.SetFocus(c.UI.PeerTable).Run(); err != nil {
		panic(err)
	}
}

func (c *Client) collectPieces(dataQ <-chan *torrent.PieceData, requestQ <-chan [3]int) {

	buf := make([]byte, c.Torrent.Size) // Output buffer.

	var done int            // Tracks number of pieces downloaded.
	var bytesDownloaded int // Tracks number of megabytes downloaded.
	var mbps float64        // Kilobytes per second.
	sec := time.NewTicker(time.Second)
	sec10 := time.NewTicker(time.Second * 10)

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
			mbps += float64(n) / 1024 / 1024 // Convert to megabytes.
			done++

		case request := <-requestQ:

		case <-sec.C:

			c.UI.App.QueueUpdateDraw(
				func() {

					c.UI.Graph.Update(mbps)
					c.UI.UpdateTable(c.Peers)
					c.UI.UpdateProgress(done)

				},
			)
			mbps = 0

		case <-sec10.C:
			c.chokingAlgo()
		}
	}
	// Write output buffer to file.
	c.writeToFile(buf)

	// Logic for transition to seeding.
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

	last := make(map[*p2p.Peer][2]int)
	ticker := time.NewTicker(time.Second * 10)

	for range ticker.C {

		for _, peer := range c.Peers {
			if peer.Active {
				peer.DownloadRate = peer.Downloaded - last[peer][0]
				peer.UploadRate = peer.Uploaded - last[peer][1]
			} else {
				peer.DownloadRate = 0
				peer.UploadRate = 0
			}
		}

		uploadSort := c.Peers
		sort.Slice(uploadSort, func(i, j int) bool {
			return uploadSort[i].DownloadRate > uploadSort[j].DownloadRate
		})

		for i := 0; i < 4; i++ {
			for _, peer := range c.Peers {

				if peer.IP.String() == uploadSort[i].IP.String() {
					peer.Downloading = true
					peer.Activity.Write([]byte("[green]serving requests from peer.[-]"))
				} else {
					peer.Downloading = false
				}
			}
		}
	}
}
