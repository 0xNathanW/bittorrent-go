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
	PeerTable   *tview.Table
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

		PeerTable: newPeerTable(peers),

		PeerPages: newPeerPages(peers),

		ProgressBar: tview.NewTextView().
			SetScrollable(false).
			SetTextColor(tcell.ColorBlue),
	}

	ui.Graph.Object.SetBorder(true).SetTitle(" Download Speed (MB/s) ")

	ui.Progress = tview.NewFrame(ui.ProgressBar)
	ui.Progress.SetBorder(true)
	ui.ProgressBar.SetBorder(true)

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
		ui.PeerTable,
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

func (ui *UI) UpdateProgress(remaining time.Duration, done, total int) {

	ui.Progress.SetTitle(fmt.Sprintf(" Download Progress: %d%% ", (done*100)/total))

	ui.Progress.Clear()
	ui.Progress.AddText(
		fmt.Sprintf(" %d/%d pieces downloaded\n", done, total),
		true, tview.AlignLeft, tcell.ColorWhite)
	ui.Progress.AddText("", true, tview.AlignLeft, tcell.ColorWhite)
	ui.Progress.AddText(
		fmt.Sprintf(" Time remaining: %s", remaining.Round(time.Second).String()),
		true, tview.AlignLeft, tcell.ColorWhite)
	ui.Progress.AddText("", true, tview.AlignLeft, tcell.ColorWhite)
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

func newPeerTable(peers []*p2p.Peer) *tview.Table {

	table := tview.NewTable().
		SetSelectable(true, false).
		SetEvaluateAllRows(true)

	columnNames := []string{
		"IP",
		"Active",
		"Download Speed (MB/s)",
		"Upload Speed (MB/s)",
		"Downloading",
		"Choked",
		"Choking",
	}

	for i := range columnNames {
		table.SetCell(0, i, &tview.TableCell{
			Text:          columnNames[i],
			Align:         tview.AlignCenter,
			Color:         tcell.ColorBlue,
			NotSelectable: true,
		})
	}

	for r, peer := range peers {
		for c, name := range columnNames {

			var text string
			switch name {
			case "IP":
				text = peer.IP.String()
			case "Active":
				if peer.Active {
					text = "[green]Yes[-]"
				} else {
					text = "[red]No[-]"
				}
			case "Download Speed (MB/s)":
				text = fmt.Sprintf("%.2f", peer.DownloadRate)
			case "Upload Speed (MB/s)":
				text = fmt.Sprintf("%.2f", peer.UploadRate)
			case "Downloading":
				text = "N/A"
			case "Choked":
				if peer.Choked {
					text = "[red]Yes[-]"
				} else {
					text = "[green]No[-]"
				}
			case "Choking":
				if peer.IsChoking {
					text = "[red]Yes[-]"
				} else {
					text = "[green]No[-]"
				}
			}

			table.SetCell(r+1, c, &tview.TableCell{
				Reference: peer,
				Text:      text,
				Align:     tview.AlignCenter,
			})

		}
	}
	return table
}

func newPeerPages(peers []*p2p.Peer) *tview.Pages {
	peerPages := tview.NewPages()
	for _, peer := range peers {
		peerPages.AddPage(
			peer.IP.String(), peer.Activity, true, false)
	}
	peerPages.SwitchToPage(peers[0].IP.String())
	return peerPages
}
