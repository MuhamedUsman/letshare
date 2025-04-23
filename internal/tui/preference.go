package tui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	lipTable "github.com/charmbracelet/lipgloss/table"
	"github.com/mattn/go-runewidth"
	"strconv"
	"strings"
)

type preferenceType int

const (
	option preferenceType = iota
	input
)

type preferenceSection int

const (
	sharing preferenceSection = iota
	receiving
)

var prefSecNames = []string{
	"Sharing",
	"Receiving",
}

func (ps preferenceSection) string() string {
	if int(ps) < 0 || int(ps) >= len(prefSecNames) {
		return "unknown preference section " + strconv.Itoa(int(ps))
	}
	return prefSecNames[ps]
}

type preferenceQue struct {
	title, desc              string
	pType                    preferenceType
	pSec                     preferenceSection
	prompt, input            string
	startsAtLine, endsAtLine int
	check                    bool
}

type preferenceInactiveMsg struct{}

// to signal extensionSpaceModel to switch back to the previous child model
// as the user is done with the preference model.
func preferenceInactiveCmd() tea.Msg { return preferenceInactiveMsg{} }

type preferenceModel struct {
	vp                                        viewport.Model
	txtInput                                  textinput.Model
	preferenceQues                            []preferenceQue
	titleStyle                                lipgloss.Style
	cursor, visibleFirstLine, visibleLastLine int
	unsaved, active, showHelp, disableKeymap  bool
}

func initialPreferenceModel() preferenceModel {
	preferenceQues := []preferenceQue{
		{
			title: "ZIP FILES?",
			desc:  "Share selected files as a single zip.",
			pType: option,
			pSec:  sharing,
		},
		{
			title: "ISOLATE FILES?",
			desc:  "Copy selected files to a separate directory before sharing.",
			pType: option,
			pSec:  sharing,
		},
		{
			title:  "SHARED ZIP NAME?",
			desc:   "Name of the archive selected files will be zipped into.",
			prompt: "Name",
			pType:  input,
			pSec:   receiving,
		},
		{
			title:  "DOWNLOAD FOLDER?",
			desc:   "Absolute path to a folder where files will be downloaded.",
			prompt: "Path",
			pType:  input,
			pSec:   receiving,
		},
		{
			title:  "SHARED ZIP NAME?",
			desc:   "Name of the archive selected files will be zipped into.",
			prompt: "Name",
			pType:  input,
			pSec:   receiving,
		},
		{
			title:  "DOWNLOAD FOLDER?",
			desc:   "Absolute path to a folder where files will be downloaded.",
			prompt: "Path",
			pType:  input,
			pSec:   receiving,
		},
		{
			title:  "SHARED ZIP NAME?",
			desc:   "Name of the archive selected files will be zipped into.",
			prompt: "Name",
			pType:  input,
			pSec:   receiving,
		},
		{
			title:  "DOWNLOAD FOLDER?",
			desc:   "Absolute path to a folder where files will be downloaded.",
			prompt: "Path",
			pType:  input,
			pSec:   receiving,
		},
		{
			title:  "SHARED ZIP NAME?",
			desc:   "Name of the archive selected files will be zipped into.",
			prompt: "Name",
			pType:  input,
			pSec:   receiving,
		},
		{
			title:  "DOWNLOAD FOLDER?",
			desc:   "Absolute path to a folder where files will be downloaded.",
			prompt: "Path",
			pType:  input,
			pSec:   receiving,
		},
		{
			title:  "SHARED ZIP NAME?",
			desc:   "Name of the archive selected files will be zipped into.",
			prompt: "Name",
			pType:  input,
			pSec:   receiving,
		},
		{
			title:  "DOWNLOAD FOLDER?",
			desc:   "Absolute path to a folder where files will be downloaded.",
			prompt: "Path",
			pType:  input,
			pSec:   receiving,
		},
		{
			title:  "SHARED ZIP NAME?",
			desc:   "Name of the archive selected files will be zipped into.",
			prompt: "Name",
			pType:  input,
			pSec:   receiving,
		},
		{
			title:  "DOWNLOAD FOLDER?",
			desc:   "Absolute path to a folder where files will be downloaded.",
			prompt: "Path",
			pType:  input,
			pSec:   receiving,
		},
	}
	vp := viewport.New(0, 0)
	vp.Style = vp.Style.PaddingTop(1)
	vp.MouseWheelEnabled = false
	vp.KeyMap = viewport.KeyMap{} // disable default keymap
	return preferenceModel{
		preferenceQues: preferenceQues,
		vp:             vp,
	}
}

