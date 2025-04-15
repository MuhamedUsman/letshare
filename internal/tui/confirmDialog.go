package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"sync"
)

var (
	mu     sync.Mutex
	nextID uint
)

func getNextID() uint {
	mu.Lock()
	defer mu.Unlock()
	nextID++
	return nextID
}

type confirmDialogMsg struct {
	header, body string
	// which btn to be active
	cursor int // 0: NOPE, 1: YUP!
	// id uniquely identifies confirmDialogRespMsg
	// floating around in the bubble tea event loop
	id uint
}

func confirmDialogCmd(header, body string, cursor int, id uint) tea.Cmd {
	return func() tea.Msg {
		return confirmDialogMsg{
			header: header,
			body:   body,
			cursor: cursor,
			id:     id,
		}
	}
}

type confirmationResp int

const (
	esc = iota
	yup
	nope
)

type confirmDialogRespMsg struct {
	resp confirmationResp
	id   uint
}

func (msg confirmDialogRespMsg) cmd() tea.Msg { return msg }

type confirmDialogModel struct {
	// header and body of the dialog box
	header, body string
	// cursor indicates current active button
	cursor int // 0: NOPE, 1: YUP!
	// prevFocus remembers the previous focused space
	// and releases it accordingly
	prevFocus focusedTab
	// id uniquely identifies confirmDialogRespMsg
	// floating around in the bubble tea event loop
	id uint
	// render signals this model's view must be rendered
	render bool
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
			var resp confirmationResp = nope
			if m.cursor == 1 {
				resp = yup
			}
			m.render = false
			m.header, m.body = "", ""
			currentFocus = m.prevFocus
			return m, tea.Batch(confirmDialogRespMsg{resp: resp, id: m.id}.cmd, spaceFocusSwitchMsg(m.prevFocus).cmd)

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
		m.cursor = msg.cursor
		m.id = msg.id
		m.render = true
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

func isEven(n int) bool {
	return n%2 == 0
}
