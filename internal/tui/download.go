package tui

import (
	"context"
	"errors"
	"fmt"
	"github.com/MuhamedUsman/letshare/internal/client"
	"github.com/MuhamedUsman/letshare/internal/config"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	lipTable "github.com/charmbracelet/lipgloss/table"
	"github.com/dustin/go-humanize"
	"github.com/mattn/go-runewidth"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var (
	spaceReplacer = strings.NewReplacer(" ", "")
	tabs          = []string{
		all.string(),
		downloading.string(),
		completed.string(),
		paused.string(),
		failed.string(),
	}
)

type downloadState int

const (
	// all exists to satisfy download 'ALL' tab
	all downloadState = iota
	downloading
	completed
	paused
	// server returned some error
	failed
	// download is just added to the downloadManager, but not started yet
	added
	// will start downloading once some other download is completed
	queued
	// indicates that the download is deleted, from disk
	// download must not show up in the UI
	// we're not removing the download from the slice,
	// because we rely on slice's index to identify the downloads
	deleted
)

var downloadStateStr = []string{
	"ALL",
	"DOWNLOADING",
	"COMPLETED",
	"PAUSED",
	"FAILED",
	"QUEUED",
	"RESUME",
}

func (ds downloadState) string() string {
	if int(ds) < 0 && int(ds) >= len(downloadStateStr) {
		return "unknown download state" + strconv.Itoa(int(ds))
	}
	return downloadStateStr[ds]
}

type fileDownload struct {
	name string
	// mDNS instance name used to download the file
	instance string
	// filename is the full path to the file on disk
	// used when deleting the file
	filename string
	// accessID is the file ID on the server
	accessID uint32
	// createdAt is the time when the download was initiated
	// it is used to sort the downloads
	createdAt, completedAt time.Time
	state                  downloadState
	// we'll only call Close func on it,
	// to indicate completion, pause, cancel
	*client.DownloadTracker
	progCh chan client.Progress
	prog   client.Progress
}

type displayDownload struct {
	*fileDownload
	// index is the index in the global downloads slice
	index int
}

type displayDownloads []*displayDownload

func (d displayDownloads) sortByTimeDesc() displayDownloads {
	slices.SortFunc(d, func(a, b *displayDownload) int {
		return b.createdAt.Compare(a.createdAt)
	})
	return d
}

func (d displayDownloads) excludeDeleted() displayDownloads {
	f := make([]*displayDownload, 0, len(d))
	for _, fd := range d {
		if fd.state != deleted {
			f = append(f, fd)
		}
	}
	return f
}

// it filters queued & downloading in one place
func (d displayDownloads) filterByCurrentStates(s downloadState) displayDownloads {
	if s == all {
		return d
	}
	f := make([]*displayDownload, 0, len(d))
	for _, fd := range d {
		if s == downloading && (fd.state == downloading || fd.state == queued) {
			f = append(f, fd)
			continue
		} else if fd.state == s {
			f = append(f, fd)
		}
	}
	return f
}

type downloadManager struct {
	// don't move around the elements in this slice
	downloads []*fileDownload
	// path to the download directory
	downloadPath string
	activeDowns  atomic.Int32
	maxDownloads int
}

type downloadModel struct {
	vp                               viewport.Model
	dm                               *downloadManager
	client                           *client.Client
	titleStyle                       lipgloss.Style
	cursor, tabIdx, prevH, selCardID int
	disableKeymap, showHelp          bool
}

func initialDownloadModel() downloadModel {
	vp := viewport.New(0, 0)
	vp.Style = vp.Style.Padding(0, 1)
	vp.MouseWheelEnabled = false
	vp.KeyMap = viewport.KeyMap{} // disable keymap

	cfg, err := config.Get()
	if err != nil {
		cfg, _ = config.Load()
	}

	dm := &downloadManager{
		downloadPath: cfg.Receive.DownloadFolder,
		maxDownloads: cfg.Receive.ConcurrentDownloads,
	}
	return downloadModel{
		vp:            vp,
		dm:            dm,
		client:        client.Get(),
		disableKeymap: true,
	}
}

func (m downloadModel) capturesKeyEvent(msg tea.KeyMsg) bool {
	return msg.String() != "ctrl+c"
}

func (m downloadModel) Init() tea.Cmd {
	return nil
}

func (m downloadModel) Update(msg tea.Msg) (downloadModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.disableKeymap {
			return m, nil
		}
		switch msg.String() {
		case "tab":
			m.tabIdx = (m.tabIdx + 1) % len(tabs)
			m.cursor = 0 // reset cursor to 0 on tab switch
			m.vp.GotoTop()

		case "shift+tab":
			m.tabIdx = (m.tabIdx - 1 + len(tabs)) % len(tabs)
			m.handleViewportScroll(up)
			m.cursor = 0 // reset cursor to 0 on tab switch
			m.vp.GotoTop()

		case "right", "l":
			if m.tabIdx < len(tabs)-1 {
				m.tabIdx++
				m.cursor = 0 // reset cursor to 0 on tab switch
				m.vp.GotoTop()
			}

		case "left", "h":
			if m.tabIdx > 0 {
				m.tabIdx--
				m.cursor = 0 // reset cursor to 0 on tab switch
				m.vp.GotoTop()
			}

		case "down", "j":
			if m.cursor < m.getCurrentTabTotalCount()-1 {
				m.cursor++
			}
			m.handleViewportScroll(down)

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			m.handleViewportScroll(up)

		case "esc":
			return m, m.inactiveDownload()

		case "p":
			d := m.dm.downloads[m.selCardID]
			if d.state == downloading || d.state == queued {
				m.pauseDownloads(m.selCardID)
			}

		case "ctrl+p":
			ids := m.getDownloadIDs(downloading, queued)
			m.pauseDownloads(ids...)

		case "r":
			d := m.dm.downloads[m.selCardID]
			if d.state == paused || d.state == failed {
				return m, m.startDownloads(m.selCardID)
			}

		case "ctrl+r":
			var ids []int
			switch downloadState(m.tabIdx) {
			case all:
				ids = m.getDownloadIDs(paused, failed)
			case paused:
				ids = m.getDownloadIDs(paused)
			case failed:
				ids = m.getDownloadIDs(failed)
			default: // noop
			}
			return m, m.startDownloads(ids...)

		case "d", "delete":
			return m, m.confirmAndDelete(m.selCardID)

		case "ctrl+d":
			var ids []int
			switch downloadState(m.tabIdx) {
			case all:
				ids = m.getDownloadIDs(downloading, paused, failed, queued, added, completed)
			case downloading:
				ids = m.getDownloadIDs(downloading, queued)
			case completed:
				ids = m.getDownloadIDs(completed)
			case paused:
				ids = m.getDownloadIDs(paused)
			case failed:
				ids = m.getDownloadIDs(failed)
			default: // noop
			}
			return m, m.confirmAndDelete(ids...)

		case "?":
			m.showHelp = !m.showHelp
			m.updateDimensions()
		}
		m.renderViewport()

	case tea.WindowSizeMsg:
		m.updateDimensions()
		if m.prevH != msg.Height {
			m.prevH = msg.Height
			m.cursor = 0 // reset cursor to 0 on height resize
		}
		m.renderViewport()

	case spaceFocusSwitchMsg:
		m.updateTitleStyleAsFocus()

	case downloadSelectionsMsg:
		m.addDownloads(msg.instance, msg.selections)
		m.renderViewport()
		ids := m.getDownloadIDs(added)
		m.tabIdx = int(downloading) // switch to downloading tab
		return m, m.startDownloads(ids...)

	case downloadProgressMsg:
		d := m.dm.downloads[msg.gid]
		d.prog = msg.p
		m.renderViewport()
		return m, tea.Batch(m.trackProgress(d.progCh, msg.gid), m.handleViewportUpdate(msg))

	case downloadCompletedMsg:
		d := m.dm.downloads[msg]
		m.setDownloadAsCompleted(d)
		m.renderViewport()
		// start next downloads if any are queued
		ids := m.getDownloadIDs(queued)
		return m, m.startDownloads(ids...)

	case downloadFailedMsg:
		d := m.dm.downloads[msg.gid]
		m.setDownloadAsFailed(d)
		m.renderViewport()
		return m, tea.Batch(msgToCmd(msg.errMsg), m.handleViewportUpdate(msg))

	case deletionConfirmationMsg:
		m.renderViewport()
		if bool(msg) && m.vp.PastBottom() {
			m.vp.GotoBottom()
			m.cursor--
			m.renderViewport() // renderAgain
		}
	}

	return m, m.handleViewportUpdate(msg)
}

