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
	"log/slog"
	"net/http"
	"os"
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
	// the instance is serving indexes and is not idle
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
	txtInput                            textinput.Model
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
	case "left", "right", "h", "l", "enter", "q", "Q", "?":
		return !m.disableKeymap
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
				lch, dch := make(chan server.Log, 10), make(chan int, 1)
				m.server = server.New(conf.Share.StoppableInstance, lch, dch)
				extSendChCmd := msgToCmd(handleExtSendCh{lch, dch})
				return m, tea.Sequence(extSendChCmd, m.publishInstanceAndStartServer())
			}

		case "Q", "q":
			if m.isServing {
				return m, m.shutdownServer(false)
			}

		case "ctrl+c":
			if len(m.files) > 0 {
				m.deleteTempFiles()
			}

		case "esc":
			if !m.isServing {
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
			lch, dch := make(chan server.Log, 10), make(chan int, 1)
			m.server = server.New(m.getConfig().Share.StoppableInstance, lch, dch)
			extSendChCmd := msgToCmd(handleExtSendCh{lch, dch})
			return m, tea.Sequence(extSendChCmd, m.publishInstanceAndStartServer())

		case notResponding:
			m.isSelected = false
			m.instanceState = idle
			return m, msgToCmd(errMsg{
				errHeader: strings.ToUpper(http.StatusText(http.StatusRequestTimeout)),
				errStr:    "Request failed, the server instance is not responding, it might be down.",
				fatal:     false,
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

func (m *sendModel) handleUpdate(msg tea.Msg) tea.Cmd {
	var cmds [1]tea.Cmd
	m.txtInput, cmds[0] = m.txtInput.Update(msg)
	return tea.Batch(cmds[:]...)
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

	switch m.btnIdx {
	case defaultInstance:
		sb.WriteString(baseStyle.Render("Public default address, “http://letshare.local”"))
	case customInstance:
		s := baseStyle.Render("For privacy, a custom address, to update hit “ctrl+p”")
		if m.isSelected && m.btnIdx == customInstance {
			s = baseStyle.Render("Private custom address, “http://" + m.customInstance + ".local”")
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
		sb.WriteString(baseStyle.UnsetBlink().Render("The server instance is up & running… Hit “Q/q” to shutdown."))
	} else if m.isShutdown {
		sb.WriteString(baseStyle.Foreground(highlightColor).Blink(true).Render("Shutting down the server instance, please wait…"))
	} else {
		switch m.instanceState {
		case idle, notResponding, available:
		case requesting:
			sb.WriteString(baseStyle.Foreground(yellowColor).Render("Instance already serving, requesting shutdown…"))
		case serving:
			currentOwner := m.getInstanceOwner(m.getInstance())
			msg := fmt.Sprintf("Server is currently serving indexes for %q. They're notified of your request, Please either wait or switch instance.", currentOwner)
			sb.WriteString(baseStyle.Foreground(redColor).Render(msg))
		case requestAccepted:
			currentOwner := m.getInstanceOwner(m.getInstance())
			msg := fmt.Sprintf("Shutting down the server instance serving by %q please wait…", currentOwner)
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
	body := "This will delete all the processed indexes, and you'll return to file selection."
	positiveFunc := func() tea.Cmd {
		m.isSelected, m.isServing = false, false
		return tea.Batch(msgToCmd(localChildSwitchMsg{child: dirNav, focus: true}), m.deleteTempFiles())
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

func (m sendModel) showShutdownReqWhenNotIdleAlert(reqBy string) tea.Cmd {
	body := fmt.Sprintf("%q requested shutdown, but the server is currently serving indexes. Consider shutting down when idle.", reqBy)
	return alertDialogMsg{
		header:        "SHUTDOWN REQUEST",
		body:          body,
		alertDuration: 10 * time.Second,
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
			{"esc", "cancel sharing"},
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
		ch := make(chan string, 1) // client writes only once then closes the channel
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
			err = m.mdns.Publish(m.server.StopCtx, instance, instance, uname, 80)
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			return errMsg{err: err, fatal: true}
		}
		return nil
	}
	cmds[1] = func() tea.Msg {
		var err error
		bgtask.Get().RunAndBlock(func(_ context.Context) {
			err = m.server.StartServer(m.files...)
		})
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return errMsg{err: err, fatal: true}
		}
		return instanceShutdownMsg{}
	}
	cmds[2] = m.notifyForShutdownReqWhenNotIdle()
	cmds[3] = msgToCmd(instanceServingMsg{})
	return tea.Batch(cmds[:]...)
}

func (m *sendModel) shutdownServer(quit bool) tea.Cmd {
	return func() tea.Msg {
		if err := m.server.ShutdownServer(false); err != nil && errors.Is(err, server.ErrNonIdle) {
			positiveFunc := func() tea.Cmd {
				_ = m.server.ShutdownServer(true)
				if quit {
					shutdownBgTasks()
					return tea.Quit
				}
				return msgToCmd(instanceShutdownMsg{})
			}
			return alertDialogMsg{
				header:         "FORCE SHUTDOWN?",
				body:           "The server is currently serving indexes, do you want to force shutdown, ongoing downloads will abruptly halt.",
				positiveBtnTxt: "YUP!",
				negativeBtnTxt: "NOPE",
				cursor:         positive,
				positiveFunc:   positiveFunc,
			}
		}
		if quit {
			shutdownBgTasks()
			return tea.Quit()
		}
		return instanceShutdownMsg{}
	}
}

func (m *sendModel) stopOwnedServerInstance(instance string) tea.Cmd {
	return func() tea.Msg {
		statusCode, err := m.client.StopServer(instance)
		if err != nil {
			return tea.Batch(instanceStateCmd(idle), msgToCmd(errMsg{err: err, fatal: false}))
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

func (m sendModel) deleteTempFiles() tea.Cmd {
	return func() tea.Msg {
		for _, p := range m.files {
			if strings.HasPrefix(p, os.TempDir()) {
				if err := os.Remove(p); err != nil {
					slog.Error("deleting temp indexes", "err", err)
				}
			}
		}
		return nil
	}
}
