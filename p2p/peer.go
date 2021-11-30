package p2p

import (
	"net"

	"github.com/0xNathanW/bittorrent-goV2/p2p/message"
)

type Peer struct {
	PeerID        [20]byte
	IP            net.IP
	Port          uint16
	Conn          net.Conn
	BitField      message.Bitfield
	Choked        bool
	Interested    bool
	IsChoking     bool
	IsInteresting bool
	Strikes       int
}

func NewPeer() *Peer {
	return &Peer{}
}
