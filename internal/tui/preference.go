package tui

import (
	"fmt"
	"github.com/MuhamedUsman/letshare/internal/tui/overlay"
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
	vp                      viewport.Model
	txtInput                textinput.Model
	preferenceQues          []preferenceQue
	titleStyle              lipgloss.Style
	cursor                  int
	showHelp, disableKeymap bool
	// modes
	unsaved, active bool
	insertMode      bool
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
			prompt: "Name: ",
			pType:  input,
			pSec:   receiving,
			input:  "Shared-by-Usman",
		},
		{
			title:  "DOWNLOAD FOLDER?",
			desc:   "Absolute path to a folder where files will be downloaded.",
			prompt: "Path: ",
			pType:  input,
			pSec:   receiving,
			input:  "Absolute path to a folder where files will be downloaded.",
		},
	}

	vp := viewport.New(0, 0)
	vp.Style = vp.Style.PaddingTop(1)
	vp.MouseWheelEnabled = false
	vp.KeyMap = viewport.KeyMap{} // disable keymap

	txtInput := textinput.New()
	txtInput.Prompt = ""
	txtInput.ShowSuggestions = true
	txtInput.PromptStyle = txtInput.PromptStyle.Foreground(midHighlightColor)
	txtInput.TextStyle = txtInput.TextStyle.Foreground(highlightColor)
	txtInput.Cursor.Style = txtInput.Cursor.Style.Foreground(highlightColor)
	txtInput.PlaceholderStyle = txtInput.PlaceholderStyle.Foreground(subduedHighlightColor)

	return preferenceModel{
		preferenceQues: preferenceQues,
		vp:             vp,
		txtInput:       txtInput,
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
	return tea.Batch(m.vp.Init(), textinput.Blink)
}

func (m preferenceModel) Update(msg tea.Msg) (preferenceModel, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.updateDimensions()
		m.renderViewport()

	case tea.KeyMsg:
		if m.disableKeymap || !m.active {
			return m, nil
		}

		if m.insertMode {
			switch msg.String() {
			case "enter":
				m.preferenceQues[m.cursor].input = m.txtInput.Value()
				m.resetInsertMode()
				m.insertMode = false

			case "esc":
				m.insertMode = false
				m.resetInsertMode()
				return m, m.handleUpdate(msg)
			}
		}

		// No other keymap, until input is escaped
		if m.insertMode {
			return m, m.handleUpdate(msg)
		}

		switch msg.String() {
		case "tab", "enter":
			m.cursor = (m.cursor + 1) % len(m.preferenceQues)
			m.handleViewportScroll(down)

		case "down":
			if m.cursor < len(m.preferenceQues)-1 {
				m.cursor++
			}
			m.handleViewportScroll(down)

		case "shift+tab":
			m.cursor = (m.cursor - 1 + len(m.preferenceQues)) % len(m.preferenceQues)
			m.handleViewportScroll(up)

		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
			m.handleViewportScroll(up)

		case "left":
			m.preferenceQues[m.cursor].check = false

		case "right":
			m.preferenceQues[m.cursor].check = true

		case "i":
			return m, m.activateInsertMode()

		case "esc":
			if m.unsaved {
				return m, m.confirmDiscardChanges()
			} else {
				return m, m.inactivePreference()
			}

		case "?":
			m.showHelp = !m.showHelp
			m.updateDimensions()

		}
		m.renderViewport()

	case spaceFocusSwitchMsg:
		m.updateTitleStyleAsFocus(currentFocus == extension)

	case extendChildMsg:
		m.active = msg.child == preference

	}

	return m, tea.Batch(m.handleUpdate(msg))
}

func (m preferenceModel) View() string {
	title := m.renderTitle("Preferences")
	status := m.renderStatusBar()
	help := customPreferenceHelp(m.showHelp)
	help.Width(largeContainerW())
	if m.insertMode {
		o := m.renderInsertInputOverlay(m.preferenceQues[m.cursor].title, m.txtInput.View(), m.txtInput.Width)
		o = overlay.Place(lipgloss.Center, lipgloss.Center, m.vp.View(), o)
		return lipgloss.JoinVertical(lipgloss.Center, title, status, o, help.Render())
	}
	return lipgloss.JoinVertical(lipgloss.Center, title, status, m.vp.View(), help.Render())
}

