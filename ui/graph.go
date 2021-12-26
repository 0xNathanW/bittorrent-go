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
			SetText(asciigraph.Plot(
				make([]float64, 50),
				asciigraph.Width(50),
				asciigraph.Height(10),
			)).
			SetTextAlign(tview.AlignLeft).
			SetScrollable(false),

		Data: make([]float64, 50),
	}

	graph.Object.Box.SetBorderPadding(3, 3, 1, 1)
	return graph
}

// Takes a new value and updates graph,
// keeping it the same width.
func (g *Graph) Update(data float64) {
	g.Data = append(g.Data, data)
	g.Data = g.Data[1:]
	g.Object.SetText(asciigraph.Plot(g.Data,
		asciigraph.Width(50),
		asciigraph.Height(10),
		asciigraph.Precision(2),
		asciigraph.Caption("Download Speed (MB/s)"),
	))
}
