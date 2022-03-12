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

	ui.Graph.Object.SetBorder(true).
		SetTitle(" Download Speed (MB/s) ").
		SetBorderPadding(0, 0, 2, 2)

	ui.PeerTable.SetBorder(true).SetTitle(" Peers ")
	ui.PeerTable.SetSelectionChangedFunc(
		func(row, column int) {
			ui.PeerPages.SwitchToPage(ui.PeerTable.GetCell(row, 0).Text)
		},
	)

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
		"Download (MB/s)",
		"Upload (MB/s)",
		"Downloading",
		"Choked",
		"Choking",
	}

	for i := range columnNames {
		table.SetCell(0, i, &tview.TableCell{
			Text:          columnNames[i],
			Align:         tview.AlignLeft,
			Color:         tcell.ColorAquaMarine,
			NotSelectable: true,
		})
	}

	for r, peer := range peers {
		for c, name := range columnNames {

			var text string
			colour := tcell.ColorWhite

			switch name {
			case "IP":
				text = peer.IP.String()
				colour = tcell.ColorYellow

			case "Active":
				if peer.Active {
					text = "Yes"
					colour = tcell.ColorGreen
				} else {
					text = "No"
					colour = tcell.ColorRed
				}

			case "Download Speed (MB/s)":
				text = fmt.Sprintf("%.2f", peer.DownloadRate)

			case "Upload Speed (MB/s)":
				text = fmt.Sprintf("%.2f", peer.UploadRate)

			case "Downloading":
				text = boolString(peer.Downloading)

			case "Choked":
				text = boolString(peer.Choked)

			case "Choking":
				text = boolString(peer.IsChoking)
			}

			table.SetCell(r+1, c, &tview.TableCell{
				Reference: peer,
				Text:      text,
				Align:     tview.AlignLeft,
				Color:     colour,
			})

		}
	}
	return table.SetFixed(1, 0).
		SetSelectedStyle(tcell.StyleDefault.
			Foreground(tcell.ColorBlack).
			Background(tcell.ColorWhite))
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

func (ui *UI) UpdateTable(peers []*p2p.Peer) {

	columnNames := []string{
		"IP",
		"Active",
		"Download (MB/s)",
		"Upload (MB/s)",
		"Downloading",
		"Choked",
		"Choking",
	}

	for r, peer := range peers {
		for c, name := range columnNames {

			cell := ui.PeerTable.GetCell(r+1, c)
			switch name {

			case "Active":
				cell.SetText(boolString(peer.Active))
				if peer.Active {
					cell.SetTextColor(tcell.ColorGreen)
				} else {
					cell.SetTextColor(tcell.ColorRed)
				}

			case "Download Speed (MB/s)":
				cell.SetText(fmt.Sprintf("%4.2f", peer.DownloadRate))

			case "Upload Speed (MB/s)":
				cell.SetText(fmt.Sprintf("%4.2f", peer.UploadRate))

			case "Downloading":
				cell.SetText(boolString(peer.Downloading))

			case "Choked":
				cell.SetText(boolString(peer.Choked))

			case "Choking":
				cell.SetText(boolString(peer.IsChoking))

			default:
				continue
			}
		}
	}
}

func boolString(b bool) string {
	if b {
		return "Yes"
	} else {
		return "No"
	}
}