func (m preferenceModel) capturesKeyEvent(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "tab", "down", "enter", "shift+tab", "up", "left", "right", "esc", "?":
		return !m.disableKeymap && m.active
	default:
		return false
	}
}

func (m preferenceModel) Init() tea.Cmd {
	return m.vp.Init()
}

func (m preferenceModel) Update(msg tea.Msg) (preferenceModel, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.updateViewPortDimensions()
		m.renderViewport()
		if m.visibleLastLine == 0 {
			m.visibleLastLine = m.vp.VisibleLineCount()
		}

	case tea.KeyMsg:
		if m.disableKeymap || !m.active {
			return m, nil
		}
		switch msg.String() {

		case "tab", "enter":
			m.cursor = (m.cursor + 1) % len(m.preferenceQues)
			m.handleViewportScroll(down)
			m.renderViewport()

		case "down":
			if m.cursor < len(m.preferenceQues)-1 {
				m.cursor++
			}
			m.handleViewportScroll(down)
			m.renderViewport()

		case "shift+tab":
			m.cursor = (m.cursor - 1 + len(m.preferenceQues)) % len(m.preferenceQues)
			m.handleViewportScroll(up)
			m.renderViewport()

		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
			m.handleViewportScroll(up)
			m.renderViewport()

		case "left":
			m.preferenceQues[m.cursor].check = false
			m.renderViewport()

		case "right":
			m.preferenceQues[m.cursor].check = true
			m.renderViewport()

		case "esc":
			if m.unsaved {
				return m, m.confirmDiscardChanges()
			}
			return m, m.inactivePreference()

		case "?":
			m.showHelp = !m.showHelp
			m.updateViewPortDimensions()

		}

	case spaceFocusSwitchMsg:
		m.updateTitleStyleAsFocus(currentFocus == extension)

	case extendChildMsg:
		m.active = msg.child == preference

	}

	return m, m.handleViewportUpdate(msg)
}

func (m preferenceModel) View() string {
	title := m.renderTitle("Preferences")
	status := m.renderStatusBar()
	help := customPreferenceHelp(m.showHelp)
	help.Width(largeContainerW())
	c := lipgloss.JoinVertical(lipgloss.Center, title, status, m.vp.View(), help.Render())
	return c
}

func (m *preferenceModel) handleViewportUpdate(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return cmd
}

type scrollDirection = int

const (
	up scrollDirection = iota
	down
)

func (m *preferenceModel) handleViewportScroll(direction scrollDirection) {
	if m.cursor == 0 {
		m.vp.GotoTop()
		return
	}
	if m.cursor == len(m.preferenceQues)-1 {
		m.vp.GotoBottom()
		return
	}
	switch direction {
	case up:
		visibleTopLine := m.vp.YOffset
		queStartingLine := m.preferenceQues[m.cursor].startsAtLine
		// question starting line is before the visible top line
		if queStartingLine < visibleTopLine {
			m.vp.SetYOffset(queStartingLine)
		}
	case down:
		visibleBottomLine := m.vp.YOffset + m.vp.VisibleLineCount()
		queEndingLine := m.preferenceQues[m.cursor].endsAtLine
		// question ending line is after the visible bottom line
		if queEndingLine > visibleBottomLine {
			m.vp.SetYOffset(queEndingLine - m.vp.VisibleLineCount())
		}
	}
}

