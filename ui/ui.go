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
func NewUI(t *torrent.Torrent, peers map[string]*p2p.Peer) (*UI, error) {

	ui := &UI{
		App: tview.NewApplication(),

		Layout: tview.NewGrid().
			SetColumns(-1, -1). // Two equal sized columns.
			SetRows(7, -1, -1).
			SetMinSize(0, 64). // Row, Col
			SetBorders(false),

		Graph: newGraph(),

		PeerPages: newPeerPages(peers),

		Progress: tvxwidgets.NewPercentageModeGauge(),

		rightFlex: tview.NewFlex().
			SetDirection(tview.FlexRow),
	}

	ui.Progress.SetMaxValue(len(t.Pieces))
	ui.Progress.SetBorder(true).SetTitle(" Progress ")

	ui.rightFlex.AddItem(ui.Graph.Object, 0, 1, false)
	ui.rightFlex.AddItem(ui.Progress, 5, 0, false)

	ui.newPeerTable(peers)
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

func (ui *UI) newPeerTable(peers map[string]*p2p.Peer) {

	s := tcell.Style{}.
		Background(tcell.ColorWhite).
		Foreground(tcell.ColorBlack).
		Blink(true)

	table := tview.NewTable().
		SetSelectable(true, false). // Enable row selection.
		SetSelectedStyle(s).
		SetEvaluateAllRows(true).
		SetFixed(1, 0). // Fix the first row (column labels).
		SetSeparator(tview.Borders.Vertical)

	table.SetBorder(true).SetTitle(" Peers ")

	columnNames := []string{
		"IP",
		"Active",
		"Down",
		"Up",
		"Reciprocate",
		"Choked",
		"IsChoking",
	}

	// First row is the column names.
	for i := range columnNames {

		cell := &tview.TableCell{
			Text:          columnNames[i],
			Align:         tview.AlignCenter,
			Color:         tcell.ColorOrange,
			NotSelectable: true,
			Attributes:    tcell.AttrUnderline,
		}
		cell.SetTransparency(true).
			SetExpansion(1)

		table.SetCell(0, i, cell)
	}

	// Fill table.
	row := 1
	for _, peer := range peers {
		for c := range columnNames {

			var alignment int // Align all center apart from IP column.
			if c != 0 {
				alignment = 1
			}

			colour := tcell.ColorWhite
			cell := &tview.TableCell{
				Reference: peer,
				Align:     alignment,
				Color:     colour,
			}
			cell.SetTransparency(true).
				SetExpansion(1)

			table.SetCell(row, c, cell)
		}
		row++
	}

	ui.PeerTable = table
	ui.UpdateTable()
}

func newPeerPages(peers map[string]*p2p.Peer) *tview.Pages {

	peerPages := tview.NewPages()

	for address, peer := range peers {
		peerPages.AddPage(strings.Split(address, ":")[0], peer.Activity, true, false)
	}

	name, _ := peerPages.GetFrontPage()
	peerPages.SwitchToPage(name)
	return peerPages
}

func (ui *UI) UpdateTable() {

	columnNames := []string{
		"IP",
		"Active",
		"Down",
		"Up",
		"Reciprocate",
		"Choked",
		"IsChoking",
	}

	for r := 1; r < ui.PeerTable.GetRowCount(); r++ {
		for c, name := range columnNames {

			cell := ui.PeerTable.GetCell(r, c)
			peer := cell.Reference.(*p2p.Peer)

			switch name {

			case "IP":
				cell.SetText(strings.Split(peer.IP.String(), ":")[0])

			case "Active":
				cell.SetText(boolString(peer.Active))
				if peer.Active {
					cell.SetTextColor(tcell.ColorGreen)
				} else {
					cell.SetTextColor(tcell.ColorRed)
				}

			// For display, down/upload speed simply amount/(seconds since start)
			case "Down":
				if peer.Active {
					cell.SetText(fmt.Sprintf("%4.2f",
						(float64(peer.Rates.Downloaded)/1024/1024)/
							(time.Since(peer.Start).Seconds())))
				} else {
					cell.SetText(fmt.Sprintf("%4.2f", float64(0)))
				}

			case "Up":
				if peer.Active {
					cell.SetText(fmt.Sprintf("%4.2f",
						(float64(peer.Rates.Uploaded)/1024/1024)/
							(time.Since(peer.Start).Seconds())))
				} else {
					cell.SetText(fmt.Sprintf("%4.2f", float64(0)))
				}

			case "Reciprocate":
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
					cell.SetTextColor(tcell.ColorWhite)
				}

			case "IsChoking":
				cell.SetText(boolString(peer.IsChoking))
				if !peer.IsChoking {
					cell.SetTextColor(tcell.ColorBlue)
				} else {
					cell.SetTextColor(tcell.ColorWhite)
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
