package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/0xNathanW/bittorrent-go/torrent"
	"github.com/rivo/tview"
)

// Refresh rate for display.
const RefreshRate = time.Second / 60

type UI struct {
	App          *tview.Application
	Layout       *tview.Grid
	Graph        *Graph
	ActivityFeed *tview.TextView
	Logger       *tview.TextView
	ProgressBar  *tview.TextView
}

const banner = `   ___ _ _  _____                          _          ___      
  / __(_) |/__   \___  _ __ _ __ ___ _ __ | |_       / _ \___  
 /__\// | __|/ /\/ _ \| '__| '__/ _ \ '_ \| __|____ / /_\/ _ \ 
/ \/  \ | |_/ / | (_) | |  | | |  __/ | | | ||_____/ /_\\ (_) |
\_____/_|\__\/   \___/|_|  |_|  \___|_| |_|\__|    \____/\___/ 
`

// Creates a new UI instance.
func NewUI(t *torrent.Torrent) (*UI, error) {

	ui := &UI{

		App: tview.NewApplication(),

		Layout: tview.NewGrid().
			SetColumns(64, 64).
			SetRows(6, 20, 10).
			SetBorders(false),

		Graph: NewGraph(),

		ActivityFeed: tview.NewTextView().
			ScrollToEnd().
			SetMaxLines(25).
			SetScrollable(true).
			SetDynamicColors(true),

		Logger: tview.NewTextView().
			SetScrollable(true).
			SetDynamicColors(true).
			ScrollToEnd().
			SetMaxLines(15),

		ProgressBar: tview.NewTextView().
			SetScrollable(false).
			SetScrollable(false),
	}
	ui.ActivityFeed.SetBorder(true).SetTitle(" Activity Feed ")
	ui.Graph.Object.SetBorder(true).SetTitle(" Download Speed (MB/s) ")
	ui.ProgressBar.SetBorder(true).SetTitle(" Download Progress ")
	ui.Logger.SetBorder(true).SetTitle(" Log ")
	ui.ActivityFeed.Box.SetBorderPadding(1, 1, 2, 2)
	ui.Logger.Box.SetBorderPadding(0, 0, 2, 2)
	ui.ProgressBar.Box.SetBorderPadding(2, 1, 4, 4)

	ui.drawLayout(t)
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
	).AddItem(
		ui.Graph.Object,
		1, 1, 1, 1,
		10, 50, false,
	).AddItem(
		ui.ActivityFeed,
		1, 0, 1, 1,
		10, 64, false,
	).AddItem(
		ui.Logger,
		2, 0, 1, 1,
		0, 0, false,
	).AddItem(
		ui.ProgressBar,
		2, 1, 1, 1,
		0, 60, false,
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

// Writes to the activity feed.
func (ui *UI) UpdateActivity(peer, event string, active, peers int) {
	ui.ActivityFeed.Box.SetTitle(fmt.Sprintf(" Activity Feed - %d/%d peers active ",
		active, peers))
	ui.ActivityFeed.Write([]byte(fmt.Sprintf("[%s] %s\n", peer, event)))
}

// Writes to the error log.
func (ui *UI) UpdateLogger(message string) {
	ui.Logger.Write([]byte(message + "\n"))
}

func (ui *UI) UpdateProgress(progress int) {
	ui.ProgressBar.Box.SetTitle(fmt.Sprintf(" Download Progress %d%% ", progress))
	// Calculate the progress bar width.
	repitions := int(float64(progress) / 100 * 60)
	ui.ProgressBar.SetText(fmt.Sprintf(
		"%s\n%s\n%s\n%s",
		strings.Repeat("█", repitions),
		strings.Repeat("█", repitions),
		strings.Repeat("█", repitions),
		strings.Repeat("█", repitions),
	))
}
