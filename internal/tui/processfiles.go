package tui

import (
	"context"
	"errors"
	"fmt"
	"github.com/MuhamedUsman/letshare/internal/bgtask"
	"github.com/MuhamedUsman/letshare/internal/config"
	"github.com/MuhamedUsman/letshare/internal/zipr"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/dustin/go-humanize"
	"github.com/mattn/go-runewidth"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type selections struct {
	rootPath    string
	filenames   []string
	dirs, files int
}

type zippingState int

const (
	processing zippingState = iota
	canceling
	canceled
	done
)

type zipTracker struct {
	ctx                                 context.Context
	cancel                              context.CancelFunc
	prevUpdateTime, start               time.Time
	timeTaken                           time.Duration
	logs, archives                      []string
	progressCh                          <-chan uint64
	logCh                               <-chan string
	totalSize, processed                uint64
	processedInPrevSec, processedPerSec uint64
	viewableLogs                        int
	isTotalSize                         bool
	state                               zippingState
}

func newZipTracker(parentCtx context.Context, p <-chan uint64, l <-chan string) *zipTracker {
	ctx, cancel := context.WithCancel(parentCtx)
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

type processFilesModel struct {
	selections             *selections
	zipTracker             *zipTracker
	progress               progress.Model
	titleStyle             lipgloss.Style
	showProgress, showHelp bool
	disableKeymap          bool
}

func initialProcessFilesModel() processFilesModel {
	p := progress.New(
		progress.WithGradient(subduedHighlightColor.Dark, highlightColor.Dark),
		progress.WithoutPercentage(),
	)
	return processFilesModel{
		zipTracker: new(zipTracker),
		titleStyle: titleStyle.Margin(0, 2),
		progress:   p,
	}
}

func (m processFilesModel) capturesKeyEvent(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "Q", "q":
		return m.zipTracker.state == processing
	case "?":
		return !m.disableKeymap
	default:
		return false
	}
}

func (m processFilesModel) Init() tea.Cmd {
	return m.progress.Init()
}

func (m processFilesModel) Update(msg tea.Msg) (processFilesModel, tea.Cmd) {

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.updateDimensions()

	case tea.KeyMsg:
		if m.disableKeymap {
			return m, nil
		}
		switch msg.String() {

		case "Q", "q":
			return m, m.confirmStopZipping(false)

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
				Background(grayColor).
				Foreground(highlightColor)
		}

	case processSelectionsMsg:
		m.selections = &selections{
			rootPath:  msg.parentPath,
			filenames: msg.filenames,
			dirs:      msg.dirs,
			files:     msg.files,
		}

		shutdownCtx := bgtask.Get().ShutdownCtx()
		progCh := make(chan uint64, 1)
		logCh := make(chan string)
		m.zipTracker = newZipTracker(shutdownCtx, progCh, logCh)
		m.updateDimensions() // update the logs length

		cfg, err := config.Get()
		if errors.Is(err, config.ErrNoConfig) {
			cfg, err = config.Load()
		}
		if err != nil {
			return m, errMsg{err: err, fatal: true}.cmd
		}

		if !cfg.Share.ZipFiles && m.selections.dirs == 0 {
			files := make(sendFilesMsg, len(m.selections.filenames))
			for i, f := range m.selections.filenames {
				files[i] = filepath.Join(m.selections.rootPath, f)
			}
			return m, tea.Batch(localChildSwitchMsg{child: send, focus: true}.cmd, files.cmd)
		}

		return m, tea.Batch(
			m.trackProgress(),
			m.trackLogs(),
			m.processFiles(cfg, progCh, logCh, msg),
			localChildSwitchMsg{child: processFiles, focus: true}.cmd,
		)

	case progressMsg:
		m.zipTracker.updateProgress(uint64(msg))
		percentage := float64(m.zipTracker.processed) / float64(m.zipTracker.totalSize)
		return m, tea.Batch(m.trackProgress(), m.progress.SetPercent(percentage))

	case zippingLogMsg:
		l := filepath.Base(string(msg))
		m.zipTracker.appendLog(l)
		return m, m.trackLogs()

	case zippingDoneMsg:
		m.zipTracker.archives = msg
		m.zipTracker.processed = m.zipTracker.totalSize
		m.zipTracker.state = done
		m.updateDimensions()
		return m, tea.Batch(localChildSwitchMsg{child: send, focus: true}.cmd, sendFilesMsg(m.zipTracker.archives).cmd)

	case zippingCanceledMsg:
		m.zipTracker.state = canceled
		return m, localChildSwitchMsg{child: dirNav, focus: true}.cmd

	case zippingErrMsg:
		return m, tea.Batch(m.showZippingErrAlert(msg), localChildSwitchMsg{child: dirNav, focus: true}.cmd)

	}

	return m, m.handleProgressModelUpdate(msg)
}

func (m processFilesModel) View() string {
	var v string
	components := []string{
		m.renderTitle(),
		m.renderStatusBar(),
		m.renderLogsTitle(),
		m.renderLogs(),
		m.renderProgress(),
		customProcessFilesTable(m.showHelp).Width(smallContainerW() - 2).Render(),
	}
	v = lipgloss.JoinVertical(lipgloss.Top, components...)
	return smallContainerStyle.Width(smallContainerW()).Render(v)
}

func (m *processFilesModel) handleProgressModelUpdate(msg tea.Msg) tea.Cmd {
	newModel, cmd := m.progress.Update(msg)
	m.progress = newModel.(progress.Model)
	return cmd
}

func (m *processFilesModel) updateKeymap(disable bool) {
	m.disableKeymap = disable
}

