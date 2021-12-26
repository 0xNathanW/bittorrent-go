package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/0xNathanW/bittorrent-go/p2p"
	"github.com/0xNathanW/bittorrent-go/torrent"
	"github.com/rivo/tview"
)

// Refresh rate for display.
const RefreshRate = time.Second / 60

const banner = `   ___ _ _  _____                          _          ___      
  / __(_) |/__   \___  _ __ _ __ ___ _ __ | |_       / _ \___  
 /__\// | __|/ /\/ _ \| '__| '__/ _ \ '_ \| __|____ / /_\/ _ \ 
/ \/  \ | |_/ / | (_) | |  | | |  __/ | | | ||_____/ /_\\ (_) |
\_____/_|\__\/   \___/|_|  |_|  \___|_| |_|\__|    \____/\___/ 
`

type UI struct {
	App         *tview.Application
	Layout      *tview.Grid
	Graph       *Graph
	ProgressBar *tview.TextView
	PeerList    *tview.List
	PeerPages   *tview.Pages
}

// Creates a new UI instance.
func NewUI(t *torrent.Torrent, peers []*p2p.Peer) (*UI, error) {

	ui := &UI{
		App: tview.NewApplication(),

		Layout: tview.NewGrid().
			SetColumns(0, 0). // Two equal sized columns.
			SetRows(6, -4, -1).
			SetMinSize(10, 64). // Row, Col
			SetBorders(false),

		Graph: newGraph(),

		PeerList: newPeerList(peers),

		PeerPages: newPeerPages(peers),

		ProgressBar: tview.NewTextView().
			SetScrollable(false),
	}

	ui.Graph.Object.SetBorder(true).SetTitle(" Download Speed (MB/s) ")

	ui.ProgressBar.SetBorder(true).SetTitle(" Download Progress ")
	ui.ProgressBar.Box.SetBorderPadding(2, 1, 4, 4)

	// Set the peer info page to change on manouvering list.
	ui.PeerList.SetChangedFunc(
		func(index int, mainText string, secondaryText string, shortcut rune) {
			ui.PeerPages.SwitchToPage(mainText)
		},
	)
	ui.PeerList.SetCurrentItem(0)

	ui.drawLayout(t)
	ui.App.SetRoot(ui.Layout, true) // Set grid as the root primitive.
	return ui, nil
}

// Draws elements onto the grid.
func (ui *UI) drawLayout(t *torrent.Torrent) {

	banner := tview.NewTextView().
		SetText(banner).
		SetScrollable(false).
		SetTextAlign(tview.AlignCenter)

	// A element to display basic information about the torrent.
	infoText := fmt.Sprintf(
		"\tName: %s\n\tSize: %s\n\tInfo Hash: %s",
		t.Name, t.GetSize(), t.GetInfoHash(),
	)

	info := tview.NewTextView().
		SetText(infoText).
		SetScrollable(false).
		SetTextAlign(tview.AlignLeft)
	info.Box.SetBorder(true).SetTitle(" Torrent Info ")
	info.Box.SetBorderPadding(0, 0, 1, 1)

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

// Refreshes display.
func (ui *UI) Refresh() {
	tick := time.NewTicker(RefreshRate)
	for range tick.C {
		ui.App.Draw()
	}
	defer tick.Stop()
}

func (ui *UI) UpdateProgress(progress int) {
	ui.ProgressBar.Box.SetTitle(fmt.Sprintf(" Download Progress %d%% ", progress))
	// Calculate the progress bar width.
	repititons := int(float64(progress) / 100 * 60)
	ui.ProgressBar.SetText(fmt.Sprintf(
		"%s\n%s\n%s\n%s",
		strings.Repeat("█", repititons),
		strings.Repeat("█", repititons),
		strings.Repeat("█", repititons),
		strings.Repeat("█", repititons),
	))
}

func newPeerList(peers []*p2p.Peer) *tview.List {
	peerList := tview.NewList()
	peerList.SetBorder(true).SetTitle(" Peers ")
	for i, peer := range peers {
		peerList.AddItem(peer.IP.String(), strconv.FormatBool(peer.Active), rune(i), nil)
	}
	return peerList
}

func newPeerPages(peers []*p2p.Peer) *tview.Pages {
	peerPages := tview.NewPages()
	for _, peer := range peers {
		peerPages.AddPage(peer.IP.String(), peer.Page, true, false)
	}
	peerPages.SwitchToPage(peers[0].IP.String())
	return peerPages
}
