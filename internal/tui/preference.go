package tui

import (
	"fmt"
	"github.com/MuhamedUsman/letshare/internal/config"
	"github.com/MuhamedUsman/letshare/internal/tui/overlay"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	lipTable "github.com/charmbracelet/lipgloss/table"
	"github.com/mattn/go-runewidth"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"
)

type preferenceType int

const (
	option preferenceType = iota
	input
)

type preferenceSection int

const (
	personal preferenceSection = iota
	share
	receive
)

var prefSecNames = []string{
	"Personal",
	"Sharing",
	"Receiving",
}

func (ps preferenceSection) string() string {
	if int(ps) < 0 || int(ps) >= len(prefSecNames) {
		return "unknown preference section " + strconv.Itoa(int(ps))
	}
	return prefSecNames[ps]
}

type preferenceKey int

const (
	username preferenceKey = iota
	instanceName
	stoppableInstance
	zipFiles
	compression
	sharedZipName
	downloadFolder
	concurrentDownloads
)

var prefKeyNames = []string{
	"USERNAME",
	"INSTANCE NAME",
	"STOPPABLE INSTANCE",
	"ZIP FILES?",
	"COMPRESSED ZIP?",
	"SHARED ZIP NAME",
	"DOWNLOAD FOLDER",
	"CONCURRENT DOWNLOADS",
}

func (pk preferenceKey) string() string {
	if int(pk) < 0 || int(pk) >= len(prefKeyNames) {
		return "unknown preference key " + strconv.Itoa(int(pk))
	}
	return prefKeyNames[pk]
}

type scrollDirection = int

const (
	up scrollDirection = iota
	down
)

type preferenceQue struct {
	title preferenceKey
	desc  string
	pType preferenceType
	pSec  preferenceSection
	// a pSec of an input type has two fields
	prompt, input            string
	startsAtLine, endsAtLine int
	// a pSec of an option type has this check
	check bool // true -> positive!, false -> negative
}

type preferenceModel struct {
	vp             viewport.Model
	txtInput       textinput.Model
	preferenceQues []preferenceQue
	// used to check unsaved state
	titleStyle                          lipgloss.Style
	cursor                              int
	showHelp, disableKeymap, insertMode bool
}

func initialPreferenceModel() preferenceModel {
	cfg, err := config.Load()
	if err != nil {
		println()
		slog.Error("unable to load config:", "error", err)
		os.Exit(1)
	}
	ques := populatePreferencesFromConfig(cfg)

	vp := viewport.New(0, 0)
	vp.Style = vp.Style.Padding(1, 1, 0, 1)
	vp.MouseWheelEnabled = false
	vp.KeyMap = viewport.KeyMap{} // disable keymap

	txtInput := textinput.New()
	txtInput.Prompt = ""
	txtInput.ShowSuggestions = true
	txtInput.PromptStyle = txtInput.PromptStyle.Foreground(midHighlightColor)
	txtInput.TextStyle = txtInput.TextStyle.Foreground(highlightColor)
	txtInput.Cursor.TextStyle = txtInput.Cursor.Style.Foreground(highlightColor)
	txtInput.Cursor.Style = txtInput.Cursor.TextStyle.Reverse(true)
	txtInput.PlaceholderStyle = txtInput.PlaceholderStyle.Foreground(subduedHighlightColor)

	return preferenceModel{
		preferenceQues: ques,
		vp:             vp,
		txtInput:       txtInput,
		disableKeymap:  true,
	}
}

