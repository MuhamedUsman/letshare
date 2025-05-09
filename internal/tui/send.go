package tui

import (
	"context"
	"errors"
	"fmt"
	"github.com/MuhamedUsman/letshare/internal/client"
	"github.com/MuhamedUsman/letshare/internal/zipr"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/dustin/go-humanize"
	"github.com/mattn/go-runewidth"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type selections struct {
	rootPath    string
	filenames   []string
	dirs, files int
}

type zipTracker struct {
	ctx                context.Context
	cancel             context.CancelFunc
	prevUpdateTime     time.Time
	start              time.Time
	timeTaken          time.Duration
	logs               []string
	progressCh         <-chan uint64
	logCh              <-chan string
	totalSize          uint64
	processed          uint64
	processedInPrevSec uint64
	processedPerSec    uint64
	viewableLogs       int
	isTotalSize        bool
	done               bool
}

func newZipTracker(p <-chan uint64, l <-chan string) *zipTracker {
	ctx, cancel := context.WithCancel(context.Background())
	return &zipTracker{
		ctx:         ctx,
		cancel:      cancel,
		isTotalSize: true,
		logs:        make([]string, 0),
		progressCh:  p,
		logCh:       l,
	}
}

func (z *zipTracker) updateProgress(c uint64) {
	now := time.Now()

	if time.Since(z.prevUpdateTime) >= time.Second {
		bytesDelta := c - z.processedInPrevSec
		secondsElapsed := now.Sub(z.prevUpdateTime).Seconds()
		z.processedPerSec = uint64(float64(bytesDelta) / secondsElapsed)

		z.prevUpdateTime = time.Now()
		z.processedInPrevSec = c
	}
	if z.isTotalSize {
		z.start = time.Now()
		z.totalSize = c
		z.isTotalSize = false
	} else {
		z.timeTaken = now.Sub(z.start)
		z.processed = c
	}
}

func (z *zipTracker) appendLog(l string) {
	copy(z.logs[1:], z.logs[:len(z.logs)-1])
	z.logs[0] = l
}

func (z *zipTracker) setLogsLength(l int) {
	z.viewableLogs = max(0, l)
	if l <= len(z.logs) {
		return // no need to undersize
	}
	newLogs := make([]string, l)
	copy(newLogs, z.logs)
	z.logs = newLogs
}

func (z *zipTracker) markDone() {
	z.cancel()
	z.ctx, z.cancel, z.progressCh, z.logCh, z.logCh = nil, nil, nil, nil, nil
	z.done = true
}

type sendModel struct {
	selections              *selections
	zipTracker              *zipTracker
	btnIndex                int
	progress                progress.Model
	titleStyle              lipgloss.Style
	showProgress, showHelp  bool
	isActive, disableKeymap bool
}

func (m sendModel) capturesKeyEvent(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "ctrl+c":
		return !m.zipTracker.done
	case "left", "right", "h", "l", "tab", "shift+tab":
		return m.isActive
	default:
		return false
	}
}

func initialSendModel() sendModel {
	p := progress.New(
		progress.WithGradient(subduedHighlightColor.Dark, highlightColor.Dark),
		progress.WithoutPercentage(),
	)
	return sendModel{
		zipTracker: new(zipTracker), // so we don't get nil pointers
		titleStyle: titleStyle.Margin(0, 2),
		progress:   p,
	}
}

func (m sendModel) Init() tea.Cmd {
	return m.progress.Init()
}

func (m sendModel) Update(msg tea.Msg) (sendModel, tea.Cmd) {

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.updateDimensions()

	case tea.KeyMsg:
		if m.disableKeymap {
			return m, nil
		}
		switch msg.String() {

		case "ctrl+c":
			return m, m.confirmStopZipping()

		case "left", "h":
			m.btnIndex = 0

		case "right", "l":
			m.btnIndex = 1

		case "tab":
			m.btnIndex = (m.btnIndex + 1) % 2

		case "shift+tab":
			m.btnIndex = (m.btnIndex - 1 + 2) % 2

		case "?":
			m.showHelp = !m.showHelp
			m.updateDimensions()

		}

	case spaceFocusSwitchMsg:
		if currentFocus == local {
			m.titleStyle = m.titleStyle.
				Background(highlightColor).
				Foreground(subduedHighlightColor)
		} else {
			m.titleStyle = m.titleStyle.
				Background(subduedGrayColor).
				Foreground(highlightColor)
		}

	case processSelectionsMsg:
		m.selections = &selections{
			rootPath:  msg.parentPath,
			filenames: msg.filenames,
			dirs:      msg.dirs,
			files:     msg.files,
		}

		progCh := make(chan uint64, 1)
		logCh := make(chan string, 10)
		m.zipTracker = newZipTracker(progCh, logCh)
		m.updateDimensions() // update the logs length

		return m, tea.Batch(
			m.trackProgress(),
			m.trackLogs(),
			m.processFiles(progCh, logCh, msg),
			localChildSwitchMsg{child: send, focus: true}.cmd,
		)

	case progressMsg:
		m.zipTracker.updateProgress(uint64(msg))
		percentage := float64(m.zipTracker.processed) / float64(m.zipTracker.totalSize)
		return m, tea.Batch(m.trackProgress(), m.progress.SetPercent(percentage))

	case logMsg:
		l := filepath.Base(string(msg))
		m.zipTracker.appendLog(l)
		return m, m.trackLogs()

	case zippingDoneMsg:
		m.zipTracker.markDone()
		m.updateDimensions()

	case zippingErr:
		//m.zipTracker.markDone()
		log.Println(msg.Error())

	}

	return m, m.handleProgressModelUpdate(msg)
}