func (m downloadModel) View() string {
	h := customDownloadHelp(m.showHelp).
		Width(largeContainerW() - largeContainerStyle.GetVerticalFrameSize()).
		Render()
	v := []string{
		m.renderTitle("Downloads"),
		m.renderStatusBar(),
		m.renderTabs(),
		m.vp.View(),
		h,
	}
	return lipgloss.JoinVertical(lipgloss.Center, v...)
}

func (m downloadModel) renderTitle(title string) string {
	tail := "…"
	w := largeContainerW() - (lipgloss.Width(tail) + titleStyle.GetHorizontalPadding() + lipgloss.Width(tail))
	title = runewidth.Truncate(title, w, tail)
	return m.titleStyle.Render(title)
}

func (m downloadModel) renderStatusBar() string {
	tabTotal := m.getCurrentTabTotalCount()
	at := m.cursor + 1
	if tabTotal == 0 {
		at = 0
	}
	speed := ""
	if m.dm.activeDowns.Load() > 0 {
		var s int32
		for _, d := range m.dm.downloads {
			if d.state == downloading {
				s += d.prog.S
			}
		}
		speed = fmt.Sprintf(" • Total Speed: %s/s", humanize.Bytes(uint64(s)))
	}
	s := fmt.Sprintf("Cursor: %d/%d%s", at, tabTotal, speed)
	return extStatusBarStyle.Render(s)
}

