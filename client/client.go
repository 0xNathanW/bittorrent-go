package client

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/0xNathanW/bittorrent-goV2/p2p"
	"github.com/0xNathanW/bittorrent-goV2/p2p/message"
	"github.com/0xNathanW/bittorrent-goV2/torrent"
	"github.com/0xNathanW/bittorrent-goV2/tracker"
	"github.com/0xNathanW/bittorrent-goV2/ui"
)

const clientPort = 6881

type Client struct {
	ID   [20]byte // The client's unique ID.
	Port int      // The port the client is listening on.

	Torrent *torrent.Torrent // The torrent the client is downloading.

	Peers []*p2p.Peer // Peers client has connection to.

	Tracker *tracker.Tracker // Tracker.

	BitField message.Bitfield // Current bitfield.

	UI *ui.UI // UI.
}

func NewClient(path string) (*Client, error) {
	// Uppack and parse torrent file.
	torrent, err := torrent.NewTorrent(path)
	if err != nil {
		return nil, err
	}
	client := &Client{
		ID:      idGenerator(),
		Port:    clientPort,
		Torrent: torrent,
	}

	// Generate bitfield.
	numPieces := len(torrent.Pieces)
	if numPieces%8 == 0 {
		client.BitField = make(message.Bitfield, numPieces/8)
	} else {
		client.BitField = make(message.Bitfield, numPieces/8+1)
	}

	// Setup the tracker.
	tracker, err := tracker.NewTracker(torrent.Announce, torrent.AnnounceList)
	if err != nil {
		return nil, err
	}
	tracker.InitParams(
		torrent.InfoHash,
		client.ID,
		client.Port,
		torrent.Size,
	)
	client.Tracker = tracker

	// Get peers from tracker.
	// err = client.GetPeers()
	// if err != nil {
	// 	return nil, err
	// }

	ui, err := ui.NewUI(torrent)
	client.UI = ui
	time.Sleep(time.Second * 5)

	return client, nil
}

// Generate a new client ID.
func idGenerator() [20]byte {
	rand.Seed(time.Now().UnixNano())
	var id [20]byte
	rand.Read(id[:])
	return id
}

func (c *Client) GetPeers() error {
	// Get peer info from tracker.
	peersString, err := c.Tracker.RequestPeers()
	if err != nil {
		return err
	}
	// Parse peers.
	c.Peers = p2p.ParsePeers(peersString, len(c.BitField))
	return nil
}

func (c *Client) PrintInfo() {
	fmt.Println("=== Client Info ===")
	fmt.Println("ID:", c.ID)
	fmt.Println("Port:", c.Port)
	fmt.Println("Bitfield", c.BitField)
	fmt.Println("")
	c.Torrent.PrintInfo()
	fmt.Println("")
	c.Tracker.PrintInfo()
	fmt.Println("")
	fmt.Println("=== Peer Info ===")
	for _, peer := range c.Peers {
		peer.PrintInfo()
	}
}
