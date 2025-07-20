package tui

import (
	"fmt"
	"github.com/MuhamedUsman/letshare/internal/mdns"
	"github.com/MuhamedUsman/letshare/internal/server"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	lipTable "github.com/charmbracelet/lipgloss/table"
	"github.com/mattn/go-runewidth"
	"github.com/mdp/qrterminal/v3"
	"github.com/muesli/reflow/truncate"
	"strings"
	"sync/atomic"
)

var instanceExtractor = strings.NewReplacer("http://", "", ".local", "")

type receiveModel struct {
	mdns                                                    *mdns.MDNS
	trackInstance                                           *atomic.Pointer[string]
	instanceInput                                           textinput.Model
	unsavedInput                                            string
	titleStyle                                              lipgloss.Style
	disableKeymap, fetchedOnce, showHelp, instanceAvailable bool
}

func initialReceiveModel() receiveModel {
	t := textinput.New()
	sug := []string{
		mdns.DefaultInstance,
		mdns.DefaultInstance + ".local",
		"http://" + mdns.DefaultInstance + ".local",
	}
	t.SetSuggestions(sug)
	t.SetValue(makeURL(mdns.DefaultInstance))
	s := lipgloss.NewStyle().Foreground(midHighlightColor).Italic(true)
	t.TextStyle, t.Cursor.TextStyle, t.Cursor.Style = s, s, s
	t.ShowSuggestions = true
	t.Prompt = ""

	return receiveModel{
		mdns:          mdns.Get(),
		trackInstance: new(atomic.Pointer[string]),
		instanceInput: t,
		titleStyle:    titleStyle.Margin(0, 2),
		disableKeymap: true,
	}
}

func (m receiveModel) capturesKeyEvent(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "i", "I", "esc", "enter", " ", "ctrl+r", "?":
		return !m.disableKeymap
	default:
		return m.instanceInput.Focused() && !m.disableKeymap && msg.String() != "ctrl+c"
	}
}

func (m receiveModel) Init() tea.Cmd {
	m.trackInstance.Store(ptr(mdns.DefaultInstance))
	return tea.Batch(textinput.Blink, m.trackInstanceAvailabilityOnChange())
}

func (m receiveModel) Update(msg tea.Msg) (receiveModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.disableKeymap {
			return m, m.handleInstanceInputUpdate(msg)
		}

		switch msg.String() {
		case "i", "I": // focus instance input
			if !m.instanceInput.Focused() {
				m.updateInstanceInputStyleAsFocus(true)
				m.unsavedInput = m.instanceInput.Value()
				// when in focus just display the instance name
				m.instanceInput.SetValue(instanceExtractor.Replace(m.instanceInput.Value()))
				m.instanceInput.CursorEnd()
				return m, tea.Batch(m.instanceInput.Focus(), m.handleInstanceInputUpdate(nil))
			}

		case "enter":
			if m.instanceInput.Focused() {
				m.updateInstanceInputStyleAsFocus(false)
				s := m.instanceInput.Value()
				if strings.TrimSpace(s) == "" {
					m.instanceInput.SetValue(mdns.DefaultInstance)
					s = mdns.DefaultInstance
				}
				s = instanceExtractor.Replace(s)
				m.trackInstance.Store(ptr(s))
				// make url once out of focus
				m.instanceInput.SetValue(makeURL(s))
				m.instanceInput.Blur()
				m.instanceAvailable = m.isInstanceAvailable()
			}

		case " ":
			if m.instanceAvailable {
				m.fetchedOnce = true
				fetchCmd := msgToCmd(fetchFileIndexesMsg(*m.trackInstance.Load()))
				return m, tea.Batch(msgToCmd(extensionChildSwitchMsg{child: extReceive, focus: true}), fetchCmd)
			}

		case "esc":
			if m.instanceInput.Focused() {
				m.updateInstanceInputStyleAsFocus(false)
				m.instanceInput.SetValue(m.unsavedInput)
				m.instanceInput.Blur()
			}

		case "ctrl+r":
			m.mdns.ReloadBrowser()

		case "?":
			if !m.instanceInput.Focused() {
				m.showHelp = !m.showHelp
			}

		}

	case spaceFocusSwitchMsg:
		m.updateTitleStyleAsFocus()

	case instanceAvailabilityMsg:
		m.instanceAvailable = bool(msg)
		if !m.instanceAvailable {
		}
		return m, m.trackInstanceAvailabilityOnChange()

	case fetchFileFailedMsg:
		m.fetchedOnce = false
	}

	return m, m.handleInstanceInputUpdate(msg)
}

