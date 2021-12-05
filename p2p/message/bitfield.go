package message

/*The bitfield message is variable length, where X is the length of the bitfield.
The payload is a bitfield representing the pieces that have been successfully downloaded.
The high bit in the first byte corresponds to piece index 0.
Bits that are cleared indicated a missing piece, and set bits indicate a valid and available piece.
Spare bits at the end are set to zero.*/

// Each byte = 8 bits
type Bitfield []byte

func (b Bitfield) HasPiece(idx int) bool {
	if idx/8 >= len(b) || idx < 0 {
		return false
	}
	bit := b[idx/8] >> (7 - idx%8) & 1
	return bit != 0
}

func (b Bitfield) SetPiece(idx int) {
	if idx/8 >= len(b) || idx < 0 {
		return
	}
	b[idx/8] |= 1 << uint(7-idx%8)
}
