package tui

import "github.com/charmbracelet/lipgloss"

var ( // Global Styling

	// These will be updated by any of the activeTab TabContainerModel
	terminalWidth  int
	terminalHeight int
	terminalFocus  *bool // only read the msg once the terminal in focus

	primaryColor           = lipgloss.AdaptiveColor{Light: "#4b3b00", Dark: "#FFC700"}
	primarySubtleDarkColor = lipgloss.AdaptiveColor{Light: "#6c5300", Dark: "#8b7000"}
	primaryContrastColor   = lipgloss.AdaptiveColor{Light: "#FFC700", Dark: "#4b3b00"}
	dangerColor            = lipgloss.AdaptiveColor{Light: "#ff7b4e", Dark: "#FF5C00"}
	dangerDarkColor        = lipgloss.AdaptiveColor{Light: "#b65d3e", Dark: "#a34a00"}
	whiteColor             = lipgloss.AdaptiveColor{Light: "#202020", Dark: "#E5D6A8"}
	blackColor             = lipgloss.AdaptiveColor{Light: "#E5D6A8", Dark: "#202020"}
	darkGreyColor          = lipgloss.AdaptiveColor{Light: "#808080", Dark: "#404040"}
	lightGreyColor         = lipgloss.AdaptiveColor{Light: "#404040", Dark: "#afafaf"}
	redColor               = lipgloss.AdaptiveColor{Light: "#FF0000", Dark: "#FF0000"}
	orangeColor            = lipgloss.AdaptiveColor{Light: "#ffa000", Dark: "#ffa000"}
	greenColor             = lipgloss.AdaptiveColor{Light: "#00a300", Dark: "#00ff00"}

	letschatLogo = lipgloss.NewStyle().
			Border(lipgloss.InnerHalfBlockBorder(), true).
			BorderForeground(primaryColor).
			Background(primaryColor).
			Foreground(primaryContrastColor).
			Width(10).
			MarginBottom(2).
			Align(lipgloss.Center).
			Italic(true).
			Render("Letschat")
)
