package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/0xNathanW/bittorrent-go/p2p"
	"github.com/0xNathanW/bittorrent-go/torrent"
	"github.com/gdamore/tcell/v2"
	"github.com/navidys/tvxwidgets"
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
	App       *tview.Application
	Layout    *tview.Grid
	Graph     *Graph
	Progress  *tvxwidgets.PercentageModeGauge
	PeerTable *tview.Table
	PeerPages *tview.Pages

	rightFlex *tview.Flex
}

// Creates a new UI instance.
func NewUI(t *torrent.Torrent, peers []*p2p.Peer) (*UI, error) {

	ui := &UI{
		App: tview.NewApplication(),

		Layout: tview.NewGrid().
			SetColumns(-1, -1). // Two equal sized columns.
			SetRows(7, -1, -1).
			SetMinSize(0, 64). // Row, Col
			SetBorders(false),

		Graph: newGraph(),

		PeerTable: newPeerTable(peers),

		PeerPages: newPeerPages(peers),

		Progress: tvxwidgets.NewPercentageModeGauge(),

		rightFlex: tview.NewFlex().
			SetDirection(tview.FlexRow),
	}

	ui.Progress.SetMaxValue(len(t.Pieces))
	ui.Progress.SetBorder(true).SetTitle(" Progress ")

	ui.rightFlex.AddItem(ui.Graph.Object, 0, 1, false)
	ui.rightFlex.AddItem(ui.Progress, 5, 0, false)

	// ui.Progress.SetMaxValue(len(t.Pieces))
	// ui.Progress.SetTitle(" Download Progress ")

	ui.PeerTable.SetSelectionChangedFunc(
		func(row, column int) {
			ui.PeerPages.SwitchToPage(ui.PeerTable.GetCell(row, 0).Text)
		},
	)

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
	banner.SetText(
		strings.Repeat("\n", verticalPadding) + bannerTxt,
	)
	banner.Box.SetBorder(false)

	// An element to display basic information about the torrent.
	infoText := fmt.Sprintf(
		"\n\tName: %s\n\tSize: %s\n\tInfo Hash: %s",
		t.Name, t.GetSize(), t.GetInfoHash(),
	)

	info := tview.NewTextView().
		SetText(infoText).
		SetScrollable(false).
		SetTextAlign(tview.AlignLeft)

	info.SetBorder(true).
		SetTitle(" Torrent Info ").
		SetBorderPadding(1, 1, 0, 0)

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
		0, 0, true,
	).AddItem(
		ui.PeerPages,
		2, 0, 1, 1,
		0, 0, false,
	).AddItem(
		ui.rightFlex,
		1, 1, 2, 1,
		0, 0, false,
	)
}

func (ui *UI) UpdateProgress(done int) {
	ui.Progress.SetValue(done)
}

func newPeerTable(peers []*p2p.Peer) *tview.Table {

	table := tview.NewTable().
		SetSelectable(true, false). // Enable row selection.
		SetEvaluateAllRows(true).
		SetFixed(1, 0). // Fix the first row.
		SetSelectedStyle(tcell.StyleDefault.
			Foreground(tcell.ColorBlack).
			Background(tcell.ColorWhite)).
		SetSeparator(tview.Borders.Vertical)

	table.SetBorder(true).SetTitle(" Peers ")

	columnNames := []string{
		"IP",
		"Active",
		"Down (MB/s)",
		"Up (MB/s)",
		"Downloading",
		"Choked",
		"Choking",
	}

	// First row is the column names.
	for i := range columnNames {
		table.SetCell(0, i, &tview.TableCell{
			Text:          columnNames[i],
			Align:         tview.AlignLeft,
			Color:         tcell.ColorAquaMarine,
			NotSelectable: true,
			Attributes:    tcell.AttrUnderline,
		})
	}

	// Fill table.
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

			default:
				text = ""
			}

			table.SetCell(r+1, c, &tview.TableCell{
				Reference: peer,
				Text:      text,
				Align:     tview.AlignLeft,
				Color:     colour,
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

func (ui *UI) UpdateTable(peers []*p2p.Peer) {

	columnNames := []string{
		"IP",
		"Active",
		"Down (MB/s)",
		"Up (MB/s)",
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

			case "Down (MB/s)":
				cell.SetText(fmt.Sprintf("%4.2f", peer.DownloadRate))

			case "Up (MB/s)":
				cell.SetText(fmt.Sprintf("%4.2f", peer.UploadRate))

			case "Downloading":
				cell.SetText(boolString(peer.Downloading))
				if peer.Downloading {
					cell.SetTextColor(tcell.ColorBlue)
				} else {
					cell.SetTextColor(tcell.ColorRed)
				}

			case "Choked":
				cell.SetText(boolString(peer.Choked))
				if !peer.Choked {
					cell.SetTextColor(tcell.ColorBlue)
				} else {
					cell.SetTextColor(tcell.ColorDefault)
				}

			case "Choking":
				cell.SetText(boolString(peer.IsChoking))
				if !peer.IsChoking {
					cell.SetTextColor(tcell.ColorBlue)
				} else {
					cell.SetTextColor(tcell.ColorDefault)
				}

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
