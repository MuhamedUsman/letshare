package tui

import (
	"fmt"
	"github.com/MuhamedUsman/letshare/internal/server"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	lipTable "github.com/charmbracelet/lipgloss/table"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/reflow/truncate"
	"strings"
	"time"
)

type logHandler struct {
	logs     []string
	logCh    <-chan server.Log
	portSize int
}

func newLogHandler(logCh <-chan server.Log) *logHandler {
	return &logHandler{
		logs:     make([]string, 0),
		portSize: 10,
		logCh:    logCh,
	}
}

func (h *logHandler) appendLog(l string) {
	copy(h.logs[1:], h.logs[:len(h.logs)-1])
	h.logs[0] = l
}

func (h *logHandler) setLogsLength(l int) {
	h.portSize = max(0, l)
	if l <= cap(h.logs) {
		return
	}
	newLogs := make([]string, l)
	copy(newLogs, h.logs)
	h.logs = newLogs
}

// extSendModel is the model to read & view logs when server is running
// such as who is connected, what files are being downloaded, etc.
type extSendModel struct {
	escTimer                timer.Model
	lh                      *logHandler
	activeDownCh            <-chan int
	titleStyle              lipgloss.Style
	activeDowns             int
	disableKeymap, showHelp bool
}

func initialExtSendModel() extSendModel {
	return extSendModel{
		lh:            &logHandler{}, // avoiding nil pointers
		titleStyle:    titleStyle,
		disableKeymap: true,
	}
}

func (m extSendModel) capturesKeyEvent(msg tea.KeyMsg) bool {
	return !m.disableKeymap && msg.String() == "esc"
}

func (m extSendModel) Init() tea.Cmd {
	return nil
}

func (m extSendModel) Update(msg tea.Msg) (extSendModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "?":
			m.showHelp = !m.showHelp
		}

	case tea.WindowSizeMsg:
		m.updateLogsDimesions()

	case handleExtSendCh:
		m.lh = newLogHandler(msg.logCh)
		m.activeDownCh = msg.activeDownCh
		m.updateLogsDimesions()
		return m, tea.Batch(m.trackLogs(), m.trackActiveDowns(), extensionChildSwitchMsg{child: extSend}.cmd)

	case serverLogMsg:
		s := m.parseLog(server.Log(msg))
		m.lh.appendLog(s)
		return m, m.trackLogs()

	case activeDownsMsg:
		m.activeDowns = int(msg)
		return m, m.trackActiveDowns()

	case instanceShutdownMsg:
		m.escTimer = timer.NewWithInterval(5*time.Second, 100*time.Millisecond)
		return m, m.escTimer.Init()

	case spaceFocusSwitchMsg:
		flag := currentFocus == extension
		m.updateTitleStyleAsFocus(flag)

	case timer.TickMsg:
		if msg.ID == m.escTimer.ID() {
			var cmd tea.Cmd
			m.escTimer, cmd = m.escTimer.Update(msg)
			return m, cmd
		}

	case timer.TimeoutMsg:
		if msg.ID == m.escTimer.ID() {
			return m, extensionChildSwitchMsg{home, currentFocus == extension}.cmd
		}
	}

	return m, nil
}

func (m extSendModel) View() string {
	title := m.renderTitle()
	statusBar := m.renderStatusBar()
	logs := m.renderLogs()
	help := customExtSendHelp(m.showHelp).Width(largeContainerW() - 2)
	v := lipgloss.JoinVertical(lipgloss.Center, title, statusBar, logs, help.Render())
	return lipgloss.PlaceHorizontal(largeContainerW(), lipgloss.Center, v)
}

func (m extSendModel) renderTitle() string {
	title, tail := "Server Logs", "…"
	w := largeContainerW() - (lipgloss.Width(tail) + titleStyle.GetHorizontalPadding() + lipgloss.Width(tail))
	title = runewidth.Truncate(title, w, tail)
	return m.titleStyle.Render(title)
}