func (m downloadModel) renderTabs() string {
	baseStyle := lipgloss.NewStyle().
		Background(subduedHighlightColor).
		Foreground(highlightColor).
		Padding(0, 1).
		Align(lipgloss.Center)

	t := lipTable.New().Border(lipgloss.HiddenBorder()).Wrap(false).Width(largeContainerW()).
		BorderBottom(false).BorderTop(false).BorderLeft(false).BorderBottom(false).BorderRight(false).
		StyleFunc(func(row, col int) lipgloss.Style {
			if col == m.tabIdx {
				return baseStyle.
					Background(highlightColor).
					Foreground(subduedHighlightColor).
					Faint(true)
			}
			return baseStyle
		}).Rows(tabs).Render()

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, false).
		BorderForeground(subduedHighlightColor).
		Render(t)
}

// renderViewport renders the viewport with the current downloads,
// and also sets the current cursor position, while rendering the
// active card with a different style.
func (m *downloadModel) renderViewport() {
	f := m.toDisplayDownload().
		excludeDeleted().
		filterByCurrentStates(downloadState(m.tabIdx)).
		sortByTimeDesc()

	if len(f) == 0 {
		c := lipgloss.NewStyle().
			Foreground(highlightColor).
			Faint(true).
			Italic(true).
			Render("Nothing to show here")
		c = lipgloss.Place(m.vp.Width, m.vp.Height, lipgloss.Center, lipgloss.Center, c)
		m.vp.SetContent(c)
		return
	}

	cards := make([]string, len(f))

	for i, fd := range f {
		if m.cursor == i {
			cards[i] = m.renderActiveCard(fd.fileDownload)
			m.selCardID = fd.index // store the selected card global index
			continue
		}
		cards[i] = m.renderInactiveCard(fd.fileDownload)
	}

	c := lipgloss.JoinVertical(lipgloss.Top, cards...)
	m.vp.SetContent(c)
}

func (m downloadModel) renderActiveCard(fd *fileDownload) string {
	w := m.vp.Width - m.vp.Style.GetHorizontalFrameSize() - downloadCardContainerStyle.GetHorizontalBorderSize() // container Width
	c := downloadCardContainerStyle.BorderForeground(highlightColor).Width(w)
	w = (c.GetWidth() / 2) - downloadCardProgressStyle.GetHorizontalFrameSize() // max allowed width for progress bar

	progressBar := m.constructProgressStatus(fd)
	if lipgloss.Width(progressBar) > w {
		progressBar = runewidth.Truncate(progressBar, w, "…")
	}
	progressBar = downloadCardProgressStyle.Render(progressBar)

	// remaining width for file name & -1 to add a space b/w progress bar and file name
	w = c.GetWidth() - c.GetHorizontalBorderSize() - lipgloss.Width(progressBar) - 1
	name := runewidth.Truncate(fd.name, w, "…")
	fillSpace := c.GetWidth() - c.GetHorizontalBorderSize() - lipgloss.Width(progressBar) - lipgloss.Width(name)
	fillSpace = max(0, fillSpace) // positive or a zero value
	name += strings.Repeat(" ", fillSpace)
	name = lipgloss.NewStyle().Foreground(highlightColor).Inline(true).Italic(true).Render(name)

	return c.Render(lipgloss.JoinHorizontal(lipgloss.Center, name, progressBar))
}

