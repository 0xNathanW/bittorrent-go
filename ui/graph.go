package ui

import (
	"github.com/guptarohit/asciigraph"
	"github.com/rivo/tview"
)

type Graph struct {
	Data   []float64
	Object *tview.TextView
}

func NewGraph() *Graph {
	graph := &Graph{
		Object: tview.NewTextView().
			SetText(asciigraph.Plot(
				make([]float64, 50),
				asciigraph.Width(50),
				asciigraph.Height(15),
				asciigraph.Caption("DownloadSpeed (MB/s)"),
			)).
			SetTextAlign(tview.AlignLeft).
			SetScrollable(false),
		Data: make([]float64, 50),
	}
	graph.Object.Box.SetBorderPadding(2, 2, 2, 2)
	return graph
}

func (g *Graph) Update(data float64) {
	g.Data = append(g.Data, data)
	g.Data = g.Data[1:]
	g.Object.SetText(asciigraph.Plot(g.Data,
		asciigraph.Width(50),
		asciigraph.Height(15),
		asciigraph.Precision(2),
		asciigraph.Caption("Download Speed (MB/s))"),
	))
}
