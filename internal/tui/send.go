package tui

import (
	"context"
	"errors"
	"github.com/MuhamedUsman/letshare/internal/client"
	"github.com/MuhamedUsman/letshare/internal/mdns"
	"github.com/MuhamedUsman/letshare/internal/server"
	"github.com/MuhamedUsman/letshare/internal/util/bgtask"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	lipTable "github.com/charmbracelet/lipgloss/table"
	"github.com/mattn/go-runewidth"
	"net/http"
	"strconv"
	"strings"
)

type instanceBtn int

const (
	defaultInstance instanceBtn = iota
	customInstance
)

var instanceBtnStr = []string{"DEFAULT", "CUSTOM!"}

func (b instanceBtn) string() string {
	if int(b) < 0 || int(b) >= len(instanceBtnStr) {
		return "unknown instance button index" + strconv.Itoa(int(b))
	}
	return instanceBtnStr[b]
}

type instanceState int

const (
	idle instanceState = iota
	requesting
	owned
	requestRejected
	requestAccepted
	serving
)

type sendModel struct {
	mdns                                *mdns.MDNS
	server                              *server.Server
	client                              *client.Client
	files                               []string
	customInstance                      string
	titleStyle                          lipgloss.Style
	txtInput                            textinput.Model
	btnIdx                              instanceBtn
	instanceState                       instanceState
	isSelected, showHelp, disableKeymap bool
}

func initialSendModel() sendModel {
	t := textinput.New()
	t.Prompt = ""
	t.ShowSuggestions = true
	t.PromptStyle = t.PromptStyle.Foreground(midHighlightColor)
	t.TextStyle = t.TextStyle.Foreground(highlightColor)
	t.Cursor.TextStyle = t.Cursor.Style.Foreground(highlightColor)
	t.Cursor.Style = t.Cursor.TextStyle.Reverse(true)
	t.PlaceholderStyle = t.PlaceholderStyle.Foreground(subduedHighlightColor)

	return sendModel{
		server:     server.New(),
		client:     client.New(),
		mdns:       mdns.Get(),
		titleStyle: titleStyle.Margin(0, 2),
	}
}

func (m sendModel) capturesKeyEvent(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "left", "right", "h", "l", "enter", "q", "Q", "?":
		return !m.disableKeymap
	case "esc":
		return m.instanceState == idle && !m.disableKeymap
	case "ctrl+c":
		return m.instanceState == serving && !m.disableKeymap
	default:
		return false
	}
}

func (m sendModel) Init() tea.Cmd {
	return nil
}

func (m sendModel) Update(msg tea.Msg) (sendModel, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		if m.disableKeymap {
			return m, nil
		}
		switch msg.String() {
		case "left", "h":
			if !m.isSelected {
				m.btnIdx = 0
			}

		case "right", "l":
			if !m.isSelected {
				m.btnIdx = 1
			}

		case "enter":
			if !m.isSelected {
				m.isSelected = true
				m.customInstance = m.getConfig().Share.InstanceName
				return m, m.startServer()
			}

		case "Q", "q":
			if m.instanceState == serving {
				m.isSelected = false
				m.instanceState = idle
				return m, tea.Batch(localChildSwitchMsg{child: dirNav, focus: true}.cmd, m.shutdownServer(false))
			}

		case "ctrl+c":
			if m.instanceState == serving {
				m.isSelected = false
				m.instanceState = idle
				return m, tea.Batch(m.shutdownServer(true))
			}

		case "esc":
			if m.instanceState == idle {
				return m, m.confirmEsc()
			}

		case "?":
			m.showHelp = !m.showHelp
		}

	case spaceFocusSwitchMsg:
		if currentFocus == local {
			m.titleStyle = m.titleStyle.
				Background(highlightColor).
				Foreground(subduedHighlightColor)
		} else {
			m.titleStyle = m.titleStyle.
				Background(grayColor).
				Foreground(highlightColor)
		}

	case sendFilesMsg:
		m.files = msg
		_ = m.server.SetFilePaths(m.files...)

	case instanceState:
		m.instanceState = msg
		if m.instanceState == idle {
			m.isSelected = false
		}

	}

	return m, nil
}

func (m sendModel) View() string {
	views := []string{
		m.renderTitle(),
		m.renderInstanceSelectionForm(),
		customSendHelp(m.showHelp).Width(smallContainerW() - 2).Render(),
	}
	view := lipgloss.JoinVertical(lipgloss.Top, views...)
	return smallContainerStyle.Width(smallContainerW()).Render(view)
}

func (m *sendModel) handleUpdate(msg tea.Msg) tea.Cmd {
	var cmds [1]tea.Cmd
	m.txtInput, cmds[0] = m.txtInput.Update(msg)
	return tea.Batch(cmds[:]...)
}

func (m sendModel) updateKeymap(disable bool) {
	m.disableKeymap = disable
}

func (m sendModel) renderTitle() string {
	subW := smallContainerW() - m.titleStyle.GetHorizontalFrameSize() - 2
	t := runewidth.Truncate("Local Space", subW, "…")
	return m.titleStyle.Render(t)
}

