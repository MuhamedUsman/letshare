package tui

import (
	"context"
	"errors"
	"fmt"
	"github.com/MuhamedUsman/letshare/internal/bgtask"
	"github.com/MuhamedUsman/letshare/internal/client"
	"github.com/MuhamedUsman/letshare/internal/config"
	"github.com/MuhamedUsman/letshare/internal/mdns"
	"github.com/MuhamedUsman/letshare/internal/server"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	lipTable "github.com/charmbracelet/lipgloss/table"
	"github.com/mattn/go-runewidth"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

type instanceBtn int

const (
	noInstance instanceBtn = iota - 1 // no instance selected
	defaultInstance
	customInstance
)

var instanceBtnStr = []string{"DEFAULT", "CUSTOM!"}

func (b instanceBtn) string() string {
	if int(b) < 0 || int(b) >= len(instanceBtnStr) {
		return "unknown instance button index" + strconv.Itoa(int(b))
	}
	return instanceBtnStr[b]
}

// requiredInstanceState indicates the state for the instance we want to use for our service
// but is currently serving for someone else on the local network
type requiredInstanceState int

const (
	// the instance became idle
	idle requiredInstanceState = iota
	// we're requesting the instance
	requesting
	// the instance is serving files and is not idle
	// the owner will be notified for our request
	serving
	// the instance became available for us to use
	available
	// the owner doesn't allow the server to be shutdown by others
	requestRejected
	// the instance will start graceful shutdown
	requestAccepted
	// request timed out
	notResponding
)

func instanceStateCmd(state requiredInstanceState) tea.Cmd {
	return func() tea.Msg {
		return state
	}
}

type sendModel struct {
	server                              *server.Server
	client                              *client.Client
	mdns                                *mdns.MDNS
	files                               []string
	customInstance                      string
	titleStyle                          lipgloss.Style
	btnIdx, selected                    instanceBtn
	instanceState                       requiredInstanceState
	isSelected, showHelp, disableKeymap bool
	isServing, isShutdown               bool
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
		client:        client.Get(),
		mdns:          mdns.Get(),
		titleStyle:    titleStyle.Margin(0, 2),
		disableKeymap: true, // initially dirNavModel will handle key events
	}
}

func (m sendModel) capturesKeyEvent(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "left", "right", "h", "l", "enter", " ", "q", "Q", "?":
		return !m.disableKeymap
	case "ctrl+r":
		return m.isSelected && m.isServing
	case "esc":
		return m.instanceState == idle && !m.disableKeymap
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
		// we're not respecting disableKeymap here, making sure we don't consume that keyEvent by send it back
		if msg.Type == tea.KeyCtrlC && len(m.files) > 0 {
			return m, tea.Batch(m.deleteTempFiles(), msgToCmd(tea.KeyMsg{Type: tea.KeyCtrlC}))
		}

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
				m.selected = m.btnIdx
				conf := m.getConfig()
				m.customInstance = conf.Share.InstanceName
				lch, dch := make(chan server.Log, 20), make(chan int, 20)
				m.server = server.New(conf.Share.StoppableInstance, lch, dch)
				extSendChCmd := msgToCmd(handleExtSendCh{lch, dch})
				return m, tea.Sequence(extSendChCmd, m.publishInstanceAndStartServer())
			}

		case "Q", "q":
			if m.isServing {
				return m, m.shutdownServer(false)
			}

		case " ":
			if m.grantExtSpaceSwitch() {
				return m, msgToCmd(extensionChildSwitchMsg{extSend, true})
			}

		case "ctrl+r":
			if m.isSelected && m.isServing {
				m.mdns.ReloadPublisher()
			}

		case "esc":
			if !m.isServing && !m.isShutdown {
				return m, m.confirmEsc()
			}

		case "?":
			m.showHelp = !m.showHelp
		}

	case spaceFocusSwitchMsg:
		m.updateTitleStyleAsFocus()

	case sendFilesMsg:
		m.files = msg

	case instanceServingMsg:
		m.isServing = true

	case serverStartupErrMsg:
		return m, tea.Batch(msgToCmd(instanceShutdownMsg{}), msgToCmd(errMsg(msg)))

	case instanceShutdownMsg:
		m.isSelected, m.isServing, m.isShutdown = false, false, true
		m.instanceState = idle

	case serverLogsTimeoutMsg:
		m.isShutdown = false
		return m, msgToCmd(localChildSwitchMsg{child: dirNav, focus: currentFocus == local})

	case shutdownReqWhenNotIdleMsg:
		return m, tea.Batch(m.notifyForShutdownReqWhenNotIdle(), m.showShutdownReqWhenNotIdleAlert(string(msg)))

	case requiredInstanceState:
		if m.isServing {
			// if our instance is already serving, ignore the state change of required instance
			return m, nil
		}

		m.instanceState = msg
		switch msg {
		case requesting:
		case idle:
			m.isSelected = false

		case available:
			m.isSelected = true
			lch, dch := make(chan server.Log, 20), make(chan int, 20)
			m.server = server.New(m.getConfig().Share.StoppableInstance, lch, dch)
			extSendChCmd := msgToCmd(handleExtSendCh{lch, dch})
			return m, tea.Sequence(extSendChCmd, m.publishInstanceAndStartServer())

		case notResponding:
			m.isSelected = false
			m.instanceState = idle
			return m, msgToCmd(errMsg{
				errHeader: strings.ToUpper(http.StatusText(http.StatusRequestTimeout)),
				errStr:    "Request failed, the server instance is not responding, it might be down.",
			})

		case requestRejected, serving:
			m.isSelected = false
			return m, m.notifyInstanceAvailability()

		case requestAccepted:
			return m, m.notifyInstanceAvailability()
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
	return lipgloss.JoinVertical(lipgloss.Top, views...)
}