func (m *preferenceModel) updateViewPortDimensions() {
	statusBarH := extStatusBarStyle.GetHeight() + extStatusBarStyle.GetVerticalFrameSize()
	helpH := lipgloss.Height(customPreferenceHelp(m.showHelp).String())
	viewportFrameH := m.vp.Style.GetVerticalFrameSize()
	h := extContainerWorkableH() - (statusBarH + helpH + viewportFrameH)
	w := largeContainerW() - largeContainerStyle.GetHorizontalFrameSize()
	m.vp.Width, m.vp.Height = w, h
	// set the cursor to 0
	m.cursor = 0
	m.visibleFirstLine = 0
	m.visibleLastLine = 0
	m.vp.GotoTop()
}

func (m *preferenceModel) updateTitleStyleAsFocus(focus bool) {
	if focus {
		m.titleStyle = titleStyle.
			UnsetMarginBottom().
			Background(highlightColor).
			Foreground(subduedHighlightColor)
	} else {
		m.titleStyle = titleStyle.
			UnsetMarginBottom().
			Background(subduedGrayColor).
			Foreground(highlightColor)
	}
}

func (m *preferenceModel) renderViewport() {
	sb := new(strings.Builder)
	prevSec := preferenceSection(-1)
	var startsAtLine, endsAtLine int
	for i, q := range m.preferenceQues {
		if q.pSec != prevSec {
			prevSec = q.pSec
			sb.WriteString(m.renderSectionTitle(q.pSec.string()))
			sb.WriteString("\n")
		}
		if i == m.cursor {
			sb.WriteString(m.renderActiveQue(q))
		} else {
			sb.WriteString(m.renderInactiveQue(q))
		}
		sb.WriteString("\n")
		endsAtLine = lipgloss.Height(sb.String())
		m.preferenceQues[i].startsAtLine = startsAtLine - preferenceQueContainerStyle.GetBorderTopSize()
		m.preferenceQues[i].endsAtLine = endsAtLine - preferenceQueContainerStyle.GetBorderBottomSize()
		startsAtLine = endsAtLine
	}
	m.vp.SetContent(sb.String())
}

func (m preferenceModel) renderTitle(title string) string {
	tail := "…"
	w := largeContainerW() - (lipgloss.Width(tail) + titleStyle.GetHorizontalPadding() + lipgloss.Width(tail))
	title = runewidth.Truncate(title, w, tail)
	return m.titleStyle.Render(title)
}

func (m preferenceModel) renderStatusBar() string {
	scrolled := m.vp.ScrollPercent() * 100
	s := fmt.Sprintf("Scrolled: %06.2f%%", scrolled)
	return extStatusBarStyle.Render(s)
}

func (m preferenceModel) renderSectionTitle(t string) string {
	return preferenceSectionStyle.
		Width(m.vp.Width - preferenceSectionStyle.GetHorizontalBorderSize()).
		Render(t)
}

func (m preferenceModel) renderInactiveQue(q preferenceQue) string {
	title := truncateRenderedTitle(q.title)
	title = preferenceQueTitleStyle.Render(title)
	descS := preferenceQueDescStyle
	var answerField string
	if q.pType == option {
		answerField = renderInactiveBtn(q.check)
	}
	if q.pType == input {
		answerField = renderInactiveInputField(q.prompt, q.input)
	}
	ques := lipgloss.JoinVertical(lipgloss.Left, title, descS.Render(q.desc), answerField)
	return preferenceQueContainerStyle.
		Width(m.vp.Width - preferenceQueContainerStyle.GetHorizontalBorderSize()).
		Render(ques)
}

func (m preferenceModel) renderActiveQue(q preferenceQue) string {
	title := truncateRenderedTitle(q.title)
	title = preferenceQueTitleStyle.
		Background(highlightColor).
		Foreground(subduedHighlightColor).
		Faint(true).
		Render(title)
	desc := preferenceQueDescStyle.
		Foreground(highlightColor).
		Render(q.desc)
	var answerField string
	if q.pType == option {
		answerField = renderActiveBtns(q.check)
	}
	if q.pType == input {
		answerField = renderInactiveInputField(q.prompt, q.input)
	}
	ques := lipgloss.JoinVertical(lipgloss.Left, title, desc, answerField)
	return preferenceQueContainerStyle.
		BorderForeground(highlightColor).
		Width(m.vp.Width - preferenceQueContainerStyle.GetHorizontalBorderSize()).
		Render(ques)
}