func (m receiveModel) View() string {
	title := m.renderTitle()
	w := smallContainerW() - smallContainerStyle.GetHorizontalFrameSize()
	help := customReceiveHelp(m.showHelp).Width(w).Render()

	mvH := workableH() - lipgloss.Height(title) - lipgloss.Height(help) - 1
	mvs := []string{ // middle views
		m.renderInstanceInput(),
		m.renderInfoText(),
	}
	if m.instanceAvailable {
		mvs = append(mvs, m.renderInstanceConnectionInfo())
	}
	mv := lipgloss.JoinVertical(lipgloss.Center, mvs...)
	mv = lipgloss.PlaceVertical(mvH, lipgloss.Center, mv)

	return lipgloss.JoinVertical(lipgloss.Top, title, mv, help)
}

func (m receiveModel) renderTitle() string {
	subW := smallContainerW() - m.titleStyle.GetHorizontalFrameSize() - 2
	t := runewidth.Truncate("Remote Space", subW, "…")
	t = m.titleStyle.MarginBottom(2).Render(t)
	return lipgloss.PlaceHorizontal(smallContainerW()-smallContainerStyle.GetHorizontalFrameSize(), lipgloss.Right, t)
}

func (m receiveModel) renderInstanceInput() string {
	headerStyle := lipgloss.NewStyle().
		Border(lipgloss.InnerHalfBlockBorder(), false, true).
		BorderForeground(subduedHighlightColor).
		Background(subduedHighlightColor).
		Foreground(highlightColor).
		Width(smallContainerW()-smallContainerStyle.GetHorizontalFrameSize()-2).
		Padding(0, 1).
		Align(lipgloss.Center)

	inputStyle := receiveInstanceInputContainerStyle
	w := smallContainerW() - smallContainerStyle.GetHorizontalFrameSize() - inputStyle.GetHorizontalBorderSize()
	inputStyle = inputStyle.Width(w)

	if m.instanceInput.Focused() {
		headerStyle = headerStyle.BorderForeground(highlightColor).
			Background(highlightColor).
			Foreground(subduedHighlightColor).
			Faint(true)
		inputStyle = inputStyle.BorderForeground(highlightColor).Foreground(highlightColor)
	}

	w = smallContainerW() - smallContainerStyle.GetHorizontalFrameSize() - 2 - headerStyle.GetHorizontalFrameSize()
	h := runewidth.Truncate("LOOKUP INSTANCE", w, "…")

	inputView := m.instanceInput.View()
	w = smallContainerW() -
		smallContainerStyle.GetHorizontalFrameSize() -
		receiveInstanceInputContainerStyle.GetHorizontalFrameSize()
	if !m.instanceInput.Focused() {
		// when not focused, truncate the input value to fit the width
		// because width set to textInput.Model only takes effect when focused
		inputView = truncate.String(inputView, uint(w))
	}

	return lipgloss.JoinVertical(
		lipgloss.Center,
		headerStyle.Render(h),
		inputStyle.Render(inputView),
	)
}

func (m receiveModel) renderInfoText() string {
	w := smallContainerW() - smallContainerStyle.GetHorizontalFrameSize()
	style := lipgloss.NewStyle().
		Foreground(highlightColor).
		Align(lipgloss.Center).
		Faint(true).
		Padding(0, 1).
		Width(w)

	s := "Instance currently unavailable, actively searching for it…"
	if m.instanceAvailable {
		style = style.UnsetFaint()
		s = "Instance found! Press “spacebar” to view content."
	}

	return style.Render(s)
}

func (m receiveModel) renderInstanceConnectionInfo() string {
	w := smallContainerW() - smallContainerStyle.GetHorizontalFrameSize()
	baseStyle := lipgloss.NewStyle().
		Foreground(highlightColor).
		Align(lipgloss.Center).
		Width(w).
		Padding(1)
	sb := new(strings.Builder)

	w -= baseStyle.GetHorizontalFrameSize()
	divider := strings.Repeat("—", max(0, w))
	sb.WriteString(baseStyle.Foreground(subduedHighlightColor).Render(divider))

	baseStyle = baseStyle.Padding(0, 1)
	sb.WriteRune('\n')
	s := "Access via Mobile Device!"
	s = runewidth.Truncate(s, w, "…")
	sb.WriteString(baseStyle.Foreground(midHighlightColor).Render(s))
	sb.WriteRune('\n')

	ip := m.mdns.Entries()[*m.trackInstance.Load()].IP
	ip = "http://" + ip
	qr := m.generateQR(ip)
	qr = baseStyle.Render(qr)

	if server.GetPort() == server.TestHTTPPort {
		ip = fmt.Sprintf("%s:%d", ip, server.TestHTTPPort)
	}
	ip = baseStyle.Underline(true).Italic(true).Render(ip)

	titleH := lipgloss.Height(m.renderTitle())
	sbH := lipgloss.Height(sb.String())
	qrH := lipgloss.Height(qr)
	qrW := lipgloss.Width(qr)
	ipH := lipgloss.Height(ip)
	helpH := lipgloss.Height(customReceiveHelp(m.showHelp).String())

	w = smallContainerW() - smallContainerStyle.GetHorizontalFrameSize()
	// +4 Experimental, I can't make sense of it
	if titleH+sbH+qrH+ipH+helpH+4 < workableH()-smallContainerStyle.GetVerticalFrameSize() && qrW <= w {
		sb.WriteString(qr)
	}
	sb.WriteRune('\n')
	sb.WriteString(ip)

	return sb.String()
}

