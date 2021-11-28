package client

import (
	"math/rand"
	"time"
)

const clientPort = 6881

type Client struct {
	ID   [20]byte // The client's unique ID.
	Port int      // The port the client is listening on.

	// Torrent *Torrent	// The torrent the client is downloading.

	// Peers []*Peer	// Peers client has connection to.

	// Tracker *Tracker	// Tracker.
	// BackupTrackers []*Tracker	// Backup tracker.

	// State *State	// Torrent download status.
	// BitField *BitField	// Current bitfield.

}

func newClient() *Client {
	client := &Client{
		ID:   idGenerator(),
		Port: clientPort,
	}
	return client
}

// Generate a new client ID.
func idGenerator() [20]byte {
	rand.NewSource(time.Now().UnixNano())
	var id [20]byte
	for i := 0; i < 20; i++ {
		id[i] = byte(rand.Intn(256))
	}
	return id
}

func (client *Client) unpackTorrent() error {

}