func (m *processFilesModel) updateDimensions() {
	m.progress.Width = smallContainerW() - smallContainerStyle.GetHorizontalFrameSize()
	subH := smallContainerStyle.GetVerticalFrameSize() + zipLogsContainerStyle.GetVerticalFrameSize()
	components := []string{
		m.renderTitle(),
		m.renderStatusBar(),
		m.renderLogsTitle(),
		m.renderProgress(),
		customProcessFilesTable(m.showHelp).String(),
	}
	subH += lipgloss.Height(strings.Join(components, "\n"))
	m.zipTracker.setLogsLength(max(0, workableH()-subH))
}

func (m processFilesModel) renderTitle() string {
	subW := smallContainerW() - m.titleStyle.GetHorizontalFrameSize() - 2
	t := runewidth.Truncate("Local Space", subW, "…")
	return m.titleStyle.Render(t)
}

func (m processFilesModel) renderStatusBar() string {
	processedPerSec := humanize.Bytes(m.zipTracker.processedPerSec)
	s := fmt.Sprintf("Processsing at %s/s", processedPerSec)
	if m.zipTracker.state == canceling {
		s = "Canceling, may take a while!"
	}
	if m.zipTracker.state == canceled {
		s = "Processing Canceled!"
	}
	if m.zipTracker.state == done {
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

func (m processFilesModel) renderLogsTitle() string {
	t := "Zipping Files"
	t = runewidth.Truncate(t, smallContainerW()-titleStyle.GetHorizontalFrameSize()-2, "…")
	return titleStyle.Background(subduedHighlightColor).
		Width(smallContainerW() - titleStyle.GetHorizontalFrameSize()).
		MarginTop(1).
		Align(lipgloss.Center).
		UnsetItalic().
		Render(t)
}

func (m processFilesModel) renderLogs() string {
	logs := make([]string, m.zipTracker.viewableLogs)
	gradient := generateGradient(subduedHighlightColor, highlightColor, m.zipTracker.viewableLogs)
	logStyle := lipgloss.NewStyle().Italic(true)

	for i := range logs {
		logStyle = logStyle.Foreground(gradient[i])
		vi := m.zipTracker.viewableLogs - 1 - i // viewable logs are reversed
		l := m.zipTracker.logs[vi]
		l = runewidth.Truncate(l, smallContainerW()-2, "…")
		if m.zipTracker.state == canceled {
			logStyle = logStyle.Strikethrough(true)
		}
		logs[i] = logStyle.Render(l)
	}

	return zipLogsContainerStyle.
		Width(smallContainerW()).
		Render(strings.Join(logs, "\n"))
}

func (m processFilesModel) renderProgress() string {
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
			if m.zipTracker.state == canceled {
				baseStyle = baseStyle.Strikethrough(true)
			}
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

func (m *processFilesModel) processFiles(
	cfg config.Config,
	progCh chan<- uint64,
	logCh chan<- string,
	msg processSelectionsMsg,
) tea.Cmd {

	var archives []string
	algo := zipr.Store
	if cfg.Share.Compression {
		algo = zipr.Deflate
	}

	return func() tea.Msg {
		zipper := zipr.New(progCh, logCh, algo)
		defer func() { _ = zipper.Close() }()
		var err error

		bgtask.Get().RunAndBlock(func(_ context.Context) {
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
		})

		// if the zipping is canceled, we need to wait for the cancellation to finish
		if m.zipTracker.state == canceling {
			<-m.zipTracker.ctx.Done()
			return zippingCanceledMsg{}
		}
		if err != nil {
			return zippingErrMsg(err)
		}
		return zippingDoneMsg(archives)
	}
}

func (m processFilesModel) trackProgress() tea.Cmd {
	return func() tea.Msg {
		for p := range m.zipTracker.progressCh {
			return progressMsg(p)
		}
		return nil
	}
}

func (m processFilesModel) trackLogs() tea.Cmd {
	return func() tea.Msg {
		for l := range m.zipTracker.logCh {
			return zippingLogMsg(l)
		}
		return nil
	}
}

func (m *processFilesModel) confirmStopZipping(quit bool) tea.Cmd {
	selBtn := positive
	header := "STOP ZIPPING?"
	body := "Do you want to stop zipping the files, progress will be lost."
	positiveFunc := func() tea.Cmd {
		m.zipTracker.state = canceling
		m.zipTracker.cancel()
		if quit {
			shutdownBgTasks()
			return tea.Quit
		}
		return nil
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

func (m processFilesModel) showZippingErrAlert(err error) tea.Cmd {
	var b string
	var pe *os.PathError
	if errors.As(err, &pe) {
		switch {
		case errors.Is(pe.Err, os.ErrPermission):
			b = fmt.Sprintf("Permission denied for %q. You may run this app with higher privileges.", filepath.ToSlash(pe.Path))
		case errors.Is(pe.Err, os.ErrNotExist):
			b = fmt.Sprintf("File %q does not exist.", filepath.ToSlash(pe.Path))
		default:
			b = fmt.Sprintf("Unexpected error occurred while accessing %q.", filepath.ToSlash(pe.Path))
		}
	} else {
		b = "Unexpected error occurred while zipping files."
	}
	return alertDialogMsg{header: "ZIPPING ERROR", body: b}.cmd
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

func customProcessFilesTable(show bool) *table.Table {
	baseStyle := lipgloss.NewStyle()
	var rows [][]string
	if !show {
		rows = [][]string{{"?", "help"}}
	} else {
		rows = [][]string{
			{"Q/q", "stop processing"},
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
