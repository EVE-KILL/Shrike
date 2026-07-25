package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

// Table is a thin builder over lipgloss's table with a plain-text fallback.
// The fallback is not a nicety: bordered box-drawing output is unreadable in
// log aggregators, and `doctor --json` aside, that is where most of this ends
// up being read.
type Table struct {
	headers []string
	rows    [][]string
}

func NewTable(headers ...string) *Table {
	return &Table{headers: headers}
}

func (t *Table) Row(cells ...string) *Table {
	t.rows = append(t.rows, cells)
	return t
}

func (t *Table) Render() string {
	if len(t.rows) == 0 {
		return Dim("  No results.")
	}
	if !ColorEnabled {
		return t.renderPlain()
	}

	tbl := table.New().
		Headers(t.headers...).
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(ColorDim)).
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == table.HeaderRow {
				return lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Padding(0, 1)
			}
			return lipgloss.NewStyle().Padding(0, 1)
		})

	for _, r := range t.rows {
		tbl.Row(r...)
	}
	return tbl.Render()
}

// renderPlain aligns on visible width. Cells may carry ANSI escapes even in
// plain mode (a caller can style a single cell), so width is measured with
// lipgloss.Width rather than len to avoid padding drift.
func (t *Table) renderPlain() string {
	widths := make([]int, len(t.headers))
	for i, h := range t.headers {
		widths[i] = lipgloss.Width(h)
	}
	for _, row := range t.rows {
		for i, cell := range row {
			if i < len(widths) {
				if w := lipgloss.Width(cell); w > widths[i] {
					widths[i] = w
				}
			}
		}
	}

	var sb strings.Builder
	writeRow := func(cells []string) {
		for i, cell := range cells {
			if i > 0 {
				sb.WriteString("  ")
			}
			if i < len(widths) {
				sb.WriteString(cell)
				sb.WriteString(strings.Repeat(" ", widths[i]-lipgloss.Width(cell)))
			} else {
				sb.WriteString(cell)
			}
		}
		sb.WriteString("\n")
	}

	writeRow(t.headers)
	for i, w := range widths {
		if i > 0 {
			sb.WriteString("  ")
		}
		sb.WriteString(strings.Repeat("-", w))
	}
	sb.WriteString("\n")
	for _, row := range t.rows {
		writeRow(row)
	}
	return strings.TrimRight(sb.String(), "\n")
}

// List renders label/description pairs with the labels aligned — the shape the
// help renderer wants for command listings. Labels are measured, not fixed,
// because command names vary a lot in length between groups.
func List(pairs [][2]string, indent string) string {
	if len(pairs) == 0 {
		return ""
	}
	width := 0
	for _, p := range pairs {
		if w := lipgloss.Width(p[0]); w > width {
			width = w
		}
	}
	var sb strings.Builder
	for _, p := range pairs {
		pad := strings.Repeat(" ", width-lipgloss.Width(p[0]))
		fmt.Fprintf(&sb, "%s%s%s  %s\n", indent, p[0], pad, Dim(p[1]))
	}
	return strings.TrimRight(sb.String(), "\n")
}