func (m sendModel) View() string {
	var v string
	components := []string{
		m.renderTitle(),
		m.renderStatusBar(),
		m.renderLogsTitle(),
		m.renderLogs(),
		m.renderProgress(),
		customSendHelpTable(m.showHelp).Width(smallContainerW() - 2).Render(),
	}
	if m.zipTracker.done {
		components = slices.Insert(components, 5, m.renderBtns())
		m.updateDimensions()
	}
	v = lipgloss.JoinVertical(lipgloss.Top, components...)
	return smallContainerStyle.Width(smallContainerW()).Render(v)
}

func (m *sendModel) handleProgressModelUpdate(msg tea.Msg) tea.Cmd {
	newModel, cmd := m.progress.Update(msg)
	m.progress = newModel.(progress.Model)
	return cmd
}

func (m *sendModel) updateKeymap(disable bool) {
	m.disableKeymap = disable
}

func (m *sendModel) updateDimensions() {
	m.progress.Width = smallContainerW() - smallContainerStyle.GetHorizontalFrameSize()
	subH := smallContainerStyle.GetVerticalFrameSize() + zipLogsContainerStyle.GetVerticalFrameSize()
	components := []string{
		m.renderTitle(),
		m.renderStatusBar(),
		m.renderLogsTitle(),
		m.renderProgress(),
		customSendHelpTable(m.showHelp).String(),
	}
	subH += lipgloss.Height(strings.Join(components, "\n"))
	if m.zipTracker.done {
		subH += lipgloss.Height(m.renderBtns())
	}
	m.zipTracker.setLogsLength(max(0, workableH()-subH))
}

func (m sendModel) renderTitle() string {
	subW := smallContainerW() - m.titleStyle.GetHorizontalFrameSize() - 2
	t := runewidth.Truncate("Local Space", subW, "…")
	return m.titleStyle.Render(t)
}

func (m sendModel) renderStatusBar() string {
	processedPerSec := humanize.Bytes(m.zipTracker.processedPerSec)
	s := fmt.Sprintf("Processsing at %s/s", processedPerSec)
	if m.zipTracker.done {
		s = fmt.Sprintf("Processed in %s", m.zipTracker.timeTaken.Round(time.Second))
	}
	style := lipgloss.NewStyle().
		Foreground(highlightColor).
		Faint(true).
		Margin(1, 1, 0, 2).
		Italic(true)
	s = runewidth.Truncate(s, smallContainerW()-style.GetHorizontalFrameSize()-1, "…")
	return style.Render(s)
}

func (m sendModel) renderLogsTitle() string {
	t := "Zipping Files"
	t = runewidth.Truncate(t, smallContainerW()-titleStyle.GetHorizontalFrameSize()-2, "…")
	return titleStyle.Background(subduedHighlightColor).
		Width(smallContainerW() - titleStyle.GetHorizontalFrameSize()).
		MarginTop(1).
		Align(lipgloss.Center).
		UnsetItalic().
		Render(t)
}

func (m sendModel) renderLogs() string {
	logs := make([]string, m.zipTracker.viewableLogs)
	gradient := generateGradient(subduedHighlightColor, highlightColor, m.zipTracker.viewableLogs)

	for i := range logs {
		vi := m.zipTracker.viewableLogs - 1 - i // viewable logs are reversed
		l := m.zipTracker.logs[vi]
		l = runewidth.Truncate(l, smallContainerW()-2, "…")
		logs[i] = lipgloss.NewStyle().Foreground(gradient[i]).Italic(true).Render(l)
	}

	return zipLogsContainerStyle.
		Width(smallContainerW()).
		Render(strings.Join(logs, "\n"))
}

func (m sendModel) renderProgress() string {
	currentRead := humanize.Bytes(m.zipTracker.processed)
	totalSize := humanize.Bytes(m.zipTracker.totalSize)
	progressCounter := fmt.Sprintf("%s/%s", currentRead, totalSize)
	percentage := fmt.Sprintf("%.1f%%", m.progress.Percent()*100)
	// it renders the progress information e.g:
	// 1.2GB/2.3GB                      100.0%
	p := table.New().
		Row(progressCounter, percentage).Border(lipgloss.HiddenBorder()).
		BorderTop(false).BorderLeft(false).BorderRight(false).
		Width(smallContainerW()).Wrap(false).
		StyleFunc(func(_, c int) lipgloss.Style {
			baseStyle := lipgloss.NewStyle().Foreground(highlightColor)
			switch c {
			case 0:
				return baseStyle.Align(lipgloss.Left)
			case 1:
				return baseStyle.Align(lipgloss.Right)
			default:
				return baseStyle
			}
		}).Render()
	return lipgloss.JoinVertical(lipgloss.Top, m.progress.View(), p)
}

