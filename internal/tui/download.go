package tui

import (
	"fmt"
	"github.com/MuhamedUsman/letshare/internal/client"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	lipTable "github.com/charmbracelet/lipgloss/table"
	"github.com/dustin/go-humanize"
	"github.com/mattn/go-runewidth"
	"maps"
	"slices"
	"strconv"
	"strings"
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
	// all exists to satisfy download tabs
	all downloadState = iota
	downloading
	completed
	paused
	failed
	queued
	resume
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
	// accessID is the file ID on the server
	accessID uint32
	// createdAt is the time when the download was initiated
	// it is used to sort the downloads
	createdAt time.Time
	state     downloadState
	// we'll only call Close func on it,
	// to indicate completion, pause, cancel
	*client.DownloadTracker
	prog client.Progress
}
type downloadManager struct {
	downloads map[int]*fileDownload
}

type downloadModel struct {
	vp                      viewport.Model
	dm                      *downloadManager
	titleStyle              lipgloss.Style
	cursor, tabIdx, prevH   int
	disableKeymap, showHelp bool
}

func initialDownloadModel() downloadModel {
	vp := viewport.New(0, 0)
	vp.Style = vp.Style.Padding(0, 1)
	vp.MouseWheelEnabled = false
	vp.KeyMap = viewport.KeyMap{} // disable keymap

	dm := &downloadManager{
		downloads: mockDownloads(),
	}
	return downloadModel{
		vp:            vp,
		dm:            dm,
		disableKeymap: true,
	}
}

func mockDownloads() map[int]*fileDownload {
	return map[int]*fileDownload{
		1: {
			name:            "Head First O’Reilly - Kathy Sierra, Bert Bates, Trisha Gee - Head First Java_ A Brain-Friendly Guide-O'Reilly Media (2022).pdf",
			accessID:        1,
			createdAt:       time.Now().Add(-10 * time.Minute),
			state:           failed,
			prog:            client.Progress{D: 1024 * 1024 * 10, T: 1024 * 1024 * 50, S: 1024 * 512},
			DownloadTracker: new(client.DownloadTracker),
		},
		2: {
			name:            "Clean Code - Robert C. Martin.pdf",
			accessID:        2,
			createdAt:       time.Now().Add(-9 * time.Minute),
			state:           downloading,
			prog:            client.Progress{D: 1024 * 1024 * 20, T: 1024 * 1024 * 40, S: 1024 * 256},
			DownloadTracker: new(client.DownloadTracker),
		},
		3: {
			name:            "The Pragmatic Programmer - Andrew Hunt, David Thomas.pdf",
			accessID:        3,
			createdAt:       time.Now().Add(-8 * time.Minute),
			state:           completed,
			prog:            client.Progress{D: 1024 * 1024 * 5, T: 1024 * 1024 * 30, S: 1024 * 128},
			DownloadTracker: new(client.DownloadTracker),
		},
		4: {
			name:            "Design Patterns - Erich Gamma, Richard Helm, Ralph Johnson, John Vlissides.pdf",
			accessID:        4,
			createdAt:       time.Now().Add(-7 * time.Minute),
			state:           paused,
			prog:            client.Progress{D: 1024 * 1024 * 15, T: 1024 * 1024 * 60, S: 1024 * 1024},
			DownloadTracker: new(client.DownloadTracker),
		},
		5: {
			name:            "Refactoring - Martin Fowler.pdf",
			accessID:        5,
			createdAt:       time.Now().Add(-6 * time.Minute),
			state:           queued,
			prog:            client.Progress{D: 1024 * 1024 * 25, T: 1024 * 1024 * 80, S: 1024 * 256},
			DownloadTracker: new(client.DownloadTracker),
		},
		6: {
			name:            "Effective Java - Joshua Bloch.pdf",
			accessID:        6,
			createdAt:       time.Now().Add(-5 * time.Minute),
			state:           downloading,
			prog:            client.Progress{D: 1024 * 1024 * 30, T: 1024 * 1024 * 90, S: 1024 * 512},
			DownloadTracker: new(client.DownloadTracker),
		},
		7: {
			name:            "Introduction to Algorithms - Cormen, Leiserson, Rivest, Stein.pdf",
			accessID:        7,
			createdAt:       time.Now().Add(-4 * time.Minute),
			state:           downloading,
			prog:            client.Progress{D: 1024 * 1024 * 12, T: 1024 * 1024 * 100, S: 1024 * 64},
			DownloadTracker: new(client.DownloadTracker),
		},
		8: {
			name:            "You Don't Know JS - Kyle Simpson.pdf",
			accessID:        8,
			createdAt:       time.Now().Add(-3 * time.Minute),
			state:           downloading,
			prog:            client.Progress{D: 1024 * 1024 * 8, T: 1024 * 1024 * 20, S: 1024 * 32},
			DownloadTracker: new(client.DownloadTracker),
		},
		9: {
			name:            "Cracking the Coding Interview - Gayle Laakmann McDowell.pdf",
			accessID:        9,
			createdAt:       time.Now().Add(-2 * time.Minute),
			state:           downloading,
			prog:            client.Progress{D: 1024 * 1024 * 18, T: 1024 * 1024 * 70, S: 1024 * 256},
			DownloadTracker: new(client.DownloadTracker),
		},
		10: {
			name:            "Grokking Algorithms - Aditya Bhargava.pdf",
			accessID:        10,
			createdAt:       time.Now().Add(-1 * time.Minute),
			state:           downloading,
			prog:            client.Progress{D: 1024 * 1024 * 2, T: 1024 * 1024 * 10, S: 1024 * 16},
			DownloadTracker: new(client.DownloadTracker),
		},
	}
}

