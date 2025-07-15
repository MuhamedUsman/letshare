package tui

import (
	"fmt"
	"github.com/MuhamedUsman/letshare/internal/client"
	"github.com/MuhamedUsman/letshare/internal/tui/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	lipTable "github.com/charmbracelet/lipgloss/table"
	"github.com/dustin/go-humanize"
	"github.com/mattn/go-runewidth"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"
)

type fileIndex struct {
	name, ext, size string
	accessID        uint32
	selection       bool
}

type fileIndexes struct {
	indexes  []fileIndex
	filtered []int
}

type extReceiveModel struct {
	client                                 *client.Client
	instance, fetchFailedStatus            string
	extFileIndexTable                      table.Model
	filter                                 textinput.Model
	titleStyle                             lipgloss.Style
	filterState                            filterState
	files                                  fileIndexes
	showHelp, disableKeymap                bool
	allSelected, isFetching, filterChanged bool
}

func initialExtReceiveModel() extReceiveModel {
	t := table.New(
		table.WithStyles(customTableStyles),
		table.WithColumns(getTableCols(0)),
	)
	return extReceiveModel{
		client:            client.Get(),
		extFileIndexTable: t,
		filter:            newFilterInputModel(),
		titleStyle:        titleStyle,
		disableKeymap:     true,
	}
}

func (m extReceiveModel) capturesKeyEvent(msg tea.KeyMsg) bool {
	// textInput model captures keys, so we need to handle it here
	if m.filterState == filtering && msg.String() != "ctrl+c" {
		return true
	}
	switch msg.String() {
	case "enter", "ctrl+s":
		return m.filterState == filtering ||
			(m.isValidTableShortcut() && m.filterState != filtering && m.getSelectionCount() > 0)
	case "up", "down", "?", "ctrl+a":
		return true
	case "/", "shift+up", "shift+down":
		return m.isValidTableShortcut()
	case "esc":
		return m.filterState != unfiltered || m.getSelectionCount() > 0
	default:
		return false
	}
}

func (m extReceiveModel) Init() tea.Cmd {
	return nil
}

func (m extReceiveModel) Update(msg tea.Msg) (extReceiveModel, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.updateDimensions()

	case tea.KeyMsg:
		if m.disableKeymap {
			return m, nil
		}
		switch msg.String() {

		case "enter":
			if m.filterState == filtering {
				m.filterState = filterApplied
				m.filter.Blur()
				m.extFileIndexTable.Focus()
			} else if m.isValidTableShortcut() && m.filterState != filtering {
				m.extFileIndexTable.Focus()
				sel := m.extFileIndexTable.Cursor()
				if m.filterState != unfiltered {
					// if filtering OR filterApplied then sel changes based on filtered indices
					sel = m.files.filtered[sel]
				}
				m.files.indexes[sel].selection = !m.files.indexes[sel].selection
				m.populateTable(m.files.indexes)
			}

		case "up", "down":
			if m.filterState == filtering {
				m.applyFilter()
				return m, tea.Batch(m.handleInfoTableUpdate(msg))
			}

		case "shift+down": // select a row and move down
			if m.isValidTableShortcut() {
				m.selectSingle(true)
				m.extFileIndexTable.MoveDown(1)
			}

		case "shift+up": // undo selection & move up
			if m.isValidTableShortcut() {
				m.selectSingle(false)
				m.extFileIndexTable.MoveUp(1)
			}

		case "ctrl+a":
			m.allSelected = !m.allSelected
			m.selectAll(m.allSelected)

		case "ctrl+s":
			if m.filterState != filtering && m.getSelectionCount() > 0 {
				return m, m.confirmDownload()
			}

		case "/":
			if m.isValidTableShortcut() {
				m.filterState = filtering
				m.extFileIndexTable.Blur()
				return m, m.filter.Focus()
			}

		case "?":
			if m.filterState != filtering {
				m.showHelp = !m.showHelp
				m.updateDimensions()
			}

		case "esc":
			if m.filterState != unfiltered {
				m.resetFilter()
				m.extFileIndexTable.Focus()
				m.populateTable(m.files.indexes)
			} else if m.getSelectionCount() > 0 {
				return m, m.confirmDiacardSel(home)
			}
		}

	case resetExtFileIndexTableSelectionsMsg:
		m.allSelected = false
		m.selectAll(m.allSelected)

	case spaceFocusSwitchMsg:
		if currentFocus == extension {
			m.updateTitleStyleAsFocus(true)
			m.extFileIndexTable.Focus()
		} else {
			m.updateTitleStyleAsFocus(false)
			m.resetFilter()
			m.extFileIndexTable.Blur()
		}

	case fetchFileIndexesMsg:
		m.instance = string(msg)
		m.isFetching = true
		m.clearFiles()
		return m, m.fetchFileIndexes()

	case fileIndexesMsg:
		m.isFetching = false
		m.files.indexes = msg
		m.fetchFailedStatus = ""
		m.populateTable(m.files.indexes)
		m.extFileIndexTable.GotoTop()

	case fetchFileFailedMsg:
		m.isFetching = false
		m.fetchFailedStatus = msg.status
		m.clearFiles()
		return m, tea.Batch(msgToCmd(msg.errMsg))
	}

	if m.filterState == filtering {
		m.handleFiltering()
	}

	return m, tea.Batch(m.handleInfoTableUpdate(msg), m.handleFilterInputUpdate(msg))
}

