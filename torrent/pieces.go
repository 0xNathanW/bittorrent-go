package torrent

import "fmt"

type Piece struct {
	Index  int
	Length int
	Hash   [20]byte
}

type PieceData struct {
	Index int
	Data  []byte
}

func (t *Torrent) NewWorkQueue() chan Piece {

	workQ := make(chan Piece, len(t.Pieces))
	for idx, hash := range t.Pieces {
		workQ <- Piece{idx, t.PieceSize(idx), hash}
	}

	return workQ
}

// Returns the begin and end index of a piece.
func (t *Torrent) PieceBounds(idx int) (int, int) {
	begin := idx * t.PieceLength
	end := begin + t.PieceLength
	if end > t.Size {
		end = t.Size
	}
	return begin, end
}

func (t *Torrent) PieceSize(idx int) int {
	begin, end := t.PieceBounds(idx)
	return end - begin
}

func (t *Torrent) PiecePosition(idx int) (int, int, error) {
	begin, end := t.PieceBounds(idx)
	if begin < 0 || end > t.Size {
		return 0, 0, fmt.Errorf("piece bounds out of bounds")
	}
	return begin, end, nil
}
