package ui

import (
	"github.com/0xNathanW/bittorrent-goV2/torrent"
	"github.com/rivo/tview"
)

type UI struct {
	App    *tview.Application
	Layout *tview.Grid
	//Graph *Graph
}

const banner = `   ___ _ _  _____                          _          ___      
  / __(_) |/__   \___  _ __ _ __ ___ _ __ | |_       / _ \___  
 /__\// | __|/ /\/ _ \| '__| '__/ _ \ '_ \| __|____ / /_\/ _ \ 
/ \/  \ | |_/ / | (_) | |  | | |  __/ | | | ||_____/ /_\\ (_) |
\_____/_|\__\/   \___/|_|  |_|  \___|_| |_|\__|    \____/\___/ 
`

func NewUI(t *torrent.Torrent) (*UI, error) {
	ui := &UI{
		App: tview.NewApplication(),
		Layout: tview.NewGrid().
			SetColumns(0, 0).
			SetRows(6, -2, 0).
			SetBorders(true),
		//Graph: NewGraph(),

	}
	ui.drawHeader(t)
	if err := ui.App.SetRoot(ui.Layout, true).Run(); err != nil {
		return nil, err
	}
	return ui, nil
}

func (ui *UI) drawHeader(t *torrent.Torrent) {
	ui.Layout.AddItem(
		tview.NewTextView().
			SetText(banner).
			SetScrollable(false).
			SetTextAlign(tview.AlignCenter),
		0, 0, 1, 1, // row, col, rowspan, colspan
		5, 63, false)

	infoText := "\n  === Torrent Info ===\n" +
		"  Name: " + t.Name + "\n" +
		"  Size: " + t.GetSize() + "\n" +
		"  Info Hash: " + t.GetInfoHash() + "\n"

	ui.Layout.AddItem(
		tview.NewTextView().
			SetText(infoText).
			SetScrollable(false).
			SetTextAlign(tview.AlignLeft),
		0, 1, 1, 1, // row, col, rowspan, colspan
		0, 0, false)
}

// func (ui *UI) drawGraph() {

// }
