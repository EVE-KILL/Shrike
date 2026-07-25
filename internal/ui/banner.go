package ui

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

// Version is stamped at build time via -ldflags. See the Makefile.
var (
	Version = "dev"
	Commit  = "unknown"
)

// The logo is split so "EVE" and "KILL" can be tinted differently — blue
// chrome, killboard red. Kept as two blocks joined horizontally rather than
// one pre-coloured string so the two halves stay independently restylable.
const logoEVE = `██████ ██    ██ ███████
██     ██    ██ ██
█████  ██    ██ █████
██      ██  ██  ██
██████   ████   ███████ `

const logoKILL = `██  ██ ██ ██      ██
██ ██  ██ ██      ██
████   ██ ██      ██
██ ██  ██ ██      ██
██  ██ ██ ███████ ███████`

// The logo stays EVE-KILL — that is the product. Shrike is the name of this
// binary, which the version line carries.
const tagline = "the EVE-KILL backend: API, workers, and crons"

// Banner renders the logo plus version line, or a single plain line when
// colour is off. Nothing downstream should assume a fixed height.
func Banner() string {
	if !ColorEnabled {
		return fmt.Sprintf("Shrike %s (%s) — %s\n", Version, Commit, tagline)
	}

	logo := lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render(logoEVE),
		lipgloss.NewStyle().Foreground(ColorAccent).Bold(true).Render(logoKILL),
	)

	meta := lipgloss.NewStyle().Foreground(ColorDim).Italic(true).
		Render(fmt.Sprintf(" %s · %s (%s)", tagline, Version, Commit))

	return logo + "\n" + meta + "\n"
}
