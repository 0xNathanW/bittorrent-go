package client

import (
	"crypto/sha1"
	"fmt"

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

	//fileBuf := make([]byte, c.Torrent.Size)
	fmt.Println("Starting peer operations")
	// Start workers.
	testPeer := c.Peers[2]
	c.operatePeer(testPeer, workQ, dataQ)

}

// operatePeer is a goroutine that handles communication with a single peer.
// If an error occurs, the peer is disconnected and we return from function.
func (c *Client) operatePeer(peer *p2p.Peer, workQ chan Piece, dataQ chan<- *PieceData) {
	fmt.Println("Operating peer: ", peer.IP.String())

	// Establish connection with peer.
	err := c.establishPeer(peer)
	if err != nil {
		fmt.Println("Error establishing connection with peer:", err)
		return
	}
	fmt.Println("Connection established with peer: ", peer.IP.String())
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
			fmt.Println("Error downloading piece:", err)
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
			fmt.Println("Piece hash mismatch")
			workQ <- piece
			continue
		}
		// Send piece to dataQ.
		fmt.Println("Piece downloaded:", piece.Index)
		dataQ <- &PieceData{piece.Index, data}
	}
}

func (c *Client) establishPeer(peer *p2p.Peer) error {
	// Connect to peer.
	err := peer.Connect()
	if err != nil {
		return err
	}

	err = peer.ExchangeHandshake(c.ID, c.Torrent.InfoHash)
	if err != nil {
		return err
	}

	// Peers will then send messages about what pieces they have.
	// This can come in many forms, eg bitfield or have msgs.
	// This is where we will parse the message and set the peer's bitfield.
	err = peer.BuildBitfield()
	if err != nil {
		return err
	}
	return nil
}
