package client

import (
	"fmt"

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

	//fileBuf := make([]byte, c.Torrent.Size)
	fmt.Println("Starting peer operations")
	// Start workers.
	testPeer := c.Peers[1]
	c.operatePeer(testPeer, workQ, dataQ)

}

// operatePeer is a goroutine that handles communication with a single peer.
// If an error occurs, the peer is disconnected and we return from function.
func (c *Client) operatePeer(peer *p2p.Peer, workQ chan Piece, dataQ chan<- *PieceData) {
	peer.PrintInfo()

	// Connect to peer.
	err := peer.Connect()
	if err != nil {
		return
	}
	defer peer.Conn.Close()
	fmt.Println("Successful connection to peer")
	err = peer.ExchangeHandshake(c.ID, c.Torrent.InfoHash)
	if err != nil {
		return
	}
	fmt.Println("Handshake complete with", peer.PeerID)
	// Peers will then send messages about what pieces they have.
	// This can come in many forms, eg bitfield or have msgs.
	// This is where we will parse the message and set the peer's bitfield.
	//msgs := peer.Read()

}