func (m sendModel) renderStatusBar() string {
	s := "Hello"
	style := lipgloss.NewStyle().
		Foreground(highlightColor).
		Faint(true).
		Margin(1, 1, 0, 2).
		Italic(true)
	s = runewidth.Truncate(s, smallContainerW()-style.GetHorizontalFrameSize()-1, "…")
	return style.Render(s)
}

func (m sendModel) renderInstanceSelectionForm() string {
	h := workableH() - 2 - lipgloss.Height(customSendHelp(m.showHelp).String())
	infoTxt := m.renderInfoText()
	if m.isSelected {
		s := lipgloss.JoinVertical(lipgloss.Left, m.renderSelectedInstanceHeader(), infoTxt)
		return lipgloss.PlaceVertical(h, lipgloss.Center, s)
	}
	title := m.renderFormTitle()
	btns := m.renderInstanceBtns()
	s := lipgloss.JoinVertical(lipgloss.Center, title, btns, infoTxt)
	return lipgloss.PlaceVertical(h, lipgloss.Center, s)
}

func (m sendModel) renderFormTitle() string {
	title := "SELECT SERVER INSTANCE!"
	return lipgloss.NewStyle().
		Foreground(highlightColor).
		Underline(true).
		Margin(2).
		Render(title)
}

func (m sendModel) renderInfoText() string {
	baseStyle := lipgloss.NewStyle().Foreground(midHighlightColor).Align(lipgloss.Center)
	sb := new(strings.Builder)

	switch m.btnIdx {
	case defaultInstance:
		sb.WriteString(baseStyle.Render("Public default address, “http://letshare.local”"))
	case customInstance:
		s := baseStyle.Render("For privacy, a custom address, to update hit “ctrl+p”")
		if m.isSelected && m.btnIdx == customInstance {
			s = baseStyle.Render("Private custom address, “http://" + m.customInstance + ".local”")
		}
		sb.WriteString(s)
	}

	if m.instanceState != idle {
		sb.WriteString("\n\n")
		divider := strings.Repeat("—", max(0, smallContainerW()-6))
		sb.WriteString(divider)
		sb.WriteString("\n\n")
		baseStyle = baseStyle.Foreground(highlightColor).Blink(true)
	}

	switch m.instanceState {
	case idle:
	case requesting:
		sb.WriteString(baseStyle.Foreground(yellowColor).Render("Instance already owned, requesting shutdown..."))
	case owned:
		currentOwner := m.mdns.DefaultOwner()
		sb.WriteString(baseStyle.Foreground(redColor).Render(
			"The server instance is currently serving files for",
			"“"+currentOwner+"”.",
			"Either wait or switch instance.",
		))
	case requestAccepted:
		sb.WriteString(baseStyle.Render("Shutting down the server instance, please wait..."))
	case requestRejected:
		currentOwner := m.mdns.DefaultOwner()
		if m.btnIdx == customInstance {
			currentOwner = "<ANONYMOUS HOST>"
		}
		sb.WriteString(baseStyle.Foreground(redColor).Render(
			"“"+currentOwner+"”.",
			"doesn't allow shutting down the server instance. Either wait or switch instance.",
		))
	case serving:
		sb.WriteString(baseStyle.UnsetBlink().Render("The server instance is up & running!. Hit “Q/q” to shutdown."))

	}

	return lipgloss.NewStyle().
		Foreground(midHighlightColor).
		Width(smallContainerW()-smallContainerStyle.GetHorizontalFrameSize()).
		Align(lipgloss.Center).
		Padding(0, 2).
		Render(sb.String())
}

func (m sendModel) renderInstanceBtns() string {
	inactiveStyle := lipgloss.NewStyle().
		MarginBottom(2).
		Padding(0, 2).
		Align(lipgloss.Center).
		Background(grayColor).
		Foreground(highlightColor)
	activeStyle := inactiveStyle.
		Background(highlightColor).
		Foreground(subduedHighlightColor).
		Faint(true)

	w := smallContainerW() - 24 // -24 experimental
	defaultBtn := runewidth.Truncate(defaultInstance.string(), w, "…")
	customBtn := runewidth.Truncate(customInstance.string(), w, "…")

	switch m.btnIdx {
	case defaultInstance:
		defaultBtn = activeStyle.Render(defaultBtn)
		customBtn = inactiveStyle.Render(customBtn)
	case customInstance:
		defaultBtn = inactiveStyle.Render(defaultBtn)
		customBtn = activeStyle.Render(customBtn)
	}

	btns := lipgloss.JoinHorizontal(lipgloss.Center, defaultBtn, " ", customBtn)
	return lipgloss.NewStyle().Margin(0, 2).Render(btns)
}

func (m sendModel) renderSelectedInstanceHeader() string {
	var s string
	switch m.btnIdx {
	case defaultInstance:
		s = "DEFAULT SERVER INSTANCE"
	case customInstance:
		s = "CUSTOM SERVER INSTANCE!"
	}
	return lipgloss.NewStyle().
		Margin(2, 2, 1, 2).
		Padding(0, 2).
		Align(lipgloss.Center).
		Background(subduedHighlightColor).
		Foreground(highlightColor).
		Width(smallContainerW() - 6).
		Render(s)
}

