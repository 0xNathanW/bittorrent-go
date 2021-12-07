package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"time"

	cli "github.com/0xNathanW/bittorrent-goV2/client"
)

func main() {

	torrentPath := "./KNOPPIX 7.2.0 CD.torrent"
	//os.Args[1]
	err := verifyPath(torrentPath)
	if err != nil {
		log.Fatal(err)
	}

	client, err := cli.NewClient(torrentPath)
	if err != nil {
		log.Fatal(err)
	}
	client.Run()
	time.Sleep(time.Second * 30)
}

// Verifies torrent file exists.
func verifyPath(path_ string) error {
	if _, err := os.Stat(path_); os.IsNotExist(err) {
		return fmt.Errorf("%s does not exist", path_)
	}
	if path.Ext(path_) != ".torrent" {
		return fmt.Errorf("%s is not a .torrent file", path_)
	}
	return nil
}
