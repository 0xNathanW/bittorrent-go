package ui

import (
	"strings"

	"github.com/gdamore/tcell"
	"github.com/guptarohit/asciigraph"
)

type Graph struct {
	Data   []float64
	Height int
	Width  int
}

func (d *Display) DrawGraph() {
	graph := asciigraph.Plot(
		d.Graph.Data,
		asciigraph.Height(d.Graph.Height),
		asciigraph.Width(d.Graph.Width),
	)
	lines := strings.Split(graph, "\n")
	for y, line := range lines {
		for x, c := range line {
			d.Screen.SetContent(x+1, y+1, c, nil, tcell.StyleDefault)
		}
	}
}
