package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type confirmDialogMsg struct {
	header, body string
}

func (m confirmDialogMsg) cmd() tea.Msg { return m }

type confirmDialogRespMsg bool

func (msg confirmDialogRespMsg) cmd() tea.Msg { return msg }

type confirmDialogModel struct {
	// header and body of the dialog box
	header, body string
	// cursor indicates current active button
	cursor int // 0: NOPE, 1: YUP!
	// render signals this model's view must be rendered
	render bool
	// prevFocus remembers the previous focus of the tab
	// and releases it accordingly
	prevFocus focusedTab
}

func initialConfirmDialogModel() confirmDialogModel {
	return confirmDialogModel{cursor: 1}
}

func (m confirmDialogModel) Init() tea.Cmd {
	return nil
}

func (m confirmDialogModel) Update(msg tea.Msg) (confirmDialogModel, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {

		case "enter":
			resp := false
			if m.cursor == 1 {
				resp = true
			}
			m.render = false
			m.header, m.body = "", ""
			currentFocus = m.prevFocus
			return m, confirmDialogRespMsg(resp).cmd

		case "tab":
			m.cursor = (m.cursor + 1) % 2

		case "shift+tab":
			m.cursor = (m.cursor - 1 + 2) % 2

		case "left", "h":
			m.cursor = 0

		case "right", "l":
			m.cursor = 1

		}

	case confirmDialogMsg:
		m.header = msg.header
		m.body = msg.body
		m.render = true
		currentFocus = confirmation
	}

	return m, nil
}

func (m confirmDialogModel) View() string {
	c := confirmDialogContainerStyle.Width(m.getDialogWidth())
	h := confirmDialogHeaderStyle.Render(m.header)
	b := confirmDialogBodyStyle.Render(m.body)
	nope := confirmDialogBtnStyle // inactive
	yup := confirmDialogBtnStyle. //active
					Background(highlightColor).
					Foreground(subduedHighlightColor).
					Faint(true)

	if m.cursor == 0 {
		nope = yup
		yup = confirmDialogBtnStyle
	}

	btns := lipgloss.JoinHorizontal(lipgloss.Top, nope.Render("NOPE"), yup.Render("YUP!"))
	btns = lipgloss.PlaceHorizontal(c.GetWidth()-confirmDialogBtnStyle.GetHorizontalPadding(), lipgloss.Right, btns)
	content := lipgloss.JoinVertical(lipgloss.Left, h, b, btns)
	return c.Render(content)
}

func (m confirmDialogModel) getDialogWidth() int {
	w := 50
	// condition to make the dialog centered
	if !isEven(largeContainerW()) {
		w -= 1
	}
	availableW := termW - mainContainerStyle.GetHorizontalFrameSize()
	if availableW <= w {
		w = availableW
	}
	return w
}

func isEven(n int) bool {
	return n%2 == 0
}