func sortByTimeDesc(fd []*fileDownload) {
	slices.SortFunc(fd, func(a, b *fileDownload) int {
		return b.createdAt.Compare(a.createdAt)
	})
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
	s := fmt.Sprintf("Cursor: %d/%d", at, tabTotal)
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

func (m *downloadModel) renderViewport() {
	d := slices.Collect(maps.Values(m.dm.downloads)) // convert values off map[int]*fileDownload to []fileDownload
	sortByTimeDesc(d)
	f := m.filterByCurrentState(d)
	cards := make([]string, len(f))

	for i, fd := range f {
		if m.cursor == i {
			cards[i] = m.renderActiveCard(*fd)
			continue
		}
		cards[i] = m.renderInactiveCard(*fd)
	}

	c := lipgloss.JoinVertical(lipgloss.Top, cards...)
	m.vp.SetContent(c)
}

func (m downloadModel) filterByCurrentState(d []*fileDownload) []*fileDownload {
	s := downloadState(m.tabIdx)
	if s == all {
		return d
	}
	f := make([]*fileDownload, 0, len(d))
	for _, fd := range d {
		if fd.state == s {
			f = append(f, fd)
		}
	}
	return f
}

func (m downloadModel) renderActiveCard(fd fileDownload) string {
	w := m.vp.Width - m.vp.Style.GetHorizontalFrameSize() - downloadCardContainerStyle.GetHorizontalBorderSize() // container Width
	c := downloadCardContainerStyle.BorderForeground(highlightColor).Width(w)
	w = (c.GetWidth() / 2) - downloadCardProgressStyle.GetHorizontalFrameSize() // max allowed width for progress bar

	progressBar := constructProgressStatus(fd.prog)
	if lipgloss.Width(progressBar) > w {
		progressBar = runewidth.Truncate(progressBar, w, "…")
	}
	progressBar = downloadCardProgressStyle.Render(progressBar)

	// remaining width for file name & -1 to add a space b/w progress bar and file name
	w = c.GetWidth() - c.GetHorizontalBorderSize() - lipgloss.Width(progressBar) - 1
	fd.name = runewidth.Truncate(fd.name, w, "…")
	fillSpace := c.GetWidth() - c.GetHorizontalBorderSize() - lipgloss.Width(progressBar) - lipgloss.Width(fd.name)
	fillSpace = max(0, fillSpace) // positive or a zero value
	fd.name += strings.Repeat(" ", fillSpace)
	fd.name = lipgloss.NewStyle().Foreground(highlightColor).Inline(true).Italic(true).Render(fd.name)

	return c.Render(lipgloss.JoinHorizontal(lipgloss.Center, fd.name, progressBar))
}

func (m downloadModel) renderInactiveCard(fd fileDownload) string {
	w := m.vp.Width - m.vp.Style.GetHorizontalFrameSize() - downloadCardContainerStyle.GetHorizontalBorderSize() // container Width
	c := downloadCardContainerStyle.Width(w)
	w = (c.GetWidth() / 2) - downloadCardProgressStyle.GetHorizontalFrameSize() // max allowed width for progress bar

	progressBar := constructProgressStatus(fd.prog)
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
	fd.name = runewidth.Truncate(fd.name, w, "…")
	fillSpace := c.GetWidth() - c.GetHorizontalBorderSize() - lipgloss.Width(progressBar) - lipgloss.Width(fd.name)
	fillSpace = max(0, fillSpace) // positive or a zero value
	fd.name += strings.Repeat(" ", fillSpace)
	fd.name = lipgloss.NewStyle().Inline(true).Foreground(midHighlightColor).Render(fd.name)

	return c.Render(lipgloss.JoinHorizontal(lipgloss.Center, fd.name, progressBar))
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
	if downloadState(m.tabIdx) == all {
		return len(m.dm.downloads)
	}
	t := 0
	for _, fd := range m.dm.downloads {
		if fd.state == downloadState(m.tabIdx) {
			t++
		}
	}
	return t
}

func customDownloadHelp(show bool) *lipTable.Table {
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

func constructProgressStatus(prog client.Progress) string {
	d := spaceReplacer.Replace(humanize.Bytes(uint64(prog.D)))
	t := spaceReplacer.Replace(humanize.Bytes(uint64(prog.T)))
	s := spaceReplacer.Replace(humanize.Bytes(uint64(prog.S)))
	p := fmt.Sprintf("%d", int(float64(prog.D)/float64(prog.T)*100))
	p = spaceReplacer.Replace(p)
	return fmt.Sprintf("%s/%s • %s/s • %s%%", d, t, s, p)
}