func (m *sendModel) updateKeymap(disable bool) {
	m.disableKeymap = disable
}

func (m *sendModel) updateTitleStyleAsFocus() {
	if currentFocus == local {
		m.titleStyle = m.titleStyle.
			Background(highlightColor).
			Foreground(subduedHighlightColor)
	} else {
		m.titleStyle = m.titleStyle.
			Background(grayColor).
			Foreground(highlightColor)
	}
}

func (m sendModel) renderTitle() string {
	subW := smallContainerW() - m.titleStyle.GetHorizontalFrameSize() - 2
	t := runewidth.Truncate("Local Space", subW, "…")
	return m.titleStyle.Render(t)
}

func (m sendModel) renderInstanceSelectionForm() string {
	h := workableH() - 2 - lipgloss.Height(customSendHelp(m.showHelp).String())
	infoTxt := m.renderInfoText()
	if m.isSelected || m.isShutdown {
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
	title = runewidth.Truncate(title, smallContainerW()-4, "…")
	return lipgloss.NewStyle().
		Foreground(highlightColor).
		Underline(true).
		Margin(2, 1).
		Render(title)
}

func (m sendModel) renderInfoText() string {
	baseStyle := lipgloss.NewStyle().Foreground(midHighlightColor).Align(lipgloss.Center)
	sb := new(strings.Builder)
	var port string
	if server.GetPort() == server.TestHTTPPort {
		port = fmt.Sprint(":", server.TestHTTPPort)
	}
	switch m.btnIdx {
	case defaultInstance:
		s := fmt.Sprintf("Public default address, “http://letshare.local%s”", port)
		sb.WriteString(baseStyle.Render(s))
	case customInstance:
		s := baseStyle.Render("A custom address for privacy, to update hit “ctrl+p”")
		if m.isSelected && m.btnIdx == customInstance {
			s = fmt.Sprintf("Private custom address, “http://%s.local%s”", m.customInstance, port)
			s = baseStyle.Render(s)
		}
		sb.WriteString(s)
	case noInstance:
	}

	if m.instanceState != idle || m.isServing || m.isShutdown {
		sb.WriteString("\n\n")
		divider := strings.Repeat("—", max(0, smallContainerW()-4))
		sb.WriteString(baseStyle.Foreground(subduedHighlightColor).Render(divider))
		sb.WriteString("\n\n")
		baseStyle = baseStyle.Foreground(highlightColor).Blink(true)
	}

	if m.isServing {
		sb.WriteString(baseStyle.UnsetBlink().Render("The server instance is up & running… Press “Q/q” to shutdown."))
	} else if m.isShutdown {
		sb.WriteString(baseStyle.Foreground(highlightColor).Blink(true).Render("Shutting down the server instance, please wait…"))
	} else {
		switch m.instanceState {
		case idle, notResponding, available:
		case requesting:
			sb.WriteString(baseStyle.Foreground(yellowColor).Render("Instance already serving, requesting shutdown…"))
		case serving:
			currentOwner := m.getInstanceOwner(m.getInstance())
			msg := fmt.Sprintf("Server is currently serving files for %q. They're notified of your request, Please either wait or switch instance.", currentOwner)
			sb.WriteString(baseStyle.Foreground(redColor).Render(msg))
		case requestAccepted:
			currentOwner := m.getInstanceOwner(m.getInstance())
			msg := fmt.Sprintf("Shutting down the server instance serving for %q please wait…", currentOwner)
			sb.WriteString(baseStyle.Foreground(yellowColor).Render(msg))
		case requestRejected:
			currentOwner := m.getInstanceOwner(m.getInstance())
			msg := fmt.Sprintf("Instance owner %q has blocked shutdown. Please either wait for availability or switch instance.", currentOwner)
			sb.WriteString(baseStyle.Foreground(redColor).Render(msg))
		}
	}

	return lipgloss.NewStyle().
		Foreground(midHighlightColor).
		Width(smallContainerW()-smallContainerStyle.GetHorizontalFrameSize()).
		Align(lipgloss.Center).
		Padding(0, 1).
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

	w := smallContainerW() - 20 // -24 experimental
	defaultBtn := runewidth.Truncate(defaultInstance.string(), w, "…")
	customBtn := runewidth.Truncate(customInstance.string(), w, "…")

	switch m.btnIdx {
	case noInstance:
	case defaultInstance:
		defaultBtn = activeStyle.Render(defaultBtn)
		customBtn = inactiveStyle.Render(customBtn)
	case customInstance:
		defaultBtn = inactiveStyle.Render(defaultBtn)
		customBtn = activeStyle.Render(customBtn)
	}

	btns := lipgloss.JoinHorizontal(lipgloss.Center, defaultBtn, " ", customBtn)
	return lipgloss.NewStyle().Margin(0, 1).Render(btns)
}

func (m sendModel) renderSelectedInstanceHeader() string {
	style := lipgloss.NewStyle().
		Margin(2, 1, 1, 1).
		Padding(0, 2).
		Align(lipgloss.Center).
		Background(subduedHighlightColor).
		Foreground(highlightColor).
		Width(smallContainerW() - 4)
	var s string
	switch m.btnIdx {
	case noInstance:
	case defaultInstance:
		s = "DEFAULT SERVER INSTANCE"
	case customInstance:
		s = "CUSTOM SERVER INSTANCE!"
	}
	w := smallContainerW() - smallContainerStyle.GetHorizontalFrameSize() - style.GetHorizontalFrameSize()
	s = runewidth.Truncate(s, w, "…")
	return style.Render(s)
}

func (m *sendModel) confirmEsc() tea.Cmd {
	selBtn := positive
	header := "CANCEL SHARING?"
	body := "This will delete all the processed files, and you'll return to file selection."
	positiveFunc := func() tea.Cmd {
		m.isSelected, m.isServing = false, false
		return tea.Batch(msgToCmd(localChildSwitchMsg{child: dirNav, focus: true}), m.deleteTempFiles())
	}
	return msgToCmd(alertDialogMsg{
		header:         header,
		body:           body,
		cursor:         selBtn,
		positiveBtnTxt: "YUP!",
		negativeBtnTxt: "NOPE",
		positiveFunc:   positiveFunc,
	})
}

func (m sendModel) showShutdownReqWhenNotIdleAlert(reqBy string) tea.Cmd {
	body := fmt.Sprintf("%q requested shutdown, but the server is currently serving files. Consider shutting down when idle.", reqBy)
	return msgToCmd(alertDialogMsg{
		header:        "SHUTDOWN REQUEST",
		body:          body,
		alertDuration: 10 * time.Second,
	})
}

func (m sendModel) getInstance() string {
	cfg := m.getConfig()
	switch m.selected {
	case defaultInstance:
		return mdns.DefaultInstance
	case customInstance:
		return cfg.Share.InstanceName
	default:
		return ""
	}
}

func (m sendModel) isInstanceAvailable(instance string) bool {
	_, ok := m.mdns.Entries()[instance]
	return !ok
}

func (m sendModel) getInstanceOwner(instance string) string {
	return m.mdns.Entries()[instance].Owner
}

func (m sendModel) notifyInstanceAvailability() tea.Cmd {
	instance := m.getInstance()
	return func() tea.Msg {
		for {
			select {
			case <-m.server.StopCtx.Done():
				return nil // server stopped, exit the loop
			default:
				<-m.mdns.NotifyOnChange()
				if m.isInstanceAvailable(instance) {
					return available
				}
			}
		}
	}
}

func (m sendModel) notifyForShutdownReqWhenNotIdle() tea.Cmd {
	return func() tea.Msg {
		ch := make(chan string, 1) // server writes only once then closes the channel
		m.server.NotifyForShutdownReqWhenNotIdle(ch)
		select {
		case reqBy := <-ch:
			return shutdownReqWhenNotIdleMsg(reqBy)
		case <-m.server.StopCtx.Done():
		}
		return nil
	}
}

func (m sendModel) publishInstanceAndStartServer() tea.Cmd {
	instance := m.getInstance()
	if !m.isInstanceAvailable(instance) {
		return tea.Batch(instanceStateCmd(requesting), m.stopOwnedServerInstance(instance))
	}

	var cmds [4]tea.Cmd

	// publish the mdns service
	cmds[0] = func() tea.Msg {
		var err error
		uname := m.getConfig().Personal.Username

		bgtask.Get().RunAndBlock(func(_ context.Context) {
			hostname := fmt.Sprintf("%s.%s", instance, mdns.Domain)
			err = m.mdns.Publish(m.server.StopCtx, instance, hostname, uname, uint16(server.GetPort()))
		})

		if err != nil && !errors.Is(err, context.Canceled) {
			return serverStartupErrMsg(errMsg{
				errHeader: "INSTANCE PUBLISHING FAILED!",
				errStr:    unwrapErr(err).Error(),
			})
		}
		return nil
	}

	cmds[1] = func() tea.Msg {
		var err error

		bgtask.Get().RunAndBlock(func(_ context.Context) {
			err = m.server.StartServer(m.files...)
		})

		if err != nil {
			var ser server.ShutdownErr
			if errors.As(err, &ser) {
				if ser.ActiveDowns == 0 {
					return nil
				}
				return alertDialogMsg{
					header: "GRACEFUL SHUTDOWN!",
					body: fmt.Sprintf(
						"Shutting down server, %d active download connections will be served, but no new requests will be accepted.",
						ser.ActiveDowns,
					),
				}
			}
			return serverStartupErrMsg(errMsg{
				errHeader: "SERVER STARTUP FAILED!",
				errStr:    unwrapErr(err).Error(),
			})
		}
		return instanceShutdownMsg{}
	}

	cmds[2] = m.notifyForShutdownReqWhenNotIdle()
	cmds[3] = msgToCmd(instanceServingMsg{})
	return tea.Batch(cmds[:]...)
}

func (m *sendModel) shutdownServer(quitting bool) tea.Cmd {
	return func() tea.Msg {
		if quitting {
			m.server.ShutdownServer()
			return instanceShutdownMsg{}
		}

		if m.server.ActiveDowns > 0 {
			p := func() tea.Cmd {
				m.server.ShutdownServer()
				return msgToCmd(instanceShutdownMsg{})
			}
			return alertDialogMsg{
				header:         "FORCED SHUTDOWN!",
				body:           "The server instance is being shutdown forcefully, all active downloads will abruptly halt.",
				positiveBtnTxt: "YUP!",
				negativeBtnTxt: "NOPE!",
				cursor:         positive,
				positiveFunc:   p,
			}
		}

		m.server.ShutdownServer()
		return instanceShutdownMsg{}
	}
}

func (m *sendModel) stopOwnedServerInstance(instance string) tea.Cmd {
	return func() tea.Msg {
		statusCode, err := m.client.StopServer(instance)
		if err != nil {
			return tea.Batch(instanceStateCmd(idle), msgToCmd(errMsg{err: err}))
		}
		switch statusCode {
		case http.StatusAccepted:
			return requestAccepted
		case http.StatusForbidden:
			return requestRejected
		case http.StatusConflict:
			return serving
		case http.StatusRequestTimeout:
			return notResponding
		default:
			return nil
		}
	}
}

func (m sendModel) getConfig() config.Config {
	cfg, err := config.Get()
	if err != nil {
		cfg, _ = config.Load()
	}
	return cfg
}

func (m *sendModel) deleteTempFiles() tea.Cmd {
	c := slices.Clone(m.files)
	m.files = nil // clear the files slice to avoid deleting them again
	return func() tea.Msg {
		for _, p := range c {
			if strings.HasPrefix(p, os.TempDir()) {
				_ = os.Remove(p)
			}
		}
		return nil
	}
}

func (m sendModel) grantExtSpaceSwitch() bool {
	return m.isServing || m.isShutdown
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
			{"esc", "cancel sharing"},
			{"ctrl+r", "reload MDNS publisher"},
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
