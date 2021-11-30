package torrent

import (
	"fmt"
	//"crypto/sha1"
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
	frame, err := UnpackFile(path)
	if err != nil {
		return nil, err
	}
	torrent, err := frame.Parse(path)
	if err != nil {
		return nil, err
	}
	return torrent, nil
}

func (t *Torrent) PrintInfo() {
	fmt.Println("=== Torrent Info ===")
	fmt.Printf("Name: %s\n", t.Name)
	fmt.Printf("Announce: %s\n", t.Announce)
	fmt.Printf("AnnounceList: %v\n", t.AnnounceList)
	fmt.Printf("InfoHash: %x\n", t.InfoHash)
	fmt.Printf("Size: %d\n", t.Size)
	fmt.Printf("Piece length: %d\n", t.PieceLength)
	fmt.Printf("Pieces: %d\n", len(t.Pieces))
	fmt.Println("")
	fmt.Println("=== Files ===")
	for _, file := range t.Files {
		fmt.Printf("%s\n", file.Path)
		fmt.Printf("%d\n", file.Length)
	}
}
