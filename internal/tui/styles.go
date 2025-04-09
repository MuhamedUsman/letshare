package tui

import (
	"github.com/charmbracelet/lipgloss"
	"math"
)

var ( // color scheme from https://github.com/morhetz/gruvbox

	bgColor               = lipgloss.AdaptiveColor{Light: "#fbf1c7", Dark: "#282828"}
	fgColor               = lipgloss.AdaptiveColor{Light: "#282828", Dark: "#fbf1c7"}
	redColor              = lipgloss.AdaptiveColor{Light: "#cc241d", Dark: "#cc241d"}
	redBrightColor        = lipgloss.AdaptiveColor{Light: "#9d0006", Dark: "#fb4934"}
	greenColor            = lipgloss.AdaptiveColor{Light: "#5f7a3d", Dark: "#D7FF87"}
	yellowColor           = lipgloss.AdaptiveColor{Light: "#b57614", Dark: "#fabd2f"}
	blueColor             = lipgloss.AdaptiveColor{Light: "#076678", Dark: "#458588"}
	purpleColor           = lipgloss.AdaptiveColor{Light: "#8f3f71", Dark: "#b16286"}
	highlightColor        = lipgloss.AdaptiveColor{Light: "#04B575", Dark: "#ECFD65"}
	midHighlightColor     = lipgloss.AdaptiveColor{Light: "#49D8A1", Dark: "#9DA947"}
	subduedHighlightColor = lipgloss.AdaptiveColor{Light: "#8EFBCD", Dark: "#4e562a"}
	grayColor             = lipgloss.AdaptiveColor{Light: "#7c6f64", Dark: "#928374"}
	subduedGrayColor      = lipgloss.AdaptiveColor{Light: "#999999", Dark: "#444444"}
	orangeColor           = lipgloss.AdaptiveColor{Light: "#af3a03", Dark: "#d65d0e"}
)

var ( // Container width calculations

	smallContainerW = func() int {
		w := float64(termW*25) / 100
		return int(math.RoundToEven(w))
	}

	largeContainerW = func() int {
		return termW - (smallContainerW()*2 + mainContainerStyle.GetHorizontalFrameSize())
	}
)

var ( // mainModel Styles

	mainContainerStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), true).
		BorderForeground(highlightColor)
)

var ( // sendModel Styles

	sendContainerStyle = lipgloss.NewStyle().
		Margin(1, 1, 0, 1)
)

var ( // infoModel Styles
	infoContainerStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, true).
		BorderForeground(subduedHighlightColor).
		AlignHorizontal(lipgloss.Center).
		Padding(1, 0)
)

var ( // sendInfoModel Styles
	tableBaseStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, false).
		BorderForeground(subduedHighlightColor)
)
