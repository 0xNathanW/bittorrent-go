package torrent

import (
	"encoding/hex"
	"strconv"
)

type Torrent struct {
	Name         string
	Announce     string
	AnnounceList []string
	InfoHash     [20]byte
	Size         int
	PieceLength  int
	Pieces       [][20]byte
	Files        []File
}

type File struct {
	Path   string
	Length int
}

func NewTorrent(path string) (*Torrent, error) {
	frame, err := unpackFile(path)
	if err != nil {
		return nil, err
	}
	torrent, err := frame.parse(path)
	if err != nil {
		return nil, err
	}
	return torrent, nil
}

// Returns string repersentation of size.
func (t *Torrent) GetSize() string {
	var size string
	if t.Size > 1000000000 {
		size = strconv.Itoa(t.Size/1000000000) + "GB"
	} else {
		size = strconv.Itoa(t.Size/1000000) + "MB"
	}
	return size
}

// Returns infohash hexstring.
func (t *Torrent) GetInfoHash() string {
	return hex.EncodeToString(t.InfoHash[:])
}
