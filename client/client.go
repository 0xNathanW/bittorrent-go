package client

import (
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/0xNathanW/bittorrent-go/p2p"
	"github.com/0xNathanW/bittorrent-go/p2p/message"
	"github.com/0xNathanW/bittorrent-go/torrent"
	"github.com/0xNathanW/bittorrent-go/tracker"
	"github.com/0xNathanW/bittorrent-go/ui"
)

// Client is the highest level of the application.
type Client struct {
	ID       [20]byte // The client's unique ID.
	Torrent  *torrent.Torrent
	Peers    map[string]*p2p.Peer
	Active   *active
	Tracker  *tracker.Tracker
	BitField message.Bitfield
	UI       *ui.UI
	Seed     *sync.Cond // Used to signal when to start seeding.

	Logger *log.Logger
}

type active struct {
	sync.Mutex
	int
}

// Create a new client instance.
// Contains all information required to start download.
func NewClient(path string) (*Client, error) {

	// Unpack and parse torrent file.
	torrent, err := torrent.NewTorrent(path)
	if err != nil {
		return nil, err
	}

	client := &Client{ // Client instance.
		ID:      idGenerator(),
		Torrent: torrent,
		Active:  &active{int: 0},
	}

	// Generate empty bitfield.
	numPieces := len(torrent.Pieces)
	if numPieces%8 == 0 {
		client.BitField = make(message.Bitfield, numPieces/8)
	} else {
		client.BitField = make(message.Bitfield, numPieces/8+1)
	}

	// Setup tracker.
	tracker, err := tracker.NewTracker(torrent.Announce, torrent.AnnounceList)
	if err != nil {
		return nil, err
	}
	tracker.InitParams(torrent.InfoHash, client.ID, torrent.Size)
	client.Tracker = tracker

	if err = client.GetPeers(); err != nil {
		return nil, err
	}

	ui, err := ui.NewUI(torrent, client.Peers)
	if err != nil {
		return nil, err
	}
	client.UI = ui

	return client, nil
}

// Generate a new client ID.
func idGenerator() [20]byte {
	rand.Seed(time.Now().UnixNano())
	var id [20]byte
	rand.Read(id[:])
	return id
}

// Client retrieves and parses peers from tracker.
func (c *Client) GetPeers() error {

	peerString, err := c.Tracker.RequestPeers()
	if err != nil {
		return err
	}

	fmt.Println("here")

	// Each peer is a string of length 6.
	numPeers := len(peerString) / 6

	peers := make(map[string]*p2p.Peer, numPeers)

	for i := 0; i < numPeers; i++ {

		ip := [6]byte{}
		copy(ip[:], peerString[i*6:i*6+4])
		p := [2]byte{}
		copy(p[:], peerString[i*6+4:i*6+6])

		port := binary.BigEndian.Uint16(p[:])

		tcpIP := strconv.Itoa(int(ip[0])) + "." +
			strconv.Itoa(int(ip[1])) + "." +
			strconv.Itoa(int(ip[2])) + "." +
			strconv.Itoa(int(ip[3])) + ":" +
			strconv.Itoa(int(port))

		address, err := net.ResolveTCPAddr("tcp", tcpIP)
		if err != nil {
			print("failed to resolve address:", peerString[i*6:(i+1)*6], err, "\n")
			continue
		}

		peers[address.String()] = p2p.NewPeer(address, len(c.BitField))

	}
	c.Peers = peers

	return nil
}