func (m receiveModel) generateQR(s string) string {
	sb := new(strings.Builder)
	cfg := qrterminal.Config{
		Level:      qrterminal.L,
		Writer:     sb,
		HalfBlocks: true,
		WithSixel:  true,
	}
	qrterminal.GenerateWithConfig(s, cfg)
	return sb.String()
}

func (m *receiveModel) updateTitleStyleAsFocus() {
	if currentFocus == remote {
		m.titleStyle = m.titleStyle.
			Background(highlightColor).
			Foreground(subduedHighlightColor)
	} else {
		m.titleStyle = m.titleStyle.
			Background(grayColor).
			Foreground(highlightColor)
	}
}

func (m *receiveModel) updateInstanceInputStyleAsFocus(focus bool) {
	s := lipgloss.NewStyle().Foreground(midHighlightColor).Italic(true)
	if focus {
		s = s.Foreground(highlightColor)
	}
	m.instanceInput.TextStyle, m.instanceInput.Cursor.TextStyle, m.instanceInput.Cursor.Style = s, s, s
}

func (m *receiveModel) handleInstanceInputUpdate(msg tea.Msg) tea.Cmd {
	if !m.instanceInput.Focused() && m.instanceInput.Position() > 0 {
		m.instanceInput.CursorStart()
	}
	m.updateInstanceInputDimension()
	var cmd tea.Cmd
	m.instanceInput, cmd = m.instanceInput.Update(msg)
	return cmd
}

// updateInstanceInputDimension dynamically manages textInput width for proper centering.
// - No width set: input centers itself but may wrap if content exceeds container
// - Width set: input doesn't center but prevents wrapping
// This function switches between modes based on content length vs available space.
func (m *receiveModel) updateInstanceInputDimension() {
	w := smallContainerW() -
		smallContainerStyle.GetHorizontalFrameSize() -
		receiveInstanceInputContainerStyle.GetHorizontalFrameSize()
	valW := lipgloss.Width(m.instanceInput.Value())
	if m.instanceInput.Focused() {
		// for the cursor
		w--
		valW++
	}
	if valW > w {
		m.instanceInput.Width = w
	} else if m.instanceInput.Width > 0 {
		m.instanceInput.Width = 0 // no width
	}
}

func (m *receiveModel) updateKeymap(disable bool) {
	m.disableKeymap = disable
}

func (m receiveModel) isInstanceAvailable() bool {
	i := m.trackInstance.Load()
	_, ok := m.mdns.Entries()[*i]
	return ok
}

func (m receiveModel) trackInstanceAvailabilityOnChange() tea.Cmd {
	return func() tea.Msg {
		<-m.mdns.NotifyOnChange() // blocking
		return instanceAvailabilityMsg(m.isInstanceAvailable())
	}
}

func (m receiveModel) grantExtSpaceSwitch() bool {
	return m.instanceAvailable && m.fetchedOnce
}

func customReceiveHelp(show bool) *lipTable.Table {
	baseStyle := lipgloss.NewStyle()
	var rows [][]string
	if !show {
		rows = [][]string{{"?", "help"}}
	} else {
		rows = [][]string{
			{"space", "fetch hosted files"},
			{"i/I", "focus input field"},
			{"esc", "blur input field"},
			{"enter", "confirm input"},
			{"ctrl+r", "reload MDNS browser"},
			{"?", "hide help"},
		}
	}
	return lipTable.New().
		Border(lipgloss.HiddenBorder()).
		BorderBottom(true).
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

func makeURL(s string) string {
	hasHTTP := strings.HasPrefix(s, "http://")
	hasLocal := strings.HasSuffix(s, ".local")
	if hasHTTP && hasLocal {
		return s // already a valid URL
	}
	if !hasHTTP {
		s = "http://" + s
	}
	if !hasLocal {
		s += ".local"
	}
	return s
}

func ptr[T any](v T) *T {
	return &v
}
