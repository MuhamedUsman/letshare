package tui

import (
	"github.com/MuhamedUsman/letshare/internal/tui/table"
	"github.com/charmbracelet/lipgloss"
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
		return (termW * 25) / 100
	}

	largeContainerW = func() int {
		return termW - (smallContainerW()*2 + mainContainerStyle.GetHorizontalFrameSize())
	}

	infoContainerWorkableH = func(includeTitle bool) int {
		h := mainContainerStyle.GetVerticalFrameSize() +
			infoContainerStyle.GetVerticalFrameSize()
		if includeTitle {
			h += lipgloss.Height(titleStyle.String())
		}
		return termH - h
	}
)

var ( // Common Styles

	titleStyle = lipgloss.NewStyle().
			Background(subduedGrayColor).
			Foreground(highlightColor).
			Italic(true).
			Padding(0, 1).
			MarginBottom(1)

	smallContainerStyle = lipgloss.NewStyle().
				Margin(1, 1, 0, 1)
)

var ( // mainModel Styles

	mainContainerStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), true).
		BorderForeground(highlightColor)
)

var ( // dirNavigationModel Styles

)

var ( // extensionSpaceModel Styles

	infoContainerStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, true).
				BorderForeground(subduedHighlightColor).
				AlignHorizontal(lipgloss.Center).
				Padding(1, 0)

	banner = lipgloss.NewStyle().
		Foreground(midHighlightColor).
		AlignVertical(lipgloss.Center).
		SetString(bannerTxt)

	slogan = lipgloss.NewStyle().
		Italic(true).
		Foreground(highlightColor).
		Faint(true).
		SetString("with Freedom!")

	bannerTxt = `
┬  ┌─┐┌┬┐┌─┐┬ ┬┌─┐┬─┐┌─┐
│  ├┤  │ └─┐├─┤├─┤├┬┘├┤ 
┴─┘└─┘ ┴ └─┘┴ ┴┴ ┴┴└─└─┘
           ` + slogan.Render()
)

var ( // extendDirModel Styles

	customTableStyles = table.Styles{
		Header: table.DefaultStyles().Header.
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			Foreground(highlightColor).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(subduedHighlightColor).
			BorderBottom(true),
		Selected: table.DefaultStyles().Selected.
			Background(subduedHighlightColor).
			Foreground(highlightColor).
			Italic(true),
		Cell: table.DefaultStyles().Cell.Foreground(midHighlightColor),
	}

	infoTableStatusBarStyle = lipgloss.NewStyle().
				Margin(1, 1, 0, 1).
				Italic(true).
				Foreground(highlightColor).
				Faint(true)

	infoTableFilterContainerStyle = lipgloss.NewStyle().
					Align(lipgloss.Center)
)

var ( // remoteSpaceModel Styles

)

var ( // confirmDialogModel Styles

	confirmDialogContainerStyle = lipgloss.NewStyle().
					BorderStyle(lipgloss.RoundedBorder()).
					BorderForeground(highlightColor).
					Padding(1, 2)

	confirmDialogHeaderStyle = lipgloss.NewStyle().
					Background(highlightColor).
					Foreground(subduedHighlightColor).
					Padding(0, 1).
					Faint(true)

	confirmDialogBodyStyle = lipgloss.NewStyle().
				Italic(true).
				Padding(1, 0).
				Foreground(highlightColor)

	confirmDialogBtnStyle = lipgloss.NewStyle().
				Background(subduedGrayColor).
				Foreground(fgColor).
				Padding(0, 2).
				MarginLeft(1)
)
