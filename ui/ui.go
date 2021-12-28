package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/0xNathanW/bittorrent-go/p2p"
	"github.com/0xNathanW/bittorrent-go/torrent"
	"github.com/gdamore/tcell/v2"
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
	Progress    *tview.Frame
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
			SetScrollable(false).
			SetTextColor(tcell.ColorBlue),
	}

	ui.Graph.Object.SetBorder(true).SetTitle(" Download Speed (MB/s) ")

	ui.Progress = tview.NewFrame(ui.ProgressBar)
	ui.Progress.SetBorder(true)
	ui.ProgressBar.SetBorder(true)

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
		ui.Progress,
		3, 1, 1, 1,
		0, 0, false,
	)
}

func (ui *UI) UpdateProgress(done, total int) {
	ui.Progress.SetTitle(fmt.Sprintf(" Download Progress: %d%% ", (done*100)/total))
	ui.Progress.Clear()
	ui.Progress.AddText(
		fmt.Sprintf(" %d/%d pieces downloaded\n\n", done, total),
		true, tview.AlignLeft, tcell.ColorWhite)
	// Calculate the progress bar width.
	_, _, _, height1 := ui.Progress.GetInnerRect()
	ui.Progress.SetBorders(height1/4, 0, 0, 0, 2, 2)
	_, _, width, height2 := ui.ProgressBar.GetInnerRect()
	repititions := (done * 100 / total) * width / 100
	var progress string
	for i := 0; i < height2; i++ {
		progress += strings.Repeat("█", repititions)
		if i != height2-1 {
			progress += "\n"
		}
	}
	ui.ProgressBar.SetText(progress)
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