func (m *preferenceModel) handleUpdate(msg tea.Msg) tea.Cmd {
	var cmds [2]tea.Cmd
	m.vp, cmds[0] = m.vp.Update(msg)
	m.txtInput, cmds[1] = m.txtInput.Update(msg)
	return tea.Batch(cmds[:]...)
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

func (m *preferenceModel) updateDimensions() {
	statusBarH := extStatusBarStyle.GetHeight() + extStatusBarStyle.GetVerticalFrameSize()
	helpH := lipgloss.Height(customPreferenceHelp(m.showHelp).String())
	viewportFrameH := m.vp.Style.GetVerticalFrameSize()
	h := extContainerWorkableH() - (statusBarH + helpH + viewportFrameH)
	w := largeContainerW() - largeContainerStyle.GetHorizontalFrameSize()
	m.vp.Width, m.vp.Height = w, h
	w = 50
	if m.vp.Width < w {
		w = m.vp.Width

	}
	m.txtInput.Width = w - preferenceQueOverlayContainerStyle.GetHorizontalFrameSize() - 4
	// set the cursor to 0
	if !m.insertMode {
		m.cursor = 0
		m.vp.GotoTop()
	}
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
	s := fmt.Sprintf("Cursor at %d/%d", m.cursor+1, len(m.preferenceQues))
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
		inputTitle := preferenceQueInputPromptStyle.Render(q.prompt)
		inputTxt := preferenceQueInputAnsStyle.Render(q.input)
		answerField = renderInactiveInputField(inputTitle, inputTxt)
	}
	ques := lipgloss.JoinVertical(lipgloss.Left, title, descS.Render(q.desc), answerField)
	return preferenceQueContainerStyle.
		Width(m.vp.Width - preferenceQueContainerStyle.GetHorizontalBorderSize()).
		Render(ques)
}

func (m *preferenceModel) renderActiveQue(q preferenceQue) string {
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
		inputTitle := preferenceQueInputPromptStyle.Render(q.prompt)
		inputTxt := preferenceQueInputAnsStyle.
			Underline(true).
			Italic(true).
			Render(q.input)
		answerField = renderInactiveInputField(inputTitle, inputTxt)
	}
	ques := lipgloss.JoinVertical(lipgloss.Left, title, desc, answerField)
	return preferenceQueContainerStyle.
		BorderForeground(highlightColor).
		Width(m.vp.Width - preferenceQueContainerStyle.GetHorizontalBorderSize()).
		Render(ques)
}

func renderInactiveInputField(prompt, placeholder string) string {
	return fmt.Sprintf("%s%s", prompt, placeholder)
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

func (m preferenceModel) renderInsertInputOverlay(title, inputView string, width int) string {
	topLeft, topRight := "╭─", "─╮"
	//topLeft, topRight := "╔═", "═╗"
	tail := "…"
	widthWithFrame := width + preferenceQueOverlayContainerStyle.GetHorizontalFrameSize()
	topBorderCornerW := lipgloss.Width(topLeft + topRight)
	// widthWithFrame has horizonal border included we need to subtract them, because we have our custom top borders
	subW := lipgloss.Width(tail) + (topBorderCornerW - preferenceQueOverlayContainerStyle.GetHorizontalBorderSize())

	title = runewidth.Truncate(title, widthWithFrame-subW, tail)
	title = preferenceQueTitleStyle.
		Background(highlightColor).
		Foreground(subduedHighlightColor).
		Faint(true).
		Render(title)

	titleW := lipgloss.Width(title)
	reqTopBorderW := widthWithFrame - 1 // -1 Experimental

	var padAfterTitle string
	if titleW < reqTopBorderW {
		n := reqTopBorderW - titleW
		padAfterTitle = strings.Repeat("─", n)
		//padAfterTitle = strings.Repeat("═", n)
	}

	borderStyle := lipgloss.NewStyle().Foreground(highlightColor)
	borderBeforeTitle := borderStyle.Render(topLeft)
	borderAfterTitle := borderStyle.Render(padAfterTitle + topRight)
	borderTop := borderStyle.
		MarginTop(1).
		MarginLeft(preferenceQueOverlayContainerStyle.GetMarginLeft()).
		MarginRight(preferenceQueOverlayContainerStyle.GetMarginRight()).
		Render(fmt.Sprintf("%s%s%s", borderBeforeTitle, title, borderAfterTitle))

	body := preferenceQueOverlayContainerStyle.BorderStyle(lipgloss.RoundedBorder()).Width(width + 9).Render(inputView)
	//body := preferenceQueOverlayContainerStyle.BorderStyle(lipgloss.DoubleBorder()).Width(width + 9).Render(inputView)
	return lipgloss.JoinVertical(lipgloss.Top, borderTop, body)
}

func (m *preferenceModel) updateKeymap(disable bool) {
	m.disableKeymap = disable
}

func (m *preferenceModel) activateInsertMode() tea.Cmd {
	m.insertMode = m.preferenceQues[m.cursor].pType == input
	m.txtInput.Prompt = m.preferenceQues[m.cursor].prompt
	m.txtInput.SetValue(m.preferenceQues[m.cursor].input)
	return m.txtInput.Focus()
}

func (m preferenceModel) resetInsertMode() {
	m.txtInput.Prompt = ""
	m.txtInput.SetValue("")
	m.insertMode = false
	m.txtInput.Blur()
}

func (m *preferenceModel) inactivePreference() tea.Cmd {
	m.cursor = 0
	m.active, m.showHelp = false, false
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
			{"tab/shift+tab", "move cursor (looped)"},
			{"↓/↑", "move cursor"},
			{"←/→", "switch option"},
			{"i", "insert/edit input"},
			{"enter", "apply inserted input"},
			{"esc", "exit insert/preference"},
			{"ctrl+s", "save changes"},
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
