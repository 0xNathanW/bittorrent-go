package ui

import (
	"fmt"

	"github.com/0xNathanW/bittorrent-goV2/torrent"
	"github.com/rivo/tview"
)

type UI struct {
	App         *tview.Application
	RefreshRate int
	Layout      *tview.Grid
	Graph       *Graph
}

const banner = `   ___ _ _  _____                          _          ___      
  / __(_) |/__   \___  _ __ _ __ ___ _ __ | |_       / _ \___  
 /__\// | __|/ /\/ _ \| '__| '__/ _ \ '_ \| __|____ / /_\/ _ \ 
/ \/  \ | |_/ / | (_) | |  | | |  __/ | | | ||_____/ /_\\ (_) |
\_____/_|\__\/   \___/|_|  |_|  \___|_| |_|\__|    \____/\___/ 
`

func NewUI(t *torrent.Torrent) (*UI, error) {

	ui := &UI{
		// App will run the main loop for the UI.
		App: tview.NewApplication(),
		// Layout is the grid holding the UI elements.
		Layout: tview.NewGrid().
			SetColumns(0, 0).
			SetRows(6, -1, 0).
			SetBorders(true),

		Graph:       NewGraph(),
		RefreshRate: 60,
	}
	ui.Layout.Box.SetTitle("Bittorrent-Go")

	ui.Graph.Object.SetChangedFunc(func() {
		ui.App.Draw()
	})
	ui.drawLayout(t)
	return ui, nil
}

// drawLayout adds the elements to the grid layout.
func (ui *UI) drawLayout(t *torrent.Torrent) {
	ui.drawHeader(t)
	ui.drawGraph()
}

func (ui *UI) drawHeader(t *torrent.Torrent) {

	banner := tview.NewTextView().
		SetText(banner).
		SetScrollable(false).
		SetTextAlign(tview.AlignCenter)

	// A element to display basic information about the torrent.
	infoText := fmt.Sprintf(
		"\n   === Torrent Info ===\n  Name: %s\n  Size: %s\n  Info Hash: %s\n",
		t.Name, t.GetSize(), t.GetInfoHash(),
	)

	info := tview.NewTextView().
		SetText(infoText).
		SetScrollable(false).
		SetTextAlign(tview.AlignLeft)
	info.Box.SetBorderPadding(1, 1, 1, 1)
	info.Box.SetTitle("Info")

	// Adds elements to grid.
	ui.Layout.AddItem(
		banner,
		0, 0, 1, 1, // row, col, rowspan, colspan
		5, 63, false,
	).AddItem(
		info,
		0, 1, 1, 1,
		0, 0, false,
	)
}

func (ui *UI) drawGraph() {
	ui.Layout.AddItem(
		ui.Graph.Object,
		1, 1, 1, 1, // row, col, rowspan, colspan
		10, 50, false)
}