func (m downloadModel) renderInactiveCard(fd *fileDownload) string {
	w := m.vp.Width - m.vp.Style.GetHorizontalFrameSize() - downloadCardContainerStyle.GetHorizontalBorderSize() // container Width
	c := downloadCardContainerStyle.Width(w)
	w = (c.GetWidth() / 2) - downloadCardProgressStyle.GetHorizontalFrameSize() // max allowed width for progress bar

	progressBar := m.constructProgressStatus(fd)
	if lipgloss.Width(progressBar) > w {
		progressBar = runewidth.Truncate(progressBar, w, "…")
	}
	progressBar = downloadCardProgressStyle.
		Background(subduedHighlightColor).
		Foreground(highlightColor).
		UnsetFaint().
		Render(progressBar)

	// remaining width for file name & -1 to add a space b/w progress bar and file name
	w = c.GetWidth() - c.GetHorizontalBorderSize() - lipgloss.Width(progressBar) - 1
	name := runewidth.Truncate(fd.name, w, "…")
	fillSpace := c.GetWidth() - c.GetHorizontalBorderSize() - lipgloss.Width(progressBar) - lipgloss.Width(name)
	fillSpace = max(0, fillSpace) // positive or a zero value
	name += strings.Repeat(" ", fillSpace)
	name = lipgloss.NewStyle().Inline(true).Foreground(midHighlightColor).Render(name)

	return c.Render(lipgloss.JoinHorizontal(lipgloss.Center, name, progressBar))
}

func (m downloadModel) constructProgressStatus(fd *fileDownload) string {
	d := spaceReplacer.Replace(humanize.Bytes(uint64(fd.prog.D)))
	t := spaceReplacer.Replace(humanize.Bytes(uint64(fd.prog.T)))
	s := spaceReplacer.Replace(humanize.Bytes(uint64(fd.prog.S)))
	percent := calculatePercent(fd.prog.D, fd.prog.T)
	prog := fmt.Sprintf("%s/%s • %s/s • %s", d, t, s, percent)

	switch downloadState(m.tabIdx) {
	case all:
		switch fd.state {
		case downloading:
			return prog
		case completed:
			return fmt.Sprintf("%s • %s • Completed", t, fd.completedAt.Sub(fd.createdAt).Round(time.Second))
		case paused:
			return fmt.Sprintf("%s/%s • %s", d, t, "Paused")
		case failed:
			return fmt.Sprintf("%s/%s • %s", d, t, "Failed")
		case queued:
			return fmt.Sprintf("%s • %s", t, "Queued")
		case added, all, deleted: // noop
		}
	case downloading:
		switch fd.state {
		case downloading:
			return prog
		case queued:
			return fmt.Sprintf("%s • %s", t, "Queued")
		case all, completed, paused, failed, added, deleted: // noop
		}
	case completed:
		return fmt.Sprintf("%s • %s", t, fd.completedAt.Sub(fd.createdAt).Round(time.Second))
	case paused, failed:
		return fmt.Sprintf("%s/%s • %s", d, t, percent)
	default: // noop
	}
	return ""
}

func (m *downloadModel) handleViewportUpdate(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return cmd
}

func (m *downloadModel) handleViewportScroll(direction scrollDirection) {
	if m.cursor == 0 {
		m.vp.GotoTop()
		return
	}
	if m.cursor == len(m.dm.downloads)-1 {
		m.vp.GotoBottom()
		return
	}
	switch direction {
	case up:
		visibleTopLine := m.vp.YOffset
		// cursor is zero indexed, +1 to avoid 0*N = 0
		cardStartLine := (m.cursor + 1) * 3 // each card takes 3 lines
		cardStartLine -= 3                  // to get the starting line of the card
		//  starting line is before the visible top line
		if cardStartLine < visibleTopLine {
			m.vp.SetYOffset(cardStartLine)
		}
	case down:
		visibleBottomLine := m.vp.YOffset + m.vp.VisibleLineCount()
		cardEndLine := (m.cursor + 1) * 3 // each card takes 3 lines
		// question ending line is after the visible bottom line
		if cardEndLine > visibleBottomLine {
			m.vp.SetYOffset(cardEndLine - m.vp.VisibleLineCount())
		}
	}
}

