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
			SetScrollable(false),

		Data: make([]float64, 50),
	}
	graph.Object.Box.SetBorderPadding(1, 1, 1, 1)
	graph.Update(0) // Intialise with 0.
	return graph
}

// Takes a new value and updates graph,
// keeping it the same width.
func (g *Graph) Update(data float64) {
	g.Data = append(g.Data, data)
	g.Data = g.Data[1:]
	_, _, width, height := g.Object.GetInnerRect()
	g.Object.SetText(asciigraph.Plot(g.Data,
		asciigraph.Width(width),
		asciigraph.Height(height),
		asciigraph.Precision(2),
		asciigraph.Caption("Download Speed (MB/s)"),
	))
}
