package main

import (
	"fmt"
	"log"
	"os"
	"path"

	cli "github.com/0xNathanW/bittorrent-goV2/client"
)

func main() {

	torrentPath := os.Args[1]
	err := verifyPath(torrentPath)
	if err != nil {
		log.Fatal(err)
	}

	// if err := ui.Init(); err != nil {
	// 	log.Fatalf("failed to initialize termui: %v", err)
	// }
	// defer ui.Close()

	client, err := cli.NewClient(torrentPath)
	if err != nil {
		log.Fatal(err)
	}
	client.PrintInfo()
	client.GetPeers()
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
