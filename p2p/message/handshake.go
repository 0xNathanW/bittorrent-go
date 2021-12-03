package message

import (
	"bytes"
	"fmt"
)

/*All of the remaining messages in the protocol take the form of:
<length prefix><message ID><payload>.
The length prefix is a four byte big-endian value.
The message ID is a single decimal byte.
The payload is message dependent.*/

// The handshake is a required message and must be the first message transmitted by the client.
// It is (49+len(pstr)) bytes long.
// handshake: <pstrlen><pstr><reserved><info_hash><peer_id>
func Handshake(ID, infoHash [20]byte) []byte {
	pstr := "BitTorrent protocol"
	buf := make([]byte, 49+len(pstr))
	buf[0] = byte(len(pstr))
	n := 1
	n += copy(buf[n:], []byte(pstr))
	n += copy(buf[n:], make([]byte, 8))
	n += copy(buf[n:], infoHash[:])
	n += copy(buf[n:], ID[:])
	return buf
}

func VerifyHandshake(handshake []byte, infoHash [20]byte) ([20]byte, error) {
	if len(handshake) != 68 {
		return [20]byte{}, fmt.Errorf("handshake length error")
	}
	if string(handshake[1:20]) != "BitTorrent protocol" {
		return [20]byte{}, fmt.Errorf("handshake error, wrong protocol")
	}
	if !bytes.Equal(handshake[28:48], infoHash[:]) {
		return [20]byte{}, fmt.Errorf("handshake error, wrong info hash")
	}
	var ID [20]byte
	copy(ID[:], handshake[48:])
	return ID, nil
}