func (m preferenceModel) capturesKeyEvent(msg tea.KeyMsg) bool {
	return msg.String() != "ctrl+c"
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
		if m.disableKeymap {
			return m, nil
		}

		if m.insertMode {
			switch msg.String() {
			case "enter":
				if isValid, txt := m.validateInput(m.txtInput.Value()); !isValid {
					return m, m.showInvalidInputAlert(txt)
				}
				m.preferenceQues[m.cursor].input = m.txtInput.Value()
				m.renderViewport()
				m.resetInsertMode()
				m.insertMode = false

			case "esc":
				m.insertMode = false
				m.resetInsertMode()
			}
			return m, m.handleUpdate(msg)
		}

		// No other keymap, until input is escaped
		if m.insertMode {
			return m, m.handleUpdate(msg)
		}

		switch msg.String() {
		case "tab", "enter":
			m.cursor = (m.cursor + 1) % len(m.preferenceQues)
			m.handleViewportScroll(down)

		case "down", "j":
			if m.cursor < len(m.preferenceQues)-1 {
				m.cursor++
			}
			m.handleViewportScroll(down)

		case "shift+tab":
			m.cursor = (m.cursor - 1 + len(m.preferenceQues)) % len(m.preferenceQues)
			m.handleViewportScroll(up)

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			m.handleViewportScroll(up)

		case "left", "h":
			m.preferenceQues[m.cursor].check = false

		case "right", "l":
			m.preferenceQues[m.cursor].check = true

		case "i":
			return m, m.activateInsertMode()

		case "ctrl+s":
			if m.isUnsavedState() {
				return m, m.savePreferences(false)
			}

		case "esc":
			if m.isUnsavedState() {
				return m, m.confirmSaveChanges()
			} else {
				return m, tea.Batch(m.inactivePreference(), m.handleUpdate(msg))
			}

		case "?":
			m.showHelp = !m.showHelp
			m.updateDimensions()

		}
		m.renderViewport()

	case spaceFocusSwitchMsg:
		m.updateTitleStyleAsFocus()

	case preferencesSavedMsg:
		if msg {
			return m, tea.Batch(m.inactivePreference(), m.handleUpdate(msg))
		}

	case rerenderPreferencesMsg:
		m.renderViewport()

	}

	return m, m.handleUpdate(msg)
}

