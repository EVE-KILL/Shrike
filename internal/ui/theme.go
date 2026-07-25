// Package ui owns every byte the CLI writes for human consumption: colours,
// tables, banners, progress. Nothing outside this package should reach for
// lipgloss directly, so that --no-color and --json have exactly one place to
// take effect.
package ui

import (
	"image/color"
	"os"

	"charm.land/lipgloss/v2"
	"golang.org/x/term"
)

// ColorEnabled reports whether styled output is wanted. Every helper in this
// package must branch on it and emit plain ASCII when false — the CLI has to
// stay readable in `kubectl logs`, CI output, and pipes, which is exactly
// where a killboard operator reads it most.
var ColorEnabled = true

// JSONMode suppresses all decorative output. Commands that support structured
// output check it before printing anything human-facing, so stdout carries
// only the JSON document and stays pipeable into jq.
var JSONMode = false

// Init resolves output modes from the persistent flags plus the environment.
// Explicit flags win; otherwise we honour NO_COLOR and refuse to emit escape
// sequences when stdout is not a terminal.
func Init(jsonOut, noColor bool) {
	JSONMode = jsonOut

	switch {
	case noColor, jsonOut:
		ColorEnabled = false
	case os.Getenv("NO_COLOR") != "":
		ColorEnabled = false
	case !term.IsTerminal(int(os.Stdout.Fd())):
		ColorEnabled = false
	default:
		ColorEnabled = true
	}
}

// Palette. Tuned for dark terminals, which is where this runs — the numbers
// are deliberately close to GitHub's dark theme so red reads as "loss" and
// cyan as "chrome" without any further explanation.
var (
	ColorPrimary   = lipgloss.Color("#58A6FF") // EVE cyan-blue — headings, command names
	ColorSecondary = lipgloss.Color("#7C8DA6") // steel — supporting values
	ColorAccent    = lipgloss.Color("#F85149") // killboard red — the KILL half of the logo
	ColorSuccess   = lipgloss.Color("#56D364")
	ColorError     = lipgloss.Color("#F85149")
	ColorWarn      = lipgloss.Color("#E3B341")
	ColorDim       = lipgloss.Color("#6E7681")
	ColorWhite     = lipgloss.Color("#FFFFFF")
)

var (
	StyleBold      = lipgloss.NewStyle().Bold(true)
	StyleDim       = lipgloss.NewStyle().Foreground(ColorDim)
	StylePrimary   = lipgloss.NewStyle().Foreground(ColorPrimary)
	StyleSecondary = lipgloss.NewStyle().Foreground(ColorSecondary)
	StyleAccent    = lipgloss.NewStyle().Foreground(ColorAccent)
	StyleSuccess   = lipgloss.NewStyle().Foreground(ColorSuccess)
	StyleError     = lipgloss.NewStyle().Foreground(ColorError)
	StyleWarn      = lipgloss.NewStyle().Foreground(ColorWarn)
	StyleSection   = lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary)
	StyleCommand   = lipgloss.NewStyle().Foreground(ColorSuccess)

	badgeStyle = lipgloss.NewStyle().Bold(true).Padding(0, 1)
)

// Bold, Dim, Primary, Secondary, Accent and Command render a fragment in the
// requested style, or return it untouched when colour is off. They return
// strings rather than printing so callers can compose them into tables.
func Bold(s string) string      { return render(StyleBold, s) }
func Dim(s string) string       { return render(StyleDim, s) }
func Primary(s string) string   { return render(StylePrimary, s) }
func Secondary(s string) string { return render(StyleSecondary, s) }
func Accent(s string) string    { return render(StyleAccent, s) }
func Command(s string) string   { return render(StyleCommand, s) }

// Warn2 renders a fragment in the warning colour. Named to avoid colliding with
// output.go's Warn, which prints a whole line to stderr.
func Warn2(s string) string { return render(StyleWarn, s) }

func render(style lipgloss.Style, s string) string {
	if !ColorEnabled {
		return s
	}
	return style.Render(s)
}

// StatusBadge renders a health verdict. The plain-text form keeps the same
// column width intent by bracketing, so degraded `doctor` output still lines
// up when piped to a file.
func StatusBadge(status string) string {
	var bg color.Color
	switch status {
	case "ok":
		bg = ColorSuccess
	case "warn":
		bg = ColorWarn
	case "fail":
		bg = ColorError
	default:
		bg = ColorDim
	}
	if !ColorEnabled {
		return "[" + status + "]"
	}
	return badgeStyle.Foreground(ColorWhite).Background(bg).Render(status)
}
