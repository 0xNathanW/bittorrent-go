package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/0xNathanW/bittorrent-go/p2p"
	"github.com/0xNathanW/bittorrent-go/torrent"
	"github.com/rivo/tview"
)

// Refresh rate for display.
const RefreshRate = time.Second / 60

const bannerTxt = `   ___ _ _  _____                          _          ___      
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
			SetRows(7, 0, 0, 0).
			SetMinSize(7, 64). // Row, Col
			SetBorders(false),

		Graph: newGraph(),

		PeerList: newPeerList(peers),

		PeerPages: newPeerPages(peers),

		ProgressBar: tview.NewTextView().
			SetScrollable(false),
	}

	ui.Graph.Object.SetBorder(true).SetTitle(" Download Speed (MB/s) ")

	ui.ProgressBar.SetBorder(true).SetTitle(" Download Progress ")
	ui.ProgressBar.Box.SetBorderPadding(1, 1, 4, 4)

	// Set the peer info page to change on manouvering list.
	ui.PeerList.SetChangedFunc(
		func(index int, mainText string, secondaryText string, shortcut rune) {
			ui.PeerPages.SwitchToPage(mainText)
		},
	)
	ui.PeerList.Box.SetBorderPadding(0, 0, 2, 0)
	ui.PeerList.SetCurrentItem(0)

	ui.drawLayout(t)
	ui.App.SetRoot(ui.Layout, true) // Set grid as the root primitive.
	return ui, nil
}

// Draws elements onto the grid.
func (ui *UI) drawLayout(t *torrent.Torrent) {

	banner := tview.NewTextView().
		SetScrollable(false).
		SetTextAlign(tview.AlignCenter)

	_, _, _, height := banner.Box.GetRect()
	verticalPadding := (height - 5) / 2 // Padding to center the banner vertically.
	fmt.Println(verticalPadding)
	banner.SetText(
		strings.Repeat("\n", verticalPadding) + bannerTxt,
	)
	banner.Box.SetBorder(false)
	// A element to display basic information about the torrent.
	infoText := fmt.Sprintf(
		"\n\tName: %s\n\tSize: %s\n\tInfo Hash: %s",
		t.Name, t.GetSize(), t.GetInfoHash(),
	)

	info := tview.NewTextView().
		SetText(infoText).
		SetScrollable(false).
		SetTextAlign(tview.AlignLeft)
	info.Box.SetBorder(true).SetTitle(" Torrent Info ")
	info.Box.SetBorderPadding(1, 1, 0, 0)

	// Adds elements to grid.
	ui.Layout.AddItem(
		banner,
		0, 0, 1, 1, // row, col, rowspan, colspan
		7, 63, false,
	).AddItem(
		info,
		0, 1, 1, 1,
		0, 0, false,
	).AddItem(
		ui.PeerList,
		1, 0, 1, 1,
		0, 0, false,
	).AddItem(
		ui.PeerPages,
		2, 0, 2, 1,
		0, 0, false,
	).AddItem(
		ui.Graph.Object,
		1, 1, 2, 1,
		0, 0, false,
	).AddItem(
		ui.ProgressBar,
		3, 1, 1, 1,
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
	ui.ProgressBar.Box.SetTitle(fmt.Sprintf(" Download Progress: %d%% ", progress))
	// Calculate the progress bar width.
	_, _, width, height := ui.ProgressBar.Box.GetInnerRect()
	repititons := int(float64(progress) / 100 * float64(width))
	var progressBar string
	for i := 0; i < height; i++ {
		progressBar += strings.Repeat("█", repititons)
		progressBar += "\n"
	}
	ui.ProgressBar.SetText(progressBar)
}

func newPeerList(peers []*p2p.Peer) *tview.List {
	peerList := tview.NewList()
	peerList.SetBorder(true).SetTitle(" Peers ")
	for i, peer := range peers {
		peerList.AddItem(
			formatPeerString(peer.IP.String(), i), formatActive(peer.Active), '>', nil,
		)
	}
	return peerList
}

func newPeerPages(peers []*p2p.Peer) *tview.Pages {
	peerPages := tview.NewPages()
	for i, peer := range peers {
		peerPages.AddPage(formatPeerString(peer.IP.String(), i), peer.Page, true, false)
	}
	peerPages.SwitchToPage(peers[0].IP.String())
	return peerPages
}

// Util functions.
func formatActive(active bool) string {
	if active {
		return "[green]Active"
	}
	return "[red]Inactive"
}

func formatPeerString(IP string, num int) string {
	return fmt.Sprintf("Peer: %d - %s", num, IP)
}
