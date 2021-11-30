package client

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/0xNathanW/bittorrent-goV2/p2p"
	"github.com/0xNathanW/bittorrent-goV2/p2p/message"
	"github.com/0xNathanW/bittorrent-goV2/torrent"
	"github.com/0xNathanW/bittorrent-goV2/tracker"
)

const clientPort = 6881

type Client struct {
	ID   [20]byte // The client's unique ID.
	Port int      // The port the client is listening on.

	Torrent *torrent.Torrent // The torrent the client is downloading.

	Peers []*p2p.Peer // Peers client has connection to.

	Tracker *tracker.Tracker // Tracker.

	// State *State	// Torrent download status.
	BitField *message.Bitfield // Current bitfield.
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
	// Get peers from tracker.
	peersString, err := c.Tracker.RequestPeers()
	fmt.Println(peersString)
	if err != nil {
		return err
	}
	// Parse peers.
	return nil
}

func (c *Client) PrintInfo() {
	fmt.Println("=== Client Info ===")
	fmt.Println("ID:", c.ID)
	fmt.Println("Port:", c.Port)
	fmt.Println("")
	c.Torrent.PrintInfo()
	fmt.Println("")
	c.Tracker.PrintInfo()
}
