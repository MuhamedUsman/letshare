package overlay

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"math"
	"regexp"
	"strings"
)

// Overlay foreground on background
var ansiStyleRegexp = regexp.MustCompile(`\x1b[[\d;]*m`)

// Place places foreground content over background content at specified position
// width and height are the total dimensions of the target area
// hPos and vPos specify the horizontal and vertical positioning using lipgloss.Position
// bg is the background content to overlay on
// fg is the foreground content to overlay
func Place(width, height int, hPos, vPos lipgloss.Position, bg, fg string) string {
	// Parse the background and foreground content
	bgLines := strings.Split(bg, "\n")
	fgLines := strings.Split(fg, "\n")

	// Ensure background has enough lines
	for len(bgLines) < height {
		bgLines = append(bgLines, "")
	}

	// Trim or pad background lines to width
	for i, line := range bgLines {
		lineWidth := ansi.StringWidth(line)
		if lineWidth < width {
			bgLines[i] = line + strings.Repeat(" ", width-lineWidth)
		} else if lineWidth > width {
			bgLines[i] = ansi.Truncate(line, width, "")
		}
	}

	// Get dimensions of the foreground content
	fgWidth := 0
	for _, line := range fgLines {
		lineWidth := ansi.StringWidth(line)
		if lineWidth > fgWidth {
			fgWidth = lineWidth
		}
	}
	fgHeight := len(fgLines)

	// Calculate position based on lipgloss.Position values
	var hOffset int
	var vOffset int

	// Handle horizontal positioning using lipgloss's approach
	switch hPos {
	case lipgloss.Left:
		hOffset = 0
	case lipgloss.Right:
		hOffset = width - fgWidth
	default:
		// For Center and other values, interpret as a percentage (0.0-1.0)
		gap := width - fgWidth
		if gap > 0 {
			hOffset = int(math.Round(float64(gap) * float64(hPos)))
		}
	}

	// Handle vertical positioning using lipgloss's approach
	switch vPos {
	case lipgloss.Top:
		vOffset = 0
	case lipgloss.Bottom:
		vOffset = height - fgHeight
	default:
		// For Center and other values, interpret as a percentage (0.0-1.0)
		gap := height - fgHeight
		if gap > 0 {
			vOffset = int(math.Round(float64(gap) * float64(vPos)))
		}
	}

	// Ensure offsets are in bounds
	if hOffset < 0 {
		hOffset = 0
	}
	if vOffset < 0 {
		vOffset = 0
	}

	for i, fgLine := range fgLines {
		bgIdx := i + vOffset

		// Skip if outside bounds
		if bgIdx < 0 || bgIdx >= len(bgLines) {
			continue
		}

		bgLine := bgLines[bgIdx]

		// Add padding if needed
		if ansi.StringWidth(bgLine) < hOffset {
			bgLine += strings.Repeat(" ", hOffset-ansi.StringWidth(bgLine))
		}

		// Split the background line at the overlay position
		bgLeft := ansi.Truncate(bgLine, hOffset, "")
		bgRight := truncateLeft(bgLine, hOffset+ansi.StringWidth(fgLine))

		// Combine with the foreground line
		bgLines[bgIdx] = bgLeft + fgLine + bgRight
	}

	return strings.Join(bgLines, "\n")
}

// truncateLeft returns the portion of a string that would appear
// after the given width, preserving ANSI escape sequences
func truncateLeft(line string, padding int) string {
	if strings.Contains(line, "\n") {
		panic("line must not contain newline")
	}

	// Wrap to the padding width and split by newlines
	wrapped := strings.Split(ansi.Hardwrap(line, padding, true), "\n")
	if len(wrapped) == 1 {
		return ""
	}

	// Preserve any ANSI style from the beginning portion
	var ansiStyle string
	ansiStyles := ansiStyleRegexp.FindAllString(wrapped[0], -1)
	if l := len(ansiStyles); l > 0 {
		ansiStyle = ansiStyles[l-1]
	}

	return ansiStyle + strings.Join(wrapped[1:], "")
}
