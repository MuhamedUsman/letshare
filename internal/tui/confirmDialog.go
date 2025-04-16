package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// cursor indicates current active button
type confirmationCursor = int

const (
	nope = iota
	yup
)

type confirmDialogMsg struct {
	header, body string
	// which btn to be active
	cursor            confirmationCursor
	yupFunc, nopeFunc func() tea.Cmd
}

func confirmDialogCmd(header, body string, cursor confirmationCursor, yupFunc, nopeFunc func() tea.Cmd) tea.Cmd {
	return func() tea.Msg {
		return confirmDialogMsg{
			header:   header,
			body:     body,
			cursor:   cursor,
			yupFunc:  yupFunc,
			nopeFunc: nopeFunc,
		}
	}
}

type confirmDialogModel struct {
	// header and body of the dialog box
	header, body string
	cursor       confirmationCursor
	// prevFocus remembers the previous focused space
	// and releases it accordingly
	prevFocus focusedTab
	// render signals this model's view must be rendered
	render bool
	// functions to all on appropriate buttons
	yupFunc, nopeFunc func() tea.Cmd
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
		if currentFocus != confirmation {
			return m, nil
		}
		switch msg.String() {

		case "enter":
			var cmd tea.Cmd
			if m.cursor == 1 {
				cmd = m.yupFunc()
			} else {
				cmd = m.nopeFunc()
			}
			return m, tea.Batch(m.hide(), cmd)

		case "tab":
			m.cursor = (m.cursor + 1) % 2

		case "shift+tab":
			m.cursor = (m.cursor - 1 + 2) % 2

		case "left", "h":
			m.cursor = 0

		case "right", "l":
			m.cursor = 1

		case "esc": // works same as pressing nope btn
			return m, tea.Batch(m.nopeFunc(), m.hide())
		}

	case confirmDialogMsg:
		m.header, m.body = msg.header, msg.body
		m.yupFunc, m.nopeFunc = msg.yupFunc, msg.nopeFunc
		m.cursor = msg.cursor
		m.render = true
		m.prevFocus = currentFocus
		currentFocus = confirmation
		return m, spaceFocusSwitchMsg(confirmation).cmd
	}

	return m, nil
}

func (m confirmDialogModel) View() string {
	c := confirmDialogContainerStyle.Width(m.getDialogWidth())
	h := confirmDialogHeaderStyle.Render(m.header)
	b := confirmDialogBodyStyle.Render(m.body)
	nopeStyle := confirmDialogBtnStyle // inactive
	yupStyle := confirmDialogBtnStyle. //active
						Background(highlightColor).
						Foreground(subduedHighlightColor).
						Faint(true)

	if m.cursor == 0 {
		nopeStyle = yupStyle
		yupStyle = confirmDialogBtnStyle
	}

	btns := lipgloss.JoinHorizontal(lipgloss.Top, nopeStyle.Render("NOPE"), yupStyle.Render("YUP!"))
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

func (m *confirmDialogModel) hide() tea.Cmd {
	m.render = false
	m.header, m.body = "", ""
	m.yupFunc, m.nopeFunc = nil, nil
	currentFocus = m.prevFocus
	return spaceFocusSwitchMsg(m.prevFocus).cmd
}

func isEven(n int) bool {
	return n%2 == 0
}
