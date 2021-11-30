package message

/*All of the remaining messages in the protocol take the form of:
<length prefix><message ID><payload>.
The length prefix is a four byte big-endian value.
The message ID is a single decimal byte.
The payload is message dependent.*/

// The handshake is a required message and must be the first message transmitted by the client.
// It is (49+len(pstr)) bytes long.
// handshake: <pstrlen><pstr><reserved><info_hash><peer_id>
func Handshake(id, iHash [20]byte) []byte {
	pstr := "BitTorrent protocol"
	buf := make([]byte, 49+len(pstr))
	buf[0] = byte(len(pstr))
	n := 1
	n += copy(buf[n:], []byte(pstr))
	n += copy(buf[n:], make([]byte, 8))
	n += copy(buf[n:], iHash[:])
	n += copy(buf[n:], id[:])
	return buf
}
