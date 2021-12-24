package torrent

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"os"

	"github.com/jackpal/bencode-go"
)

// Frames enable the torrent file to be unmarshalled from bencoded form.
type TorrentFrame struct {
	info         InfoFrame `bencode:"info"`
	announce     string    `bencode:"announce"`
	announceList []string  `bencode:"announce-list"`
}

type InfoFrame struct {
	name         string      `bencode:"name"`
	size         int         `bencode:"length"`
	piecesString string      `bencode:"pieces"`
	pieceLength  int         `bencode:"piece length"`
	files        []FileFrame `bencode:"files"`
}

type FileFrame struct {
	length int      `bencode:"length"`
	path   []string `bencode:"path"`
}

// ParseTorrent parses the torrent file and returns a TorrentFrame struct.
func unpackFile(path string) (*TorrentFrame, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open torrent file: %w", err)
	}
	defer file.Close()

	var frame TorrentFrame                // Declare frame.
	err = bencode.Unmarshal(file, &frame) // Unmarshalling into frame struct.
	if err != nil {
		return nil, fmt.Errorf("could not parse torrent file: %w", err)
	}
	// Piece hashes should all be 20 bytes long.
	if len(frame.info.piecesString)%20 != 0 {
		return nil, fmt.Errorf("invalid pieces length: %d", len(frame.info.piecesString))
	}
	return &frame, nil
}

// Parses frame into a Torrent struct.
func (f *TorrentFrame) parse(path string) (*Torrent, error) {
	infoHash, err := getInfoHash(path)
	if err != nil {
		return nil, err
	}
	//Sets size as sum of all file sizes if the torrent is multifile.
	size := f.info.size
	if size == 0 {
		for _, file := range f.info.files {
			size += file.length
		}
	}
	// Parse file info.
	files := make([]File, len(f.info.files))
	for i, file := range f.info.files {
		files[i] = File{
			Length: file.length,
			Path:   file.path[0],
		}
	}
	torrent := &Torrent{
		Name:         f.info.name,
		Announce:     f.announce,
		AnnounceList: f.announceList,
		InfoHash:     infoHash,
		Size:         size,
		PieceLength:  f.info.pieceLength,
		Pieces:       f.info.splitPieces(),
		Files:        files,
	}
	return torrent, nil
}

// Each piece is a 20 byte SHA1 hash.
func (i *InfoFrame) splitPieces() [][20]byte {
	buf := []byte(i.piecesString)
	pieces := make([][20]byte, len(buf)/20)
	for i := 0; i < len(pieces); i++ {
		copy(pieces[i][:], buf[i*20:(i+1)*20])
	}
	return pieces
}

// Calculates the SHA1 hash of the info dict.
// This is used to verify the integrity of the torrent file.
func getInfoHash(path string) ([20]byte, error) {
	packed, _ := os.Open(path)
	defer packed.Close()
	raw, err := bencode.Decode(packed)
	if err != nil {
		return [20]byte{}, fmt.Errorf("could not decode torrent file: %s", err)
	}
	if data, ok := raw.(map[string]interface{}); ok {
		buffer := bytes.Buffer{}
		err := bencode.Marshal(&buffer, data["info"])
		if err != nil {
			return [20]byte{}, err
		}
		return sha1.Sum(buffer.Bytes()), nil
	} else {
		return [20]byte{}, fmt.Errorf("could not decode torrent file: %s", err)
	}
}
