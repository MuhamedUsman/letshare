package tui

import (
	"errors"
	"fmt"
	"github.com/MuhamedUsman/letshare/internal/tui/table"
	"github.com/MuhamedUsman/letshare/internal/util"
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	lipTable "github.com/charmbracelet/lipgloss/table" // lipTable -> lipglossTable
	"github.com/mattn/go-runewidth"
	"github.com/sahilm/fuzzy"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

type filterState = int

const (
	unfiltered = iota
	filtering
	filterApplied
)

type dirContent struct {
	name, ext, size string
	selection       bool
}

type dirContents struct {
	contents []dirContent
	// indices of filtered contents,
	//if filteredState == filtering || filterApplied
	filteredContents []int
	dirs, files      int
}

// filterDirContent uses the sahilm/fuzzy to filter through the list.
// This is set by default.
func filterDirContent(term string, targets []string) []int {
	matches := fuzzy.Find(term, targets)
	result := make([]int, len(matches))
	for i, r := range matches {
		result[i] = r.Index
	}
	return result
}

type sendInfoModel struct {
	infoTable                              table.Model
	filter                                 textinput.Model
	filterState                            filterState
	dirContents                            dirContents
	dirPath                                string
	filterChanged, focusOnExtend, showHelp bool
}

func initialSendInfoModel() sendInfoModel {
	t := table.New(
		table.WithStyles(customTableStyles),
		table.WithColumns(getTableCols(0)),
	)
	return sendInfoModel{infoTable: t, filter: newFilterInputModel()}
}

func getTableCols(tableWidth int) []table.Column {
	cols := []string{"✓", "Name", "Type", "Size"}
	subW := customTableStyles.Cell.GetHorizontalFrameSize() * len(cols)
	tableWidth -= subW
	selectionW := 1
	tableWidth -= selectionW
	nameW := (tableWidth * 62) / 100
	typeW := (tableWidth * 18) / 100
	sizeW := (tableWidth * 20) / 100
	// terminals have rows x cols as integer values
	// this condition is to take into account the
	// precision loss from above division ops
	nameW += int(math.Abs(float64(tableWidth - (sizeW + typeW + nameW))))
	return []table.Column{
		{cols[0], selectionW},
		{cols[1], nameW},
		{cols[2], typeW},
		{cols[3], sizeW},
	}
}

func (m sendInfoModel) Init() tea.Cmd {
	return nil
}

func (m sendInfoModel) Update(msg tea.Msg) (sendInfoModel, tea.Cmd) {

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.updateDimensions()

	case tea.KeyMsg:
		switch msg.String() {

		case "enter":
			if m.filterState == filtering {
				m.filterState = filterApplied
				m.filter.Blur()
				m.infoTable.Focus()
				return m, hideInfoSpaceTitle(false).cmd
			}
			if m.isValidTableShortcut() && m.filterState != filtering {
				m.infoTable.Focus()
				sel := m.infoTable.Cursor()
				if m.filterState != unfiltered {
					// if filtering OR filterApplied then sel changes based on filtered indices
					sel = m.dirContents.filteredContents[sel]
				}
				m.dirContents.contents[sel].selection = !m.dirContents.contents[sel].selection
				m.populateTable(m.dirContents.contents)
			}

		case "up", "down":
			if m.filterState == filtering {
				return m, tea.Batch(m.handleInfoTableUpdate(msg), m.applyFilter())
			}

		case "shift+down", "ctrl+down": // select row & move down
			if m.isValidTableShortcut() {
				m.selectSingle(true)
				m.infoTable.MoveDown(1)
			}

		case "shift+up", "ctrl+up": // undo selection & move up
			if m.isValidTableShortcut() {
				m.selectSingle(false)
				m.infoTable.MoveUp(1)
			}

		case "ctrl+a":
			m.selectAll(true)

		case "ctrl+z":
			m.selectAll(false)

		case "/":
			if m.isValidTableShortcut() {
				m.filterState = filtering
				m.infoTable.Blur()
				return m, tea.Batch(m.filter.Focus(), hideInfoSpaceTitle(true).cmd)
			}

		case "?":
			if currentFocus == info && m.filterState != filtering {
				m.showHelp = !m.showHelp
				m.updateDimensions()
			}

		case "esc":
			if m.filterState != unfiltered {
				m.resetFilter()
				m.infoTable.Focus()
				m.populateTable(m.dirContents.contents)
			}
			return m, hideInfoSpaceTitle(false).cmd

		case "backspace":
			if m.getSelectionCount() > 0 {
				return m, confirmDialogMsg{"ARE YOU SURE?", "All the selections will be lost..."}.cmd
			}

		}

	case extendDirMsg:
		m.focusOnExtend = msg.focus
		return m, m.readDir(msg.path)

	case dirContents:
		m.dirContents = msg
		m.populateTable(msg.contents)
		if m.focusOnExtend {
			m.infoTable.Focus()
		} else {
			m.infoTable.Blur()
		}
		// if table is focused, then info space also needs to be focused
		return m, extendSpaceMsg{send, m.focusOnExtend}.cmd

	case spaceFocusSwitchMsg:
		if focusedTab(msg) == info {
			m.infoTable.Focus()
		} else {
			m.resetFilter()
			m.infoTable.Blur()
			return m, hideInfoSpaceTitle(false).cmd
		}

	case confirmDialogRespMsg:
		if msg && currentFocus == info {
			currentFocus = send
			return m, spaceFocusSwitchMsg(send).cmd
		}

	}

	if m.filterState == filtering {
		m.handleFiltering()
	}

	return m, tea.Batch(m.handleInfoTableUpdate(msg), m.handleFilterInputUpdate(msg))
}