func (m *sendModel) confirmEsc() tea.Cmd {
	selBtn := positive
	header := "CANCEL SHARING?"
	body := "This will delete all the processed files, and you'll return to file selection."
	positiveFunc := func() tea.Cmd {
		m.isSelected = false
		return localChildSwitchMsg{child: dirNav, focus: true}.cmd
	}
	return alertDialogMsg{
		header:         header,
		body:           body,
		cursor:         selBtn,
		positiveBtnTxt: "YUP!",
		negativeBtnTxt: "NOPE",
		positiveFunc:   positiveFunc,
	}.cmd
}

func customSendHelp(show bool) *lipTable.Table {
	baseStyle := lipgloss.NewStyle()
	var rows [][]string
	if !show {
		rows = [][]string{{"?", "help"}}
	} else {
		rows = [][]string{
			{"←/→ OR l/h", "switch button"},
			{"enter", "select button"},
			{"Q/q", "shutdown server"},
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

func (m sendModel) getInstance() string {
	cfg := m.getConfig()
	switch m.btnIdx {
	case defaultInstance:
		return mdns.DefaultInstance
	case customInstance:
		return cfg.Share.InstanceName
	}
	return ""
}

func (m sendModel) isInstanceAvailable() bool {
	checkFor := m.getInstance()
	_, ok := m.mdns.Entries()[checkFor]
	return !ok
}

func (m sendModel) startServer() tea.Cmd {
	instance := m.getInstance()
	if !m.isInstanceAvailable() {
		return tea.Batch(instanceStateCmd(requesting), m.stopOwnedServerInstance(instance))
	}
	var cmds [3]tea.Cmd
	// publish the mdns service
	cmds[0] = func() tea.Msg {
		var err error
		uname := m.getConfig().Personal.Username
		bgtask.Get().RunAndBlock(func(shutdownCtx context.Context) {
			err = m.mdns.Publish(m.server.StopCtx, instance, instance, uname, 80)
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			return errMsg{err: err, errStr: "Publish Fucked You Up!", fatal: false}
		}
		return nil
	}
	cmds[1] = func() tea.Msg {
		var err error
		bgtask.Get().RunAndBlock(func(shutdownCtx context.Context) {
			err = m.server.StartServer()
		})
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return errMsg{err: err, fatal: false}
		}
		return nil
	}
	cmds[2] = instanceStateCmd(serving)

	return tea.Batch(cmds[:]...)
}

func (m *sendModel) shutdownServer(quit bool) tea.Cmd {
	return func() tea.Msg {
		if err := m.server.ShutdownServer(false); err != nil && errors.Is(err, server.ErrNonIdle) {
			positiveFunc := func() tea.Cmd {
				_ = m.server.ShutdownServer(true)
				if quit {
					return tea.Quit
				}
				m.server = server.New() // reset the server instance
				return nil
			}
			return alertDialogMsg{
				header:         "FORCE SHUTDOWN?",
				body:           "The server is currently serving files, do you want to force shutdown, ongoing downloads will abruptly halt.",
				positiveBtnTxt: "YUP!",
				negativeBtnTxt: "NOPE",
				cursor:         positive,
				positiveFunc:   positiveFunc,
			}
		}
		if quit {
			return tea.Quit()
		}
		m.server = server.New() // reset the server instance
		return nil
	}
}

func (m sendModel) waitForInstanceAndStartServer() tea.Cmd {
	instance := m.getInstance()
	return func() tea.Msg {
		for {
			select {
			case <-m.server.StopCtx.Done():
				return nil // server stopped, exit the loop
			default:
				m.mdns.NotifyOnChange()
				_, ok := m.mdns.Entries()[instance]
				if ok {
					return m.startServer()
				}
			}
		}
	}
}

func (m *sendModel) stopOwnedServerInstance(instance string) tea.Cmd {
	return func() tea.Msg {

		statusCode, err := m.client.StopServer(instance)
		if err != nil {
			return tea.Batch(instanceStateCmd(idle), errMsg{err: err, fatal: false}.cmd)
		}
		switch statusCode {
		case http.StatusAccepted:
			return instanceStateCmd(requestAccepted)
		case http.StatusForbidden:
			return instanceStateCmd(requestRejected)
		case http.StatusConflict:
			return instanceStateCmd(owned)
		case http.StatusRequestTimeout:
			return errMsg{
				errHeader: strings.ToUpper(http.StatusText(http.StatusRequestTimeout)),
				errStr:    "Request failed, the server instance is not responding, it might be down.",
				fatal:     false,
			}
		default:
			return m.waitForInstanceAndStartServer()
		}
	}
}

func (m sendModel) getConfig() client.Config {
	cfg, err := client.GetConfig()
	if err != nil {
		cfg, _ = client.LoadConfig()
	}
	return cfg
}
