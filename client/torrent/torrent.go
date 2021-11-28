package torrent

import (
	"fmt"
	"os"

	//"crypto/sha1"
	"github.com/jackpal/bencode-go"
)

type Torrent struct {
	Name     string `bencode:"name"`
	Info     *Info  `bencode:"info"`
	InfoHash [20]byte
}

type Info struct {
	Size         int    `bencode:"length"`
	PiecesString string `bencode:"pieces"`
	Pieces       [][20]byte
	PieceLength  int    `bencode:"piece length"`
	Files        []File `bencode:"files"`
}

type File struct {
	Length int    `bencode:"length"`
	Path   string `bencode:"path"`
}

// ParseTorrent parses the torrent file and returns a Torrent struct.
func UnpackTorrent(path string) (*Torrent, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Could not open torrent file: %s", err)
	}
	defer file.Close()

	var torrent Torrent
	err = bencode.Unmarshal(file, &torrent)
	if err != nil {
		return nil, fmt.Errorf("Could not parse torrent file: %s", err)
	}

	if len(torrent.Info.Pieces)%20 != 0 {
		return nil, fmt.Errorf("Invalid pieces length: %d", len(torrent.Info.Pieces))
	}

	torrent.Info.Pieces = make([][20]byte, len(torrent.Info.PiecesString)/20)
	for i := 0; i < len(torrent.Info.Pieces); i++ {
		copy(torrent.Info.Pieces[i][:], torrent.Info.PiecesString[i*20:(i+1)*20])
	}
	// torrent.InfoHash = torrent.InfoHash(path)

	return &torrent, nil
}

// func infoHash(path string) ([20]byte, error) {

// }

func (t *Torrent) PrintInfo() {
	fmt.Println("=== Torrent Info ===")
	fmt.Printf("Name: %s\n", t.Name)
	fmt.Printf("Size: %d\n", t.Info.Size)
	fmt.Printf("Piece length: %d\n", t.Info.PieceLength)
	fmt.Printf("Pieces: %d\n", len(t.Info.Pieces))
	fmt.Printf("Files: %d\n", len(t.Info.Files))
}
