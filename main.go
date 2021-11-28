package main

import (
	"fmt"
	"log"
	"os"
	"path"

	ui "github.com/gizak/termui/v3"
)

func main() {

	torrentFile := os.Args[1]
	err := verifyPath(torrentFile)
	if err != nil {
		log.Fatal(err)
	}

	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	_ = cli.newClient()

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
