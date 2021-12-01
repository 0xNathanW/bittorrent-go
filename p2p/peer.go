package p2p

import (
	"fmt"
	"net"

	"github.com/0xNathanW/bittorrent-goV2/p2p/message"
)

type Peer struct {
	PeerID        [20]byte
	IP            net.IP
	Port          string
	Conn          net.Conn
	BitField      message.Bitfield
	Choked        bool
	Interested    bool
	IsChoking     bool
	IsInteresting bool
	Strikes       int
}

func ParsePeers(peerString string, bfLength int) []*Peer {
	var peers []*Peer
	numPeers := len(peerString) / 6
	for i := 0; i < numPeers; i++ {
		ip := peerString[i*6 : i*6+4]
		port := peerString[i*6+4 : i*6+6]
		peer := &Peer{
			IP:            net.IP{ip[0], ip[1], ip[2], ip[3]},
			Port:          port,
			BitField:      make(message.Bitfield, bfLength),
			Choked:        true,
			Interested:    false,
			IsChoking:     true,
			IsInteresting: false,
			Strikes:       0,
		}
		peers = append(peers, peer)
	}
	return peers
}

func (p *Peer) PrintInfo() {
	fmt.Println("PeerID:", p.PeerID)
	fmt.Println("IP:", p.IP.String())
	fmt.Println("Port:", []byte(p.Port))
	fmt.Println("Choked:", p.Choked)
	fmt.Println("Interested:", p.Interested)
	fmt.Println("IsChoking:", p.IsChoking)
	fmt.Println("IsInteresting:", p.IsInteresting)
	fmt.Println("Strikes:", p.Strikes)
}
