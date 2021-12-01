package ui

import (
	"encoding/hex"
	"strconv"

	"github.com/0xNathanW/bittorrent-goV2/torrent"
	"github.com/gdamore/tcell"
)

type Display struct {
	Screen tcell.Screen
	Height int
	Width  int
	Graph  *Graph
}

func NewDisplay() (*Display, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}
	if err = screen.Init(); err != nil {
		return nil, err
	}

	width, height := screen.Size()

	screen.Clear()
	screen.SetStyle(tcell.StyleDefault.
		Foreground(tcell.ColorHotPink).
		Background(tcell.ColorBlack))
	screen.Show()

	display := &Display{
		Screen: screen,
		Height: height,
		Width:  width,
		Graph: &Graph{
			Height: 10,
			Width:  width / 2,
			Data:   []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
	}

	return display, nil
}

// Draw header should only be called once on startup.
func (d *Display) DrawHeader(t *torrent.Torrent) {

	var size string
	if t.Size > 1000000000 {
		size = "\tSize: " + strconv.Itoa(t.Size/1000000000) + "GB"
	} else {
		size = "\tSize: " + strconv.Itoa(t.Size/1000000) + "MB"
	}

	info := []string{
		"=== Torrent Info ===",
		"Name: " + t.Name,
		"Size: " + size,
		"InfoHash: " + hex.EncodeToString(t.InfoHash[:]),
	}

	lines := []string{
		`   ___ _ _  _____                          _          ___      `,
		`  / __(_) |/__   \___  _ __ _ __ ___ _ __ | |_       / _ \___  `,
		` /__\// | __|/ /\/ _ \| '__| '__/ _ \ '_ \| __|____ / /_\/ _ \ `,
		`/ \/  \ | |_/ / | (_) | |  | | |  __/ | | | ||_____/ /_\\ (_) |`,
		`\_____/_|\__\/   \___/|_|  |_|  \___|_| |_|\__|    \____/\___/ `,
	}

	for i, line := range lines {
		for j, char := range line {
			d.Screen.SetContent(j, i, char, nil, tcell.StyleDefault)
		}
	}

	for i, line := range info {
		for j, char := range line {
			// Prints 2 spaces after title.
			d.Screen.SetContent(j+len(lines[0])+2, i+1, char, nil, tcell.StyleDefault)
		}
	}
}
