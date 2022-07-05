package ui

import (
	"github.com/guptarohit/asciigraph"
	"github.com/rivo/tview"
)

type Graph struct {
	Data   []float64
	Object *tview.TextView
}

// Creates a new graph instance.
func newGraph() *Graph {
	graph := &Graph{

		Object: tview.NewTextView().
			SetTextAlign(tview.AlignLeft).
			SetScrollable(false).
			SetWrap(false),

		Data: make([]float64, 15),
	}

	graph.Object.
		SetBorderPadding(2, 2, 2, 2).
		SetBorder(true).
		SetTitle(" Download Speed (MB/s) ").
		SetBorderPadding(0, 0, 2, 2)

	return graph
}

// Takes a new value and updates graph,
// keeping it the same width.
func (g *Graph) Update(data float64) {

	g.smooth(data)
	_, _, width, height := g.Object.GetInnerRect()

	g.Object.SetText(asciigraph.Plot(g.Data,
		asciigraph.Width(width),
		asciigraph.Height(height),
		asciigraph.Precision(2),
		asciigraph.Caption("Download Speed (MB/s)"),
	))
}

func (g *Graph) smooth(data float64) {

	for i := 9; i > 5; i-- {
		data = data + g.Data[i]
	}
	data = data / 5

	g.Data = append(g.Data, data)
	g.Data = g.Data[1:]
}
