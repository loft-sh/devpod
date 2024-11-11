package huh

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// A Layout is responsible for laying out groups in a form.
type Layout interface {
	View(f *Form) string
	GroupWidth(f *Form, g *Group, w int) int
}

// Default layout shows a single group at a time.
var LayoutDefault Layout = &layoutDefault{}

// Stack layout stacks all groups on top of each other.
var LayoutStack Layout = &layoutStack{}

// Column layout distributes groups in even columns.
func LayoutColumns(columns int) Layout {
	return &layoutColumns{columns: columns}
}

// Grid layout distributes groups in a grid.
func LayoutGrid(rows int, columns int) Layout {
	return &layoutGrid{rows: rows, columns: columns}
}

type layoutDefault struct{}

func (l *layoutDefault) View(f *Form) string {
	return f.selector.Selected().View()
}

func (l *layoutDefault) GroupWidth(_ *Form, _ *Group, w int) int {
	return w
}

type layoutColumns struct {
	columns int
}

func (l *layoutColumns) visibleGroups(f *Form) []*Group {
	segmentIndex := f.selector.Index() / l.columns
	start := segmentIndex * l.columns
	end := start + l.columns

	total := f.selector.Total()
	if end > total {
		end = total
	}

	var groups []*Group
	f.selector.Range(func(i int, group *Group) bool {
		if i >= start && i < end {
			groups = append(groups, group)
			return true
		}
		return false
	})

	return groups
}

func (l *layoutColumns) View(f *Form) string {
	groups := l.visibleGroups(f)
	if len(groups) == 0 {
		return ""
	}

	columns := make([]string, 0, len(groups))
	for _, group := range groups {
		columns = append(columns, group.Content())
	}
	footer := f.selector.Selected().Footer()

	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, columns...),
		footer,
	)
}

func (l *layoutColumns) GroupWidth(_ *Form, _ *Group, w int) int {
	return w / l.columns
}

type layoutStack struct{}

func (l *layoutStack) View(f *Form) string {
	var columns []string
	f.selector.Range(func(_ int, group *Group) bool {
		columns = append(columns, group.Content())
		return true
	})
	footer := f.selector.Selected().Footer()

	var view strings.Builder
	view.WriteString(strings.Join(columns, "\n"))
	view.WriteString(footer)
	return view.String()
}

func (l *layoutStack) GroupWidth(_ *Form, _ *Group, w int) int {
	return w
}

type layoutGrid struct {
	rows, columns int
}

func (l *layoutGrid) visibleGroups(f *Form) [][]*Group {
	total := l.rows * l.columns
	segmentIndex := f.selector.Index() / total
	start := segmentIndex * total
	end := start + total

	if glen := f.selector.Total(); end > glen {
		end = glen
	}

	var visible []*Group
	f.selector.Range(func(i int, group *Group) bool {
		if i >= start && i < end {
			visible = append(visible, group)
			return true
		}
		return false
	})
	grid := make([][]*Group, l.rows)
	for i := 0; i < l.rows; i++ {
		startRow := i * l.columns
		endRow := startRow + l.columns
		if startRow >= len(visible) {
			break
		}
		if endRow > len(visible) {
			endRow = len(visible)
		}
		grid[i] = visible[startRow:endRow]
	}
	return grid
}

func (l *layoutGrid) View(f *Form) string {
	grid := l.visibleGroups(f)
	if len(grid) == 0 {
		return ""
	}

	rows := make([]string, 0, len(grid))
	for _, row := range grid {
		var columns []string
		for _, group := range row {
			columns = append(columns, group.Content())
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, columns...))
	}
	footer := f.selector.Selected().Footer()

	return lipgloss.JoinVertical(lipgloss.Left, strings.Join(rows, "\n"), footer)
}

func (l *layoutGrid) GroupWidth(_ *Form, _ *Group, w int) int {
	return w / l.columns
}