func (m extReceiveModel) View() string {
	help := customExtReceiveTableHelp(m.showHelp)
	help.Width(m.extFileIndexTable.Width() - 2)
	title := "Extended Instance: " + m.instance
	tail := "…"
	w := largeContainerW() - (lipgloss.Width(tail) + titleStyle.GetHorizontalPadding() + lipgloss.Width(tail))
	title = runewidth.Truncate(title, w, tail)
	title = m.titleStyle.Render(title)

	status := m.getStatus()
	status = extStatusBarStyle.Render(status)

	if m.filter.Focused() {
		filter := m.filter.View()
		c := extDirNavTableFilterContainerStyle.Width(m.filter.Width)
		// Match container width to input field width, but increment by 1 when text exceeds
		// container width to accommodate cursor. This prevents initial centering issues.
		if utf8.RuneCountInString(m.filter.Value()) >= c.GetWidth() {
			c = c.Width(c.GetWidth() + 1)
		}
		return lipgloss.JoinVertical(lipgloss.Center, c.Render(filter), status, m.extFileIndexTable.View(), help.Render())
	}
	return lipgloss.JoinVertical(lipgloss.Center, title, status, m.extFileIndexTable.View(), help.Render())
}

func (m *extReceiveModel) updateDimensions() {
	w := largeContainerW() - largeContainerStyle.GetHorizontalFrameSize()
	m.extFileIndexTable.SetWidth(w + 2)
	m.filter.Width = (w * 60) / 100 // 60% of available width
	helpHeight := lipgloss.Height(customExtReceiveTableHelp(m.showHelp).String())
	statusBarHeight := extStatusBarStyle.GetHeight() + extStatusBarStyle.GetVerticalFrameSize()
	titleHeight := m.titleStyle.GetHeight() + m.titleStyle.GetVerticalFrameSize()
	m.extFileIndexTable.SetHeight(extContainerWorkableH() - (titleHeight + statusBarHeight + helpHeight))
	m.extFileIndexTable.SetColumns(getTableCols(m.extFileIndexTable.Width()))
}

