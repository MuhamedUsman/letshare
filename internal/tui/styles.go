package tui

import (
	"github.com/MuhamedUsman/letshare/internal/tui/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
)

var (
	bgColor               = lipgloss.AdaptiveColor{Light: "#fbf1c7", Dark: "#282828"}
	fgColor               = lipgloss.AdaptiveColor{Light: "#282828", Dark: "#fbf1c7"}
	redColor              = lipgloss.AdaptiveColor{Light: "#9d0006", Dark: "#fb4934"}
	yellowColor           = lipgloss.AdaptiveColor{Light: "#b57614", Dark: "#fabd2f"}
	highlightColor        = lipgloss.AdaptiveColor{Light: "#4e562a", Dark: "#ECFD65"}
	midHighlightColor     = lipgloss.AdaptiveColor{Light: "#9DA947", Dark: "#9DA947"}
	subduedHighlightColor = lipgloss.AdaptiveColor{Light: "#ECFD65", Dark: "#4e562a"}
	grayColor             = lipgloss.AdaptiveColor{Light: "#999999", Dark: "#444444"}

	generateGradient = func(base, target lipgloss.AdaptiveColor, steps int) []lipgloss.AdaptiveColor {
		bLight, _ := colorful.Hex(base.Light)
		bDark, _ := colorful.Hex(base.Dark)
		tLight, _ := colorful.Hex(target.Light)
		tDark, _ := colorful.Hex(target.Dark)
		gradient := make([]lipgloss.AdaptiveColor, steps)

		// Generate the gradient colors
		for i := range steps {
			factor := float64(i) / float64(steps)
			lighter := bLight.BlendLuv(tLight, factor)
			darker := bDark.BlendLuv(tDark, factor)
			gradient[i] = lipgloss.AdaptiveColor{
				Light: lighter.Hex(),
				Dark:  darker.Hex(),
			}
		}
		return gradient
	}
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
		w := (workableW() * 26) / 100
		w -= smallContainerStyle.GetHorizontalFrameSize()
		return max(0, w)
	}

	largeContainerW = func() int {
		w := workableW() - smallContainerW()*2
		w -= largeContainerStyle.GetHorizontalFrameSize()
		return max(0, w)
	}

	extContainerWorkableH = func() int {
		return workableH() - largeContainerStyle.GetVerticalFrameSize()
	}
)

var ( // Common Styles

	titleStyle = lipgloss.NewStyle().
			Background(grayColor).
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

var ( // processFilesModel Styles
	zipLogsContainerStyle = lipgloss.NewStyle().
		Padding(1, 0)
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

var ( // receiveModel Styles
	receiveInstanceInputContainerStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(subduedHighlightColor).
		Foreground(midHighlightColor).
		Padding(0, 1).
		Align(lipgloss.Center).
		Italic(true)
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
				Background(grayColor).
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

var (
	downloadCardContainerStyle = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(subduedHighlightColor).
					Padding(0, 1)

	downloadCardProgressStyle = lipgloss.NewStyle().
					Background(highlightColor).
					Foreground(subduedHighlightColor).
					Faint(true).
					Padding(0, 1)
)

var ( // alertDialogModel Styles

	alertDialogContainerStyle = lipgloss.NewStyle().
					BorderStyle(lipgloss.RoundedBorder()).
					BorderForeground(highlightColor).
					Padding(1, 2)

	alertDialogHeaderStyle = lipgloss.NewStyle().
				Background(highlightColor).
				Foreground(subduedHighlightColor).
				Padding(0, 1).
				Faint(true)

	alertDialogBodyStyle = lipgloss.NewStyle().
				Italic(true).
				Padding(1, 0).
				Foreground(highlightColor)

	alertDialogBtnStyle = lipgloss.NewStyle().
				Background(grayColor).
				Foreground(fgColor).
				Padding(0, 2).
				MarginLeft(1)
)