func (m preferenceModel) View() string {
	title := m.renderTitle("Preferences")
	status := m.renderStatusBar()
	help := customPreferenceHelp(m.showHelp)
	help.Width(largeContainerW() - 2)
	if m.insertMode {
		o := m.renderInsertInputOverlay(m.preferenceQues[m.cursor].title.string(), m.txtInput.View(), m.txtInput.Width)
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
	w := largeContainerW()
	m.vp.Width, m.vp.Height = w, h
	w = 50 // input field width
	if m.vp.Width < w {
		w = m.vp.Width

	}
	m.txtInput.Width = w - 5 -
		preferenceQueOverlayContainerStyle.GetHorizontalFrameSize() -
		m.vp.Style.GetHorizontalFrameSize()

	if !m.insertMode {
		m.cursor = 0
		m.vp.GotoTop()
	}
}

func (m *preferenceModel) updateTitleStyleAsFocus() {
	if currentFocus == extension {
		m.titleStyle = titleStyle.
			Background(highlightColor).
			Foreground(subduedHighlightColor)
	} else {
		m.titleStyle = titleStyle.
			Background(grayColor).
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
	savedStatus := "Saved"
	if m.isUnsavedState() {
		savedStatus = "Unsaved"
	}
	s := fmt.Sprintf("Cursor: %d/%d • State: %s", m.cursor+1, len(m.preferenceQues), savedStatus)
	return extStatusBarStyle.Render(s)
}

func (m preferenceModel) renderSectionTitle(t string) string {
	return preferenceSectionStyle.
		Width(m.vp.Width - m.vp.Style.GetHorizontalFrameSize() - preferenceSectionStyle.GetHorizontalBorderSize()).
		Render(t)
}

func (m preferenceModel) renderInactiveQue(q preferenceQue) string {
	title := truncateRenderedTitle(q.title.string())
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
		Width(m.vp.Width - m.vp.Style.GetHorizontalFrameSize() - preferenceQueContainerStyle.GetHorizontalBorderSize()).
		Render(ques)
}

func (m *preferenceModel) renderActiveQue(q preferenceQue) string {
	title := truncateRenderedTitle(q.title.string())
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
		Width(m.vp.Width - m.vp.Style.GetHorizontalFrameSize() - preferenceQueContainerStyle.GetHorizontalBorderSize()).
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
	btn1 := preferenceQueBtnStyle  // negative
	btn2 := preferenceQueBtnStyle. // positive!
					Background(highlightColor).
					Foreground(subduedHighlightColor).
					Faint(true)
	if !check { // btn1(negative) should be active
		btn1, btn2 = btn2, btn1
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, btn1.Render("NOPE"), btn2.Render("YUP!"))
}

func (m preferenceModel) renderInsertInputOverlay(title, inputView string, width int) string {
	topLeft, topRight := "╭─", "─╮"
	tail := "…"
	topBorderCornerW := lipgloss.Width(topLeft + topRight)

	title = runewidth.Truncate(title, width+topBorderCornerW+lipgloss.Width(tail), tail)
	title = preferenceQueTitleStyle.
		Background(highlightColor).
		Foreground(subduedHighlightColor).
		Faint(true).
		Render(title)

	titleW := lipgloss.Width(title)
	reqTopBorderW := width + topBorderCornerW - 1 +
		preferenceQueOverlayContainerStyle.GetHorizontalPadding() +
		preferenceQueOverlayContainerStyle.GetHorizontalBorderSize()

	var padAfterTitle string
	if titleW < reqTopBorderW {
		n := reqTopBorderW - titleW
		padAfterTitle = strings.Repeat("─", n)
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
	return lipgloss.JoinVertical(lipgloss.Top, borderTop, body)
}

func (m *preferenceModel) updateKeymap(disable bool) {
	m.disableKeymap = disable
}

func (m *preferenceModel) activateInsertMode() tea.Cmd {
	m.insertMode = m.preferenceQues[m.cursor].pType == input
	m.txtInput.Prompt = m.preferenceQues[m.cursor].prompt
	s := m.preferenceQues[m.cursor].input
	m.txtInput.SetValue(s)
	m.txtInput.SetCursor(utf8.RuneCountInString(s))
	return m.txtInput.Focus()
}

func (m preferenceModel) resetInsertMode() {
	m.txtInput.Prompt = ""
	m.txtInput.SetValue("")
	m.insertMode = false
	m.txtInput.Blur()
}

func (m *preferenceModel) resetToSavedState() {
	cfg, _ := config.Load() // initialPreferenceModel loaded the config -> err ignored
	for i, q := range m.preferenceQues {
		switch q.title {
		case username:
			m.preferenceQues[i].input = cfg.Personal.Username
		case instanceName:
			m.preferenceQues[i].input = cfg.Share.InstanceName
		case stoppableInstance:
			m.preferenceQues[i].check = cfg.Share.StoppableInstance
		case zipFiles:
			m.preferenceQues[i].check = cfg.Share.ZipFiles
		case compression:
			m.preferenceQues[i].check = cfg.Share.Compression
		case sharedZipName:
			m.preferenceQues[i].input = cfg.Share.SharedZipName
		case downloadFolder:
			m.preferenceQues[i].input = cfg.Receive.DownloadFolder
		case concurrentDownloads:
			m.preferenceQues[i].input = strconv.Itoa(cfg.Receive.ConcurrentDownloads)
		}
	}
}

func (m *preferenceModel) inactivePreference() tea.Cmd {
	m.cursor = 0
	m.handleViewportScroll(up)
	m.renderViewport()
	return msgToCmd(preferenceInactiveMsg{})
}

func (m preferenceModel) savePreferences(exit bool) tea.Cmd {
	cfg := config.Config{}
	for _, q := range m.preferenceQues {
		switch q.title {
		case username:
			cfg.Personal.Username = q.input
		case instanceName:
			cfg.Share.InstanceName = q.input
		case stoppableInstance:
			cfg.Share.StoppableInstance = q.check
		case zipFiles:
			cfg.Share.ZipFiles = q.check
		case compression:
			cfg.Share.Compression = q.check
		case sharedZipName:
			cfg.Share.SharedZipName = q.input
		case downloadFolder:
			cfg.Receive.DownloadFolder = q.input
		case concurrentDownloads:
			cfg.Receive.ConcurrentDownloads, _ = strconv.Atoi(q.input)
		}
	}
	return func() tea.Msg {
		if err := config.Save(cfg); err != nil {
			return errMsg{
				err:   err,
				fatal: true,
			}
		}
		return preferencesSavedMsg(exit)
	}
}

func (m preferenceModel) isUnsavedState() bool {
	var unsaved bool
	cfg, _ := config.Load() // initialPreferenceModel loaded the config -> err ignored
	for _, q := range m.preferenceQues {
		// early return so we don't loop for other titles
		if unsaved {
			return unsaved
		}
		switch q.title {
		case username:
			unsaved = q.input != cfg.Personal.Username
		case instanceName:
			unsaved = q.input != cfg.Share.InstanceName
		case stoppableInstance:
			unsaved = q.check != cfg.Share.StoppableInstance
		case zipFiles:
			unsaved = q.check != cfg.Share.ZipFiles
		case compression:
			unsaved = q.check != cfg.Share.Compression
		case sharedZipName:
			unsaved = q.input != cfg.Share.SharedZipName
		case downloadFolder:
			unsaved = q.input != cfg.Receive.DownloadFolder
		case concurrentDownloads:
			unsaved = q.input != strconv.Itoa(cfg.Receive.ConcurrentDownloads)
		}
	}
	return unsaved
}

// grant the discard request and envelops "esc" as a command for positiveFunc
func (m *preferenceModel) confirmSaveChanges() tea.Cmd {
	selBtn := positive
	header := "UPDATE PREFERENCES?"
	body := "Do you want to update preferences, unsaved changes will be lost."
	yupFunc := func() tea.Cmd {
		return tea.Batch(m.inactivePreference(), m.savePreferences(true))
	}
	nopeFunc := func() tea.Cmd {
		m.resetToSavedState()
		return tea.Sequence(msgToCmd(rerenderPreferencesMsg{}), m.inactivePreference())
	}
	return msgToCmd(alertDialogMsg{
		header:         header,
		body:           body,
		cursor:         selBtn,
		positiveBtnTxt: "YUP!",
		negativeBtnTxt: "NOPE",
		positiveFunc:   yupFunc,
		negativeFunc:   nopeFunc,
	})
}

func (m preferenceModel) validateInput(in string) (bool, string) {
	switch m.preferenceQues[m.cursor].title {
	case username:
		return utf8.RuneCountInString(in) >= 3, "Username must be at least 3 characters long."
	case instanceName:
		return utf8.RuneCountInString(in) >= 3, "Instance name must be at least 3 characters long."
	case sharedZipName:
		return utf8.RuneCountInString(in) >= 3 && strings.HasSuffix(in, ".zip"),
			"Shared ZIP name must be at least 3 characters long & ends with “.zip”"
	case downloadFolder:
		fstat, err := os.Stat(in)
		return err == nil && fstat.IsDir(), "Download folder must be a valid directory path with read & write access."
	case concurrentDownloads:
		n, err := strconv.Atoi(in)
		return err == nil && n >= 1 && n <= 10, "Concurrent downloads must be a number between 1 and 10."
	default:
		return true, ""
	}
}

func (m preferenceModel) showInvalidInputAlert(txt string) tea.Cmd {
	return msgToCmd(alertDialogMsg{header: "INVALID INPUT!", body: txt})
}

func populatePreferencesFromConfig(cfg config.Config) []preferenceQue {
	return []preferenceQue{
		{
			title:  username,
			desc:   "Your display name visible to other users on the local network.",
			prompt: "Name: ",
			pType:  input,
			pSec:   personal,
			input:  cfg.Personal.Username,
		},
		{
			title:  instanceName,
			desc:   "Custom name of the server instance, accessible in browser through “http://instance-name.local”",
			prompt: "Name: ",
			pType:  input,
			pSec:   share,
			input:  cfg.Share.InstanceName,
		},
		{
			title: stoppableInstance,
			desc:  "Allow others on the same LAN to shutdown your server instance when no downloads are active.",
			pType: option,
			pSec:  share,
			check: cfg.Share.StoppableInstance,
		},
		{
			title: zipFiles,
			desc:  "Combine all selected files into a single zip archive. When disabled, each directory will be zipped separately.",
			pType: option,
			pSec:  share,
			check: cfg.Share.ZipFiles,
		},
		{
			title: compression,
			desc:  "Compress selected files while zipping, no compression will be significantly faster.",
			pType: option,
			pSec:  share,
			check: cfg.Share.Compression,
		},
		{
			title:  sharedZipName,
			desc:   "Name of the archive selected files will be zipped into.",
			prompt: "Name: ",
			pType:  input,
			pSec:   share,
			input:  cfg.Share.SharedZipName,
		},
		{
			title:  downloadFolder,
			desc:   "Absolute path to a folder where files will be downloaded.",
			prompt: "Path: ",
			pType:  input,
			pSec:   receive,
			input:  cfg.Receive.DownloadFolder,
		},
		{
			title:  concurrentDownloads,
			desc:   "Maximum number of files that can be downloaded concurrently.",
			prompt: "Count: ",
			pType:  input,
			pSec:   receive,
			input:  strconv.Itoa(cfg.Receive.ConcurrentDownloads),
		},
	}
}

func truncateRenderedTitle(title string) string {
	subW := largeContainerStyle.GetHorizontalFrameSize() +
		preferenceQueContainerStyle.GetHorizontalFrameSize() +
		preferenceQueTitleStyle.GetHorizontalFrameSize()
	titleW := largeContainerW() - subW
	return runewidth.Truncate(title, titleW, "…")
}

func customPreferenceHelp(show bool) *lipTable.Table {
	baseStyle := lipgloss.NewStyle()
	var rows [][]string
	if !show {
		rows = [][]string{{"?", "help"}}
	} else {
		rows = [][]string{
			{"tab/shift+tab", "move cursor (looped)"},
			{"↓/↑", "move cursor"},
			{"←/→ OR l/h", "switch option"},
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