func (m sendInfoModel) View() string {
	help := customInfoTableHelp(m.showHelp)
	help.Width(m.infoTable.Width())

	status := m.getStatus()
	status = infoTableStatusBarStyle.Render(status)

	if m.filter.Focused() {
		filter := m.filter.View()
		c := infoTableFilterContainerStyle.Width(m.filter.Width)
		// Match container width to input field width, but increment by 1 when text exceeds
		// container width to accommodate cursor. This prevents initial centering issues.
		if utf8.RuneCountInString(m.filter.Value()) >= c.GetWidth() {
			c = c.Width(c.GetWidth() + 1)
		}
		return lipgloss.JoinVertical(lipgloss.Center, c.Render(filter), status, m.infoTable.View(), help.Render())
	}
	return lipgloss.JoinVertical(lipgloss.Center, status, m.infoTable.View(), help.Render())
}

func newFilterInputModel() textinput.Model {
	c := cursor.New()
	c.TextStyle = lipgloss.NewStyle().Foreground(highlightColor)
	c.Style = c.TextStyle.Reverse(true)

	f := textinput.New()
	f.PromptStyle = f.PromptStyle.Foreground(highlightColor).Align(lipgloss.Center)
	f.TextStyle = f.TextStyle.Foreground(highlightColor).Align(lipgloss.Center)
	f.Placeholder = "Filter by Name"
	f.PlaceholderStyle = f.PromptStyle.Faint(true)
	f.Cursor = c
	f.Prompt = ""
	return f
}

func (m *sendInfoModel) updateDimensions() {
	w := largeContainerW() - (infoContainerStyle.GetHorizontalFrameSize())
	m.infoTable.SetWidth(w + 2)
	m.filter.Width = (w * 60) / 100 // 60% of available width
	helpHeight := lipgloss.Height(customInfoTableHelp(m.showHelp).String())
	statusBarHeight := infoTableStatusBarStyle.GetHeight() + infoTableStatusBarStyle.GetVerticalFrameSize()
	m.infoTable.SetHeight(infoContainerWorkableH(true) - (helpHeight + statusBarHeight))
	m.infoTable.SetColumns(getTableCols(m.infoTable.Width()))
}

func (m *sendInfoModel) handleInfoTableUpdate(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.infoTable, cmd = m.infoTable.Update(msg)
	return cmd
}

// handle update while also evaluating if the filter is changed
func (m *sendInfoModel) handleFilterInputUpdate(msg tea.Msg) tea.Cmd {
	newModel, cmd := m.filter.Update(msg)
	m.filterChanged = m.filter.Value() != newModel.Value()
	m.filter = newModel
	return cmd
}

func (sendInfoModel) readDir(path string) tea.Cmd {
	return func() tea.Msg {
		entries, err := os.ReadDir(path)
		if err != nil {
			if errors.Is(err, fs.ErrPermission) {
				return fsErrMsg("Perm denied!")
			}
			if errors.Is(err, fs.ErrNotExist) {
				return fsErrMsg("No such dir!")
			}
			return errMsg{
				err:    fmt.Errorf("reading directory %q: %v", path, err),
				errStr: "Unable to read directory contents",
			}.cmd
		}

		dc := dirContents{contents: make([]dirContent, len(entries))}

		for i, entry := range entries {
			eInfo, _ := entry.Info()
			filetype := filepath.Ext(entry.Name())
			// name without ext
			name := strings.TrimSuffix(entry.Name(), filetype)
			filetype = strings.TrimPrefix(filepath.Ext(entry.Name()), ".")
			// for files like .gitignore
			if strings.Count(entry.Name(), ".") == 1 &&
				strings.HasPrefix(entry.Name(), ".") {
				name = entry.Name()
				filetype = ""
			}
			size := util.UserFriendlyFilesize(eInfo.Size())

			if entry.IsDir() {
				name = entry.Name()
				filetype = "dir"
				size = "–––"
				dc.dirs++
			} else {
				dc.files++
			}

			if filetype == "" {
				filetype = "–––"
			}
			dc.contents[i] = dirContent{
				name: name,
				ext:  filetype,
				size: size,
			}
		}
		return dc
	}
}