func (m *downloadModel) updateDimensions() {
	statusBarH := extStatusBarStyle.GetHeight() + extStatusBarStyle.GetVerticalFrameSize()
	tabsH := lipgloss.Height(m.renderTabs())
	helpH := lipgloss.Height(customDownloadHelp(m.showHelp).String())
	viewportFrameH := m.vp.Style.GetVerticalFrameSize()
	h := extContainerWorkableH() - (titleStyle.GetHeight() + statusBarH + tabsH + viewportFrameH + helpH)
	w := largeContainerW()
	m.vp.Width, m.vp.Height = w, h
}

func (m *downloadModel) updateKeymap(disable bool) {
	m.disableKeymap = disable
}

func (m *downloadModel) updateTitleStyleAsFocus() {
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

func (m *downloadModel) inactiveDownload() tea.Cmd {
	m.cursor = 0
	m.handleViewportScroll(up)
	m.renderViewport()
	m.updateKeymap(true)
	return msgToCmd(downloadInactiveMsg{})
}

func (m downloadModel) getCurrentTabTotalCount() int {
	var t int
	for _, fd := range m.dm.downloads {
		if (downloadState(m.tabIdx) == fd.state || downloadState(m.tabIdx) == all) && fd.state != deleted {
			t++
		}
	}
	return t
}

func (m downloadModel) toDisplayDownload() displayDownloads {
	fd := make([]*displayDownload, len(m.dm.downloads))
	for i, d := range m.dm.downloads {
		fd[i] = &displayDownload{
			fileDownload: d,
			index:        i,
		}
	}
	return fd
}

func (m *downloadModel) addDownloads(instance string, ds []downloadSelection) {
	if len(ds) == 0 {
		m.dm.downloads = nil
		return
	}
	for _, d := range ds {
		fd := &fileDownload{
			name:      d.name,
			instance:  instance,
			accessID:  d.accessID,
			createdAt: time.Now(),
			state:     added,
		}
		m.dm.downloads = append(m.dm.downloads, fd)
	}
}

func (m *downloadModel) startDownloads(ids ...int) tea.Cmd {
	var cmds []tea.Cmd
	for _, id := range ids {
		d := m.dm.downloads[id]
		if m.dm.activeDowns.Load() >= int32(m.dm.maxDownloads) {
			d.state = queued
			continue
		}
		d.state = downloading
		d.progCh = make(chan client.Progress, 10)
		if d.prog.D == 0 {
			// no progress, so reset the createdAt time
			// so time taken to download is accurate
			d.createdAt = time.Now()
		}

		name := filepath.Join(m.dm.downloadPath, d.name)
		dt, err := client.NewDownloadTracker(name, d.progCh)
		if err != nil {
			return msgToCmd(errMsg{err: err, fatal: true})
		}
		d.DownloadTracker = dt

		cmds = append(cmds, m.trackProgress(d.progCh, id))
		cmds = append(cmds, m.downloadFile(d, id))

		m.dm.activeDowns.Add(1)
	}
	return tea.Batch(cmds...)
}

func (m *downloadModel) pauseDownloads(ids ...int) {
	for _, id := range ids {
		d := m.dm.downloads[id]
		if d.state == downloading {
			_ = d.Close()
			d.filename = d.Filename()
			m.dm.activeDowns.Add(-1)
			d.DownloadTracker = nil // dereference the tracker
		}
		d.state = paused
	}
}

func (m downloadModel) deleteDownloads(ids ...int) {
	for _, id := range ids {
		d := m.dm.downloads[id]
		if d.state == downloading {
			_ = d.Close()
			d.filename = d.Filename()
			m.dm.activeDowns.Add(-1)
		}
		_ = os.Remove(d.filename) // ignore error, if file is not found, it is already deleted
		d.state = deleted         // mark the download as deleted
		d.DownloadTracker = nil   // dereference the tracker
	}
}

func (m downloadModel) confirmAndDelete(ids ...int) tea.Cmd {
	if len(ids) == 0 {
		return nil
	}
	var body string
	if len(ids) == 1 {
		body = m.dm.downloads[ids[0]].name
		body = fmt.Sprintf("Are you sure, %q will be permanently deleted from the disk.", body)
	} else {
		body = fmt.Sprintf(
			"Are you sure, %q files from the current tab will be permanently deleted from the disk.",
			strconv.Itoa(len(ids)),
		)
	}
	pf := func() tea.Cmd {
		m.deleteDownloads(ids...) // works because fileDownload is a pointer
		single := len(ids) == 1
		return msgToCmd(deletionConfirmationMsg(single))
	}
	return msgToCmd(alertDialogMsg{
		header:         "DELETE DOWNLOADS",
		body:           body,
		cursor:         positive,
		positiveBtnTxt: "YUP!",
		negativeBtnTxt: "NOPE",
		positiveFunc:   pf,
	})
}

func (m *downloadModel) setDownloadAsCompleted(d *fileDownload) {
	d.state = completed
	m.dm.activeDowns.Add(-1)
	_ = d.Close()
	d.filename = d.Filename()
	d.completedAt = time.Now()
	d.DownloadTracker = nil // dereference the tracker
}

func (m *downloadModel) setDownloadAsFailed(d *fileDownload) {
	d.state = failed
	m.dm.activeDowns.Add(-1)
	_ = d.Close()
	d.filename = d.Filename()
	d.DownloadTracker = nil // dereference the tracker
}

// getDownloadIDs returns the IDs of the downloads that are in the given states.
func (m downloadModel) getDownloadIDs(s ...downloadState) []int {
	ids := make([]int, 0, len(m.dm.downloads))
	for i, d := range m.dm.downloads {
		for _, state := range s {
			if d.state == state {
				ids = append(ids, i)
			}
		}
	}
	return ids
}

// gid(global ID) will be used to identify the download once downloadFile sends a response back to downloadModel
func (m downloadModel) downloadFile(f *fileDownload, gid int) tea.Cmd {
	return func() tea.Msg {
		status, err := m.client.DownloadFile(f.DownloadTracker, f.instance, f.accessID)
		// we'll close the download once its completed || paused || failed in the update function
		if err != nil {
			if errors.Is(err, context.Canceled) {
				// if context is cancelled, download is stopped by user, so return quietly
				return nil
			}
			err = unwrapErr(err)
			return downloadFailedMsg{
				gid: gid,
				errMsg: errMsg{
					errHeader: "DOWNLOAD FAILED",
					errStr:    fmt.Sprintf("Failed to download file, %s.", err.Error()),
				},
			}
		}

		var errStr string
		switch status {
		case http.StatusOK, http.StatusPartialContent:
			return downloadCompletedMsg(gid)
		case http.StatusRequestTimeout:
			errStr = "Download failed, the server instance is not responding, it might be down."
		case http.StatusNotFound:
			errStr = fmt.Sprintf("Download Failed, file with access id %q not found on the server.", strconv.Itoa(int(f.accessID)))
		default:
			errStr = fmt.Sprintf("Download Failed, Server returned status code %q.", strconv.Itoa(status))
		}

		return downloadFailedMsg{
			gid: gid,
			errMsg: errMsg{
				errHeader: strings.ToUpper(http.StatusText(status)),
				errStr:    errStr,
			},
		}
	}
}

func (m downloadModel) trackProgress(ch chan client.Progress, gid int) tea.Cmd {
	return func() tea.Msg {
		for p := range ch {
			return downloadProgressMsg{p, gid}
		}
		return nil
	}
}

func (m downloadModel) deletePartailDownloads() {
	// ignore the errors, just delete the files that are partially downloaded
	// it will be called when the app is closing, so no time to show the error
	e, _ := os.ReadDir(m.dm.downloadPath)
	for _, f := range e {
		if strings.HasSuffix(f.Name(), client.IncompleteDownloadKey) {
			_ = os.Remove(filepath.Join(m.dm.downloadPath, f.Name()))
		}
	}
}

func customDownloadHelp(show bool) *lipTable.Table {
	baseStyle := lipgloss.NewStyle()
	var rows [][]string
	if !show {
		rows = [][]string{{"?", "help"}}
	} else {
		rows = [][]string{
			{"d/ctrl+d", "delete at cursor/delete all"},
			{"r/ctrl+r", "resume at cursor/resume all"},
			{"p/ctrl+p", "pause at cursor/pause all"},
			{"tab/shift+tab", "switch download tabs (looped)"},
			{"←/→ OR l/h", "switch download tabs"},
			{"↓/↑", "move cursor"},
			{"esc", "exit downloads"},
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

func calculatePercent(done, total int64) string {
	if total == 0 {
		return "0%"
	}
	percent := float64(done) / float64(total) * 100
	return fmt.Sprintf("%.1f%%", percent)
}
