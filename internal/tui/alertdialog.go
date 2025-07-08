package tui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"time"
)

// cursor indicates current active button
type alertCursor int

const (
	none alertCursor = iota - 1
	negative
	positive
)

type alertDialogMsg struct {
	header, body string
	// which btn to be active
	positiveBtnTxt, negativeBtnTxt string
	cursor                         alertCursor
	// alertDuration is used to set the timer for the alert dialog
	// it defaults to 5 seconds
	// takes effect only if positiveBtnTxt and negativeBtnTxt are nil
	alertDuration                       time.Duration
	positiveFunc, negativeFunc, escFunc func() tea.Cmd
}

type alertDialogModel struct {
	// header and body of the dialog box
	header, body string
	// buttons text
	positiveBtnTxt, negativeBtnTxt string
	cursor                         alertCursor
	timer                          timer.Model
	// prevFocus remembers the previous focused child
	// and releases it accordingly
	prevFocus focusSpace
	// active signals this model's view must be rendered
	active        bool
	disableKeymap bool
	// functions to all on appropriate buttons
	positiveFunc, negativeFunc, escFunc func() tea.Cmd
}

func initialAlertDialogModel() alertDialogModel {
	return alertDialogModel{
		cursor:        positive,
		timer:         timer.NewWithInterval(5*time.Second, 100*time.Millisecond),
		disableKeymap: true,
	}
}

func (m alertDialogModel) capturesKeyEvent(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "enter", "tab", "shift+tab", "left", "right", "h", "l", "esc":
		return m.active
	default:
		return false
	}
}

func (m alertDialogModel) Init() tea.Cmd {
	return nil
}

func (m alertDialogModel) Update(msg tea.Msg) (alertDialogModel, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		if currentFocus != alert {
			return m, nil
		}
		switch msg.String() {

		case "enter":
			var cmd tea.Cmd
			if m.cursor == 1 && m.positiveFunc != nil {
				cmd = m.positiveFunc()
			} else if m.negativeFunc != nil {
				cmd = m.negativeFunc()
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

		case "esc": // works same as pressing negative btn
			var cmd tea.Cmd
			if m.escFunc != nil {
				cmd = m.escFunc()
			}
			if m.isTimerAlert() {
				m.timer = timer.NewWithInterval(5*time.Second, 100*time.Millisecond)
			}
			return m, tea.Batch(cmd, m.hide())
		}

	case alertDialogMsg:
		m.header, m.body = msg.header, msg.body
		m.positiveBtnTxt = msg.positiveBtnTxt
		m.negativeBtnTxt = msg.negativeBtnTxt
		m.positiveFunc, m.negativeFunc, m.escFunc = msg.positiveFunc, msg.negativeFunc, msg.escFunc
		m.cursor = msg.cursor
		m.active = true
		if currentFocus != alert { // in-case multiple alert dialogs become active
			m.prevFocus = currentFocus
		}
		currentFocus = alert
		// in this case, the dialog is just a simple alert, then start timer
		if m.isTimerAlert() {
			d := 5 * time.Second // default
			if msg.alertDuration > 0 {
				d = msg.alertDuration
			}
			m.timer = timer.NewWithInterval(d, 100*time.Millisecond)
			return m, tea.Batch(m.timer.Init(), msgToCmd(spaceFocusSwitchMsg{}))
		}
		return m, msgToCmd(spaceFocusSwitchMsg{})

	case timer.TickMsg:
		if msg.ID == m.timer.ID() {
			var cmd tea.Cmd
			m.timer, cmd = m.timer.Update(msg)
			return m, cmd
		}

	case timer.TimeoutMsg:
		if msg.ID == m.timer.ID() {
			var cmd tea.Cmd
			if m.escFunc != nil {
				cmd = m.escFunc()
			}
			return m, tea.Batch(cmd, m.hide())
		}
	}

	return m, nil
}

func (m alertDialogModel) View() string {
	c := alertDialogContainerStyle.Width(m.getDialogWidth())
	h := alertDialogHeaderStyle.Render(m.header)
	b := alertDialogBodyStyle.Render(m.body)
	negStyle := alertDialogBtnStyle  // inactive
	posStyle := alertDialogBtnStyle. //active
						Background(highlightColor).
						Foreground(subduedHighlightColor).
						Faint(true)

	if m.cursor == 0 {
		negStyle = posStyle
		posStyle = alertDialogBtnStyle
	}

	var view string
	if !m.isTimerAlert() {
		btns := lipgloss.JoinHorizontal(lipgloss.Center, negStyle.Render(m.negativeBtnTxt), posStyle.Render(m.positiveBtnTxt))
		btns = lipgloss.PlaceHorizontal(c.GetWidth()-alertDialogBtnStyle.GetHorizontalPadding(), lipgloss.Right, btns)
		view = lipgloss.JoinVertical(lipgloss.Left, h, b, btns)
	} else {
		style := lipgloss.NewStyle().Inline(true).Foreground(subduedHighlightColor)
		view = style.Render("Escaping in: ")
		t := style.Foreground(midHighlightColor).Render(fmt.Sprintf("%.1f", m.timer.Timeout.Seconds()))
		view = lipgloss.JoinHorizontal(lipgloss.Center, view, t)
		view = lipgloss.PlaceHorizontal(c.GetWidth()-alertDialogBtnStyle.GetHorizontalPadding(), lipgloss.Right, view)
		view = lipgloss.JoinVertical(lipgloss.Left, h, b, view)
	}

	return c.Render(view)
}

func (m alertDialogModel) getDialogWidth() int {
	w := 50
	// condition to make the dialog centered
	if workableW() < w {
		w = workableW()
	}
	if !isEven(workableW()) {
		w -= 1
	}
	return w
}

func (m *alertDialogModel) updateKeymap(disable bool) {
	m.disableKeymap = disable
}

func (m *alertDialogModel) hide() tea.Cmd {
	m.active = false
	m.header, m.body = "", ""
	m.positiveFunc, m.negativeFunc = nil, nil
	currentFocus = m.prevFocus
	return msgToCmd(spaceFocusSwitchMsg{})
}

func (m alertDialogModel) isTimerAlert() bool {
	return m.positiveBtnTxt == "" && m.negativeBtnTxt == ""
}

func isEven(n int) bool {
	return n%2 == 0
}
