package client

import (
	"github.com/0xNathanW/bittorrent-goV2/p2p"
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

	fileBuf := make([]byte, c.Torrent.Size)

	// Start workers.
	for _, peer := range c.Peers {
		c.operatePeer(peer, workQ, dataQ)
	}

}

// operatePeer is a goroutine that handles communication with a single peer.
// If an error occurs, the peer is disconnected.
func (c *Client) operatePeer(peer *p2p.Peer, workQ chan Piece, dataQ chan<- *PieceData) {
	// Connect to peer.
	err := peer.Connect()
	if err != nil {
		return
	}
	defer peer.Conn.Close()

}
