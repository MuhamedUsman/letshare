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

	workableW = func() int {
		w := termW - mainContainerStyle.GetHorizontalFrameSize()
		return max(0, w)
	}

	workableH = func() int {
		h := termH - mainContainerStyle.GetVerticalFrameSize()
		return max(0, h)
	}

	smallContainerW = func() int {
		w := (workableW() * 25) / 100
		w = w - smallContainerStyle.GetHorizontalFrameSize()
		return max(0, w)
	}

	largeContainerW = func() int {
		w := workableW() - smallContainerW()*2
		w = w - largeContainerStyle.GetHorizontalFrameSize()
		return max(0, w)
	}

	extContainerWorkableH = func() int {
		return workableH() - largeContainerStyle.GetVerticalFrameSize()
	}
)

var ( // Common Styles

	titleStyle = lipgloss.NewStyle().
			Background(subduedGrayColor).
			Foreground(highlightColor).
			Italic(true).
			Height(1).
			Padding(0, 1)

	smallContainerStyle = lipgloss.NewStyle().
				Padding(1, 1, 0, 1)
)

var ( // mainModel Styles

	mainContainerStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), true).
		BorderForeground(highlightColor)
)

var ( // dirNavModel Styles

)

var ( // extensionSpaceModel Styles

	largeContainerStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, true).
				BorderForeground(subduedHighlightColor).
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

var ( // extensionModel Styles

	extStatusBarStyle = lipgloss.NewStyle().
		Margin(1, 1, 0, 1).
		Height(1).
		Italic(true).
		Foreground(highlightColor).
		Faint(true)
)

var ( // extDirNavModel Styles

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

	extDirNavTableFilterContainerStyle = lipgloss.NewStyle().
						Align(lipgloss.Center)
)

var ( // remoteSpaceModel Styles

)

var ( // prepSelModel Styles

	preferenceSectionStyle = lipgloss.NewStyle().
				BorderForeground(midHighlightColor).
				BorderStyle(lipgloss.ASCIIBorder()).
				Foreground(midHighlightColor).
				Align(lipgloss.Center)

	preferenceQueContainerStyle = lipgloss.NewStyle().
					BorderForeground(subduedHighlightColor).
					BorderStyle(lipgloss.RoundedBorder()).
					Padding(1, 2)

	preferenceQueTitleStyle = lipgloss.NewStyle().
				Background(subduedHighlightColor).
				Foreground(highlightColor).
				Italic(true).
				Padding(0, 1)

	preferenceQueDescStyle = lipgloss.NewStyle().
				Foreground(midHighlightColor).
				Padding(1, 0).
				Italic(true)

	preferenceQueBtnStyle = lipgloss.NewStyle().
				Background(subduedGrayColor).
				Foreground(fgColor).
				Padding(0, 1).
				MarginRight(1)

	preferenceQueInputPromptStyle = lipgloss.NewStyle().
					Foreground(highlightColor).
					Faint(true)

	preferenceQueInputAnsStyle = lipgloss.NewStyle().
					Foreground(highlightColor)

	preferenceQueOverlayContainerStyle = lipgloss.NewStyle().
						Border(lipgloss.RoundedBorder(), false, true, true, true).
						Padding(1).
						Margin(0, 2, 1, 2).
						BorderForeground(highlightColor)
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
