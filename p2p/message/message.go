package message

import (
	"encoding/binary"
	"fmt"
)

type Message struct {
	Length  []byte
	ID      byte
	Payload []byte
}

var MsgIDmap = map[byte]string{
	0:    "Choke",
	1:    "Unchoke",
	2:    "Interested",
	3:    "Not Interested",
	4:    "Have",
	5:    "Bitfield",
	6:    "Request",
	7:    "Piece",
	8:    "Cancel",
	0x54: "Handshake",
}

// Serialize message to byte array.
func (m Message) SerialiseMsg() []byte {
	buf := make([]byte, len(m.Length)+len(m.Payload)+1)
	var n int
	n += copy(buf[n:], m.Length[:])
	n += copy(buf[n:], []byte{m.ID})
	n += copy(buf[n:], m.Payload)
	return buf
}

// Pads to the left to 4 byte array
func numToBuffer(num int) []byte {
	buf := make([]byte, 4)
	for i := len(buf) - 1; num != 0; i-- {
		buf[i] = byte(num & 0xff)
		num >>= 8
	}
	return buf
}

func (m Message) PrintInfo() {
	fmt.Print("\n-- Message --\n")
	fmt.Printf("Length: %d\n", binary.BigEndian.Uint32(m.Length[:]))
	fmt.Printf("Type: %v", MsgIDmap[m.ID])
}

// -------------------- Messages --------------------//

func Choke() []byte {
	msg := Message{Length: []byte{0, 0, 0, 1}, ID: 0}
	return msg.SerialiseMsg()
}

func Unchoke() []byte {
	msg := Message{Length: []byte{0, 0, 0, 1}, ID: 1}
	return msg.SerialiseMsg()
}

func Interested() []byte {
	msg := Message{Length: []byte{0, 0, 0, 1}, ID: 2}
	return msg.SerialiseMsg()
}

func NotInterested() []byte {
	msg := Message{Length: []byte{0, 0, 0, 1}, ID: 3}
	return msg.SerialiseMsg()
}

func Have(idx int) []byte {
	msg := Message{
		Length:  []byte{0, 0, 0, 5},
		ID:      4,
		Payload: numToBuffer(idx),
	}
	return msg.SerialiseMsg()
}

func Request(idx, begin, length int) []byte {
	payloadBuf := make([]byte, 12)
	var n int
	n += copy(payloadBuf[n:], numToBuffer(idx))
	n += copy(payloadBuf[n:], numToBuffer(begin))
	n += copy(payloadBuf[n:], numToBuffer(length))
	msg := Message{
		Length:  []byte{0, 0, 0, 13},
		ID:      6,
		Payload: payloadBuf,
	}
	return msg.SerialiseMsg()
}

func Cancel(idx, begin, length int) []byte {
	payloadBuf := make([]byte, 12)
	var n int
	n += copy(payloadBuf[n:], numToBuffer(idx))
	n += copy(payloadBuf[n:], numToBuffer(begin))
	n += copy(payloadBuf[n:], numToBuffer(length))

	msg := Message{
		Length:  []byte{0, 0, 0, 13},
		ID:      8,
		Payload: payloadBuf,
	}
	return msg.SerialiseMsg()
}