func renderInactiveInputField(prompt, placeholder string) string {
	return fmt.Sprintf("%s: %s", prompt, placeholder)
}

func renderInactiveBtn(check bool) string {
	s := "NOPE"
	if check {
		s = "YUP!"
	}
	return preferenceQueBtnStyle.
		Background(highlightColor).
		Foreground(subduedHighlightColor).
		Faint(true).
		Render(s)
}

func renderActiveBtns(check bool) string {
	btn1 := preferenceQueBtnStyle  // nope
	btn2 := preferenceQueBtnStyle. // yup!
					Background(highlightColor).
					Foreground(subduedHighlightColor).
					Faint(true)
	if !check { // btn1(nope) should be active
		btn1, btn2 = btn2, btn1
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, btn1.Render("NOPE"), btn2.Render("YUP!"))
}

func (m *preferenceModel) updateKeymap(disable bool) {
	m.disableKeymap = disable
}

func (m *preferenceModel) inactivePreference() tea.Cmd {
	m.cursor = 0
	m.visibleFirstLine, m.visibleLastLine = 0, 0
	m.active = false
	return preferenceInactiveCmd
}

// grant the discard request and envelops "esc" as a command for yupFunc
func (m *preferenceModel) confirmDiscardChanges() tea.Cmd {
	selBtn := yup
	header := "DISCARD CHANGES?"
	body := "Unsaved preferences will be lost."
	yupFunc := func() tea.Cmd {
		m.resetToSavedState()
		return m.inactivePreference()
	}
	return confirmDialogCmd(header, body, selBtn, yupFunc, nil)
}

func (m *preferenceModel) resetToSavedState() {

}

func (m preferenceModel) getLastVisibleLine() int {
	return int(m.vp.ScrollPercent()*float64(m.vp.TotalLineCount())) + (m.vp.VisibleLineCount())
}

func (m preferenceModel) getFirstVisibleLine() int {
	return int(m.vp.ScrollPercent() * float64(m.vp.TotalLineCount()))
}

func truncateRenderedTitle(title string) string {
	subW := largeContainerStyle.GetHorizontalFrameSize() +
		preferenceQueContainerStyle.GetHorizontalFrameSize() +
		preferenceQueTitleStyle.GetHorizontalFrameSize()
	titleW := largeContainerW() - subW
	return runewidth.Truncate(title, titleW, "…")
}

func customPreferenceHelp(show bool) *lipTable.Table {
	baseStyle := lipgloss.NewStyle().Margin(0, 2)
	var rows [][]string
	if !show {
		rows = [][]string{{"?", "help"}}
	} else {
		rows = [][]string{
			{"shift+↓/ctrl+↓", "make selection"},
			{"shift+↑/ctrl+↑", "undo selection"},
			{"enter", "select/deselect at cursor"},
			{"enter (when filtering)", "apply filter"},
			{"ctrl+a", "select all"},
			{"ctrl+z", "deselect all"},
			{"/", "filter"},
			{"esc", "exit filtering"},
			{"b/pgup", "page up"},
			{"f/child", "page down"},
			{"g/home", "go to start"},
			{"G/end", "go to end"},
			{"?", "hide help"},
		}
	}
	return lipTable.New().
		Border(lipgloss.HiddenBorder()).
		BorderBottom(false).
		Wrap(false).
		StyleFunc(func(row, col int) lipgloss.Style {
			switch col {
			case 0:
				return baseStyle.Foreground(highlightColor).Align(lipgloss.Left).Faint(true) // key style
			case 1:
				return baseStyle.Foreground(subduedHighlightColor).Align(lipgloss.Right) // desc style
			default:
				return baseStyle
			}
		}).Rows(rows...)
}
