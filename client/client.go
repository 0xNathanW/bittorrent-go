package client

import (
	"encoding/binary"
	"errors"
	"fmt"
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
	Peers    *Peers
	Tracker  *tracker.Tracker
	BitField message.Bitfield
	UI       *ui.UI
	Seed     *sync.Cond // Used to signal when to start seeding.
}

type Peers struct {
	sync.RWMutex
	active   map[string]*p2p.Peer // Maps peer IP to peers.
	inactive []*net.TCPAddr
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
	fmt.Println("found", len(client.Peers.active), "peers")
	fmt.Println("found", len(client.Peers.inactive), "inactive peers")

	ui, err := ui.NewUI(torrent, client.Peers.active)
	if err != nil {
		return nil, err
	}
	client.UI = ui
	client.Peers.RLock()

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
	fmt.Println("Retrieving peers...")
	peerString, err := c.Tracker.RequestPeers()
	if err != nil {
		return err
	}

	// Each peer is a string of length 6.
	numPeers := len(peerString) / 6
	c.Peers = &Peers{
		active:   map[string]*p2p.Peer{},
		inactive: []*net.TCPAddr{},
	}
	c.Peers.Lock()

	wg := sync.WaitGroup{}

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

		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			peer, err := p2p.NewPeer(address, len(c.BitField))
			if err != nil {
				c.Peers.Unlock()
				c.Peers.active[address.String()] = peer
				c.Peers.Lock()
			} else {
				c.Peers.Unlock()
				c.Peers.inactive = append(c.Peers.inactive, address)
				c.Peers.Lock()
			}
		}(i)
	}

	wg.Wait()
	if len(c.Peers.active) == 0 {
		return errors.New("no peers found")
	}

	return nil
}