func (m *extReceiveModel) updateTitleStyleAsFocus(focus bool) {
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

func (m *extReceiveModel) handleInfoTableUpdate(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.extFileIndexTable, cmd = m.extFileIndexTable.Update(msg)
	return cmd
}

// handle update while also evaluating if the filter is changed
func (m *extReceiveModel) handleFilterInputUpdate(msg tea.Msg) tea.Cmd {
	newModel, cmd := m.filter.Update(msg)
	m.filterChanged = m.filter.Value() != newModel.Value()
	m.filter = newModel
	return cmd
}

func (m *extReceiveModel) populateTable(idx []fileIndex) {
	// case of filtering && there is some input to filter against
	if m.filterState != unfiltered && utf8.RuneCountInString(m.filter.Value()) > 0 {
		rows := make([]table.Row, 0, len(idx))
		for _, i := range m.files.filtered {
			sel := ""
			if idx[i].selection {
				sel = "✓"
			}
			rows = append(rows, table.Row{sel, idx[i].name, idx[i].ext, idx[i].size})
		}
		m.extFileIndexTable.SetRows(rows)
		return
	}
	// case of unfiltered
	rows := make([]table.Row, len(idx))
	for i, content := range idx {
		sel := ""
		if content.selection {
			sel = "✓"
		}
		rows[i] = table.Row{sel, content.name, content.ext, content.size}
	}
	m.extFileIndexTable.SetRows(rows)
}

func (m extReceiveModel) isValidTableShortcut() bool {
	return currentFocus == extension && m.extFileIndexTable.Focused() && len(m.extFileIndexTable.Rows()) > 0
}

func (m *extReceiveModel) selectAll(selection bool) {
	if m.isValidTableShortcut() {
		if m.filterState != unfiltered {
			for _, i := range m.files.filtered {
				m.files.indexes[i].selection = selection
			}
		} else {
			for i := range m.files.indexes {
				m.files.indexes[i].selection = selection
			}
		}
		m.populateTable(m.files.indexes)
	}
}

func (m *extReceiveModel) selectSingle(selection bool) {
	sel := m.extFileIndexTable.Cursor()
	// ensure sel is not negative, panic occurred in testing, but I am not setting the cursor to -1
	sel = max(0, sel)
	if m.filterState != unfiltered {
		sel = m.files.filtered[sel]
	}
	m.files.indexes[sel].selection = selection
	m.populateTable(m.files.indexes)
}

func (m *extReceiveModel) resetFilter() {
	m.filter.Reset()
	m.filter.Blur()
	m.filterState = unfiltered
}

func (m *extReceiveModel) applyFilter() {
	m.filter.Blur()
	m.filterState = filterApplied
	m.extFileIndexTable.Focus()
}

func (m *extReceiveModel) handleFiltering() tea.Cmd {
	if !m.filterChanged {
		return nil
	}
	m.extFileIndexTable.SetCursor(0) // reset cursor, if filter changed
	toFilter := make([]string, len(m.files.indexes))
	for i, content := range m.files.indexes {
		toFilter[i] = content.name
	}
	indices := filter(m.filter.Value(), toFilter)
	m.files.filtered = indices
	m.populateTable(m.files.indexes)
	return nil
}

func (m extReceiveModel) getStatus() string {
	status := "Fetching files, please wait..."
	if m.isFetching {
		return runewidth.Truncate(status, largeContainerW()-4, "…")
	}
	if m.fetchFailedStatus != "" {
		return runewidth.Truncate(m.fetchFailedStatus, largeContainerW()-4, "…")
	}
	// unfiltered
	status = fmt.Sprintf("%d Total", len(m.files.indexes))
	selectCount := m.getSelectionCount()
	if selectCount > 0 {
		status = fmt.Sprintf("%d Selected • %d Total", selectCount, len(m.files.indexes))
	}
	if utf8.RuneCountInString(m.filter.Value()) == 0 {
		return runewidth.Truncate(status, largeContainerW()-4, "…") // -4 for tail, extensionContainer & statusBar frame size
	}
	matches := "Nothing matched"
	filtered := len(m.files.indexes)
	if len(m.files.filtered) > 0 {
		matches = fmt.Sprint(len(m.files.filtered), " ", "Match/es")
	}
	status = fmt.Sprintf("%s • %d Filtered", matches, filtered)
	if selectCount > 0 {
		status = fmt.Sprintf("%s • %d Selected • %d Filtered", matches, selectCount, filtered)
	}
	if m.filterState == filterApplied {
		status = fmt.Sprintf("“%s” %s", m.filter.Value(), status)
	}
	return runewidth.Truncate(status, largeContainerW()-4, "…") // -4 for tail, extensionContainer & statusBar frame size
}

func (m extReceiveModel) getSelectionCount() int {
	count := 0
	for _, content := range m.files.indexes {
		if content.selection {
			count++
		}
	}
	return count
}

func (m extReceiveModel) selectedFilesAsDownloadMsg() []downloadSelection {
	files := make([]downloadSelection, 0, len(m.files.indexes))
	for _, idx := range m.files.indexes {
		if !idx.selection {
			continue
		}
		if idx.ext != "---" && idx.ext != "" {
			idx.name = fmt.Sprintf("%s.%s", idx.name, idx.ext)
		}
		ds := downloadSelection{
			name:     idx.name,
			size:     idx.size,
			accessID: idx.accessID,
		}
		files = append(files, ds)
	}
	return files
}

func (m *extReceiveModel) confirmDiacardSel(space extChild) tea.Cmd {
	// when filtering, we will not grant an extension switch
	if m.filterState != unfiltered {
		return nil
	}
	cmd := msgToCmd(extensionChildSwitchMsg{space, true})
	if m.getSelectionCount() == 0 {
		return cmd
	}
	selBtn := positive
	header := "ARE YOU SURE?"
	body := "All the selections will be lost..."
	positiveFunc := func() tea.Cmd {
		m.allSelected = false
		m.selectAll(m.allSelected)
		return cmd
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

func (m *extReceiveModel) confirmDownload() tea.Cmd {
	sd := m.selectedFilesAsDownloadMsg() // selected downloads
	selBtn := positive
	header := "PROCEED?"
	body := fmt.Sprintf(`Selected “%d file/s” will be downloaded as per preferences. To change preferences, press “esc” & “ctrl+p”.`, len(sd))
	positiveFunc := func() tea.Cmd {
		return tea.Batch(
			msgToCmd(resetExtFileIndexTableSelectionsMsg{}),
			msgToCmd(downloadSelectionsMsg{m.instance, sd}),
			msgToCmd(extensionChildSwitchMsg{download, true}),
		)
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

func (m *extReceiveModel) updateKeymap(disable bool) {
	m.disableKeymap = disable
}

func customExtReceiveTableHelp(show bool) *lipTable.Table {
	baseStyle := lipgloss.NewStyle()
	var rows [][]string
	if !show {
		rows = [][]string{{"?", "help"}}
	} else {
		rows = [][]string{
			{"enter", "select/deselect at cursor"},
			{"shift+↓/↑", "make/undo selection"},
			{"ctrl+a", "select/deselect all"},
			{"ctrl+s", "send selected files"},
			{"esc", "exit filtering"},
			{"/", "filter"},
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

func (m *extReceiveModel) clearFiles() {
	m.files = fileIndexes{}
	m.populateTable(m.files.indexes)
}

func (m extReceiveModel) fetchFileIndexes() tea.Cmd {
	return func() tea.Msg {
		fInfos, status, err := m.client.IndexFiles(m.instance)
		if err != nil {
			return errMsg{err: err, fatal: true}
		}
		if status == http.StatusRequestTimeout {
			return fetchFileFailedMsg{
				status: "Fetching files failed, you may want to retry…",
				errMsg: errMsg{
					errHeader: strings.ToUpper(http.StatusText(http.StatusRequestTimeout)),
					errStr:    "Fetching files, the server instance is not responding, it might be down.",
					fatal:     false,
				},
			}
		} else if status != http.StatusOK {
			// TODO: it failed now update the status bar to show error and set the receiveModel fetchInitiated to false
			h := strings.ToUpper(http.StatusText(status))
			if h == "" {
				h = "UNKNOWN ERROR"
			}
			return fetchFileFailedMsg{
				status: "Fetching files failed, you may want to retry…",
				errMsg: errMsg{
					errHeader: h,
					errStr: fmt.Sprintf(
						"server returned status code %q while indexing directory.",
						strconv.Itoa(status),
					),
				},
			}
		}

		indexes := make([]fileIndex, len(fInfos))
		for i, f := range fInfos {
			ext := filepath.Ext(f.Name)
			if ext == f.Name { // .gitignore or similar files
				ext = ""
			}
			name := strings.TrimSuffix(f.Name, ext) // remove the extension from the name
			if ext == "" {
				ext = "---"
			} else {
				ext = strings.TrimPrefix(ext, ".") // remove the leading dot
			}
			indexes[i] = fileIndex{
				name:     name,
				ext:      ext,
				accessID: f.AccessID,
				size:     humanize.Bytes(uint64(f.Size)),
			}
		}
		return fileIndexesMsg(indexes)
	}
}