func (m sendModel) renderBtns() string {
	btn1, btn2 := "CANCEL", "CONTINUE"
	w := (smallContainerW() - 3) / 2

	btn1 = runewidth.Truncate(btn1, w-2, "✕")
	btn2 = runewidth.Truncate(btn2, w-2, "✓")

	inactiveStyle := lipgloss.NewStyle().
		Background(subduedGrayColor).
		Foreground(highlightColor).
		Align(lipgloss.Center).
		Width(w).
		Padding(0, 1)
	activeStyle := inactiveStyle.
		Background(highlightColor).
		Foreground(subduedHighlightColor)

	mr := 1
	if smallContainerW()%2 == 0 { // is odd
		mr += 1
	}
	switch m.btnIndex {
	case 0:
		btn1 = activeStyle.MarginRight(mr).Render(btn1)
		btn2 = inactiveStyle.Render(btn2)
	case 1:
		btn1 = inactiveStyle.MarginRight(mr).Render(btn1)
		btn2 = activeStyle.Render(btn2)
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, btn1, btn2)
}

func (m *sendModel) processFiles(progCh chan<- uint64, logCh chan<- string, msg processSelectionsMsg) tea.Cmd {
	cfg, err := client.GetConfig()
	if errors.Is(err, client.ErrNoConfig) {
		cfg, err = client.LoadConfig()
	}
	if err != nil {
		return errMsg{err: err, fatal: true}.cmd
	}

	var archives []string
	algo := zipr.Store
	if cfg.Share.Compression {
		algo = zipr.Deflate
	}

	return func() tea.Msg {
		zipper := zipr.New(progCh, logCh, algo)
		defer func() { _ = zipper.Close() }()

		if cfg.Share.ZipFiles {
			var archive string
			archive, err = zipper.CreateArchive(
				m.zipTracker.ctx, // will be canceled on shutdown
				os.TempDir(),
				cfg.Share.SharedZipName,
				msg.parentPath,
				msg.filenames...,
			)
			archives = []string{archive}
		} else {
			archives, err = zipDirsAndCollectWithFiles(
				m.zipTracker.ctx,
				zipper,
				msg.parentPath,
				msg.filenames...,
			)
		}
		if err != nil {
			return zippingErr(err)
		}
		return zippingDoneMsg(archives)
	}
}

func (m sendModel) trackProgress() tea.Cmd {
	return func() tea.Msg {
		for p := range m.zipTracker.progressCh {
			return progressMsg(p)
		}
		return nil
	}
}

func (m sendModel) trackLogs() tea.Cmd {
	return func() tea.Msg {
		for l := range m.zipTracker.logCh {
			return logMsg(l)
		}
		return nil
	}
}

func (m sendModel) confirmStopZipping() tea.Cmd {
	selBtn := yup
	header := "STOP ZIPPING?"
	body := "Do you want to stop zipping the files, progress will be lost."
	yupFunc := func() tea.Cmd {
		m.zipTracker.cancel()
		return tea.Quit
	}
	return confirmDialogCmd(header, body, selBtn, yupFunc, nil, nil)
}

func zipDirsAndCollectWithFiles(ctx context.Context, zipper *zipr.Zipr, root string, filenames ...string) ([]string, error) {
	dirs, files, err := splitToDirsAndFiles(root, filenames...)
	if err != nil {
		return nil, err
	}
	archives, err := zipper.CreateArchives(ctx, os.TempDir(), root, dirs...)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		archives = append(archives, filepath.Join(root, f))
	}
	return archives, nil
}

func splitToDirsAndFiles(root string, filenames ...string) (dirs, files []string, err error) {
	for _, filename := range filenames {
		var info os.FileInfo
		info, err = os.Lstat(filepath.Join(root, filename))
		if err != nil {
			return nil, nil, err
		}
		if info.IsDir() {
			dirs = append(dirs, filename)
		} else {
			files = append(files, filename)
		}
	}
	return dirs, files, nil
}

func customSendHelpTable(show bool) *table.Table {
	baseStyle := lipgloss.NewStyle()
	var rows [][]string
	if !show {
		rows = [][]string{{"?", "help"}}
	} else {
		rows = [][]string{
			{"/", "filter"},
			{"space", "extend dir"},
			{"2x(space)", "extend dir focused"},
			{"enter", "into dir"},
			{"backspace", "out of dir"},
			{"←/→ OR l/h", "shuffle pages"},
			{"esc", "exit filtering"},
			{"?", "hide help"},
		}
	}
	return table.New().
		Border(lipgloss.HiddenBorder()).
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