func (m *sendInfoModel) populateTable(contents []dirContent) {
	// case of filtering && there is some input to filter against
	if m.filterState != unfiltered && utf8.RuneCountInString(m.filter.Value()) > 0 {
		rows := make([]table.Row, 0, len(contents))
		for _, i := range m.dirContents.filteredContents {
			sel := ""
			if contents[i].selection {
				sel = "✓"
			}
			rows = append(rows, table.Row{sel, contents[i].name, contents[i].ext, contents[i].size})
		}
		m.infoTable.SetRows(rows)
		return
	}
	// case of unfiltered
	rows := make([]table.Row, len(contents))
	for i, content := range contents {
		sel := ""
		if content.selection {
			sel = "✓"
		}
		rows[i] = table.Row{sel, content.name, content.ext, content.size}
	}
	m.infoTable.SetRows(rows)
}

func (m sendInfoModel) isValidTableShortcut() bool {
	return currentFocus == info && m.infoTable.Focused() && len(m.infoTable.Rows()) > 0
}

func (m *sendInfoModel) selectAll(selection bool) {
	if m.isValidTableShortcut() {
		if m.filterState != unfiltered {
			for _, i := range m.dirContents.filteredContents {
				m.dirContents.contents[i].selection = selection
			}
		} else {
			for i := range m.dirContents.contents {
				m.dirContents.contents[i].selection = selection
			}
		}
		m.populateTable(m.dirContents.contents)
	}
}

func (m *sendInfoModel) selectSingle(selection bool) {
	sel := m.infoTable.Cursor()
	if m.filterState != unfiltered {
		sel = m.dirContents.filteredContents[sel]
	}
	m.dirContents.contents[sel].selection = selection
	m.populateTable(m.dirContents.contents)
}

func (m *sendInfoModel) resetFilter() {
	m.filter.Reset()
	m.filter.Blur()
	m.filterState = unfiltered
}

func (m sendInfoModel) applyFilter() tea.Cmd {
	m.filter.Blur()
	m.filterState = filterApplied
	m.infoTable.Focus()
	return hideInfoSpaceTitle(false).cmd
}

func (m *sendInfoModel) handleFiltering() tea.Cmd {
	if !m.filterChanged {
		return nil
	}
	m.infoTable.SetCursor(0) // reset cursor, if filter changed
	toFilter := make([]string, len(m.dirContents.contents))
	for i, content := range m.dirContents.contents {
		toFilter[i] = content.name
	}
	indices := filterDirContent(m.filter.Value(), toFilter)
	m.dirContents.filteredContents = indices
	m.populateTable(m.dirContents.contents)
	return nil
}

func (m sendInfoModel) getStatus() string {
	// unfiltered
	status := fmt.Sprintf("%d Dir/s • %d File/s • %d Total",
		m.dirContents.dirs, m.dirContents.files, len(m.dirContents.contents))
	selectCount := m.getSelectionCount()
	if selectCount > 0 {
		status = fmt.Sprintf("%d Dir/s • %d File/s • %d Selected • %d Total",
			m.dirContents.dirs, m.dirContents.files, selectCount, len(m.dirContents.contents))
	}
	if utf8.RuneCountInString(m.filter.Value()) == 0 {
		return status
	}
	matches := "Nothing matched"
	filtered := len(m.dirContents.contents)
	if len(m.dirContents.filteredContents) > 0 {
		matches = fmt.Sprint(len(m.dirContents.filteredContents), " ", "Match/es")
	}
	status = fmt.Sprintf("%s • %d Filtered", matches, filtered)
	if selectCount > 0 {
		status = fmt.Sprintf("%s • %d Selected • %d Filtered", matches, selectCount, filtered)
	}
	if m.filterState == filterApplied {
		status = fmt.Sprintf("“%s” %s", m.filter.Value(), status)
	}
	return runewidth.Truncate(status, largeContainerW()-4, "…") // -4 for tail, infoContainer & statusBar frame size
}

func (m sendInfoModel) getSelectionCount() int {
	count := 0
	for _, content := range m.dirContents.contents {
		if content.selection {
			count++
		}
	}
	return count
}

func customInfoTableHelp(show bool) *lipTable.Table {
	baseStyle := lipgloss.NewStyle().Margin(0, 2)
	var rows [][]string
	if !show {
		rows = [][]string{{"?", "help"}}
	} else {
		rows = [][]string{
			{"shift+↓/ctrl+↓", "make selection"},
			{"shift+↑/ctrl+↑", "undo selection"},
			{"enter", "select/deselect at cursor"},
			{"enter (when filtering)", "apply filter"},
			{"ctrl+a", "select all"},
			{"ctrl+z", "deselect all"},
			{"/", "filter"},
			{"esc", "exit filtering"},
			{"b/pgup", "page up"},
			{"f/space", "page down"},
			{"g/home", "go to start"},
			{"G/end", "go to end"},
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