func (m extSendModel) renderStatusBar() string {
	s := fmt.Sprintf("Active Downloads: %d", m.activeDowns)
	if m.escTimer.Running() {
		s = fmt.Sprintf("Escaping in %.1f...", m.escTimer.Timeout.Seconds())
	}
	s = runewidth.Truncate(s, largeContainerW()-4, "…")
	return extStatusBarStyle.MarginBottom(1).Render(s)
}

func (m extSendModel) renderLogs() string {
	logs := make([]string, 0, m.lh.portSize) // lr -> logs to render
	for i := m.lh.portSize - 1; i >= 0; i-- {
		l := m.lh.logs[i]
		tail := lipgloss.NewStyle().Foreground(subduedHighlightColor).Render("…")
		t := truncate.String(l, uint(largeContainerW()-2-lipgloss.Width(tail))) // truncated
		if lipgloss.Width(l) > lipgloss.Width(t) {
			t += tail
		}
		logs = append(logs, t)
	}
	s := lipgloss.JoinVertical(lipgloss.Center, logs...)
	return lipgloss.NewStyle().Width(largeContainerW() - 2).Align(lipgloss.Center).Render(s)
}

func (m *extSendModel) updateKeymap(b bool) {
	m.disableKeymap = b
}

func (m *extSendModel) updateLogsDimesions() {
	subL := largeContainerStyle.GetVerticalFrameSize() +
		lipgloss.Height(m.renderTitle()) +
		lipgloss.Height(m.renderStatusBar()) +
		lipgloss.Height(customExtSendHelp(m.showHelp).String())
	m.lh.setLogsLength(workableH() - subL)
}

func (m *extSendModel) updateTitleStyleAsFocus(focus bool) {
	if focus {
		m.titleStyle = titleStyle.
			Background(highlightColor).
			Foreground(subduedHighlightColor)
	} else {
		m.titleStyle = titleStyle.
			Background(grayColor).
			Foreground(highlightColor)
	}
}

func (m extSendModel) trackLogs() tea.Cmd {
	return func() tea.Msg {
		for l := range m.lh.logCh {
			return serverLogMsg(l)
		}
		return nil
	}
}

func (m extSendModel) trackActiveDowns() tea.Cmd {
	return func() tea.Msg {
		for n := range m.activeDownCh {
			return activeDownsMsg(n)
		}
		return nil
	}
}

func (m extSendModel) parseLog(l server.Log) string {
	baseStyle := lipgloss.NewStyle().Inline(true)
	msgStyle := baseStyle.Foreground(midHighlightColor).Italic(true)
	argIdStyle := baseStyle.Foreground(highlightColor).Faint(true)
	argValStyle := baseStyle.Foreground(midHighlightColor).Italic(true)
	divider := baseStyle.Foreground(subduedHighlightColor).Render(" • ")

	sb := new(strings.Builder)
	sb.WriteString(msgStyle.Render(l.Msg))
	if len(l.Args) > 0 {
		sb.WriteString(divider)
	}

	for i := 0; i < len(l.Args); i += 2 {
		// Handle key
		if i < len(l.Args) {
			if s, ok := l.Args[i].(string); ok {
				sb.WriteString(argIdStyle.Render(s + ": "))
			} else {
				sb.WriteString(argIdStyle.Render("!BADKEY: "))
			}
		}
		// Handle value
		if i+1 < len(l.Args) {
			sb.WriteString(argValStyle.Render(fmt.Sprintf("“%v”", l.Args[i+1])))
			// Add divider only if there are more pairs
			if i+2 < len(l.Args) {
				sb.WriteString(divider)
			}
		}
	}
	return sb.String()
}

func customExtSendHelp(show bool) *lipTable.Table {
	baseStyle := lipgloss.NewStyle()
	var rows [][]string
	if !show {
		rows = [][]string{{"?", "help"}}
	} else {
		rows = [][]string{
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
