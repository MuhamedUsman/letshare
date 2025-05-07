package tui

import (
	"errors"
	"fmt"
	"github.com/MuhamedUsman/letshare/internal/tui/table"
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	lipTable "github.com/charmbracelet/lipgloss/table" // lipTable -> lipglossTable
	"github.com/dustin/go-humanize"
	"github.com/mattn/go-runewidth"
	"github.com/sahilm/fuzzy"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	nae = "---" // Not an Extension
	dir = "dir"
)

type filterState int

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
	parentDir        string
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

type extDirNavModel struct {
	extDirTable                                           table.Model
	filter                                                textinput.Model
	titleStyle                                            lipgloss.Style
	filterState                                           filterState
	dirContents                                           dirContents
	dirPath                                               string
	filterChanged, focusOnExtend, showHelp, disableKeymap bool
}

func initialExtDirNavModel() extDirNavModel {
	t := table.New(
		table.WithStyles(customTableStyles),
		table.WithColumns(getTableCols(0)),
	)
	return extDirNavModel{extDirTable: t, filter: newFilterInputModel(), titleStyle: titleStyle}
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

func (m extDirNavModel) capturesKeyEvent(msg tea.KeyMsg) bool {
	// textInput model captures keys, so we need to handle it here
	if m.filterState == filtering {
		return true
	}
	switch msg.String() {
	case "enter", "ctrl+s":
		return m.filterState == filtering || (m.isValidTableShortcut() && m.filterState != filtering)
	case "up", "down", "?", "ctrl+a", "ctrl+z":
		return true
	case "/", "shift+up", "ctrl+up", "shift+down", "ctrl+down":
		return m.isValidTableShortcut()
	case "esc":
		return m.filterState != unfiltered || m.getSelectionCount() > 0
	default:
		return false
	}
}

func (m extDirNavModel) Init() tea.Cmd {
	return nil
}

func (m extDirNavModel) Update(msg tea.Msg) (extDirNavModel, tea.Cmd) {

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
				m.extDirTable.Focus()
			} else if m.isValidTableShortcut() && m.filterState != filtering {
				m.extDirTable.Focus()
				sel := m.extDirTable.Cursor()
				if m.filterState != unfiltered {
					// if filtering OR filterApplied then sel changes based on filtered indices
					sel = m.dirContents.filteredContents[sel]
				}
				m.dirContents.contents[sel].selection = !m.dirContents.contents[sel].selection
				m.populateTable(m.dirContents.contents)
			}

		case "up", "down":
			if m.filterState == filtering {
				m.applyFilter()
				return m, tea.Batch(m.handleInfoTableUpdate(msg))
			}

		case "shift+down", "ctrl+down": // select a row and move down
			if m.isValidTableShortcut() {
				m.selectSingle(true)
				m.extDirTable.MoveDown(1)
			}

		case "shift+up", "ctrl+up": // undo selection & move up
			if m.isValidTableShortcut() {
				m.selectSingle(false)
				m.extDirTable.MoveUp(1)
			}

		case "ctrl+a":
			m.selectAll(true)

		case "ctrl+z":
			m.selectAll(false)

		case "ctrl+s":
			if m.filterState != filtering {
				return m, m.confirmSend()
			}

		case "/":
			if m.isValidTableShortcut() {
				m.filterState = filtering
				m.extDirTable.Blur()
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
				m.extDirTable.Focus()
				m.populateTable(m.dirContents.contents)
			} else if m.getSelectionCount() > 0 {
				return m, m.confirmDiacardSel(home)
			}
		}

	case extendDirMsg:
		// the user is trying to extend a new dir, but the previous extended dir has selected items
		if m.getSelectionCount() > 0 {
			return m, m.confirmDiscardSelOnNewExtDir(msg)
		}
		m.focusOnExtend = msg.focus
		return m, m.readDir(msg.path)

	case dirContents:
		m.dirPath = msg.parentDir
		m.dirContents = msg
		m.populateTable(msg.contents)
		if m.focusOnExtend {
			m.extDirTable.Focus()
		} else {
			m.extDirTable.Blur()
		}
		m.extDirTable.SetCursor(0)
		// if table is focused, then extensionSpace child also needs to be focused
		return m, extensionChildSwitchMsg{extDirNav, m.focusOnExtend}.cmd

	case spaceFocusSwitchMsg:
		if currentFocus == extension {
			m.updateTitleStyleAsFocus(true)
			m.extDirTable.Focus()
		} else {
			m.updateTitleStyleAsFocus(false)
			m.resetFilter()
			m.extDirTable.Blur()
		}

	}

	if m.filterState == filtering {
		m.handleFiltering()
	}

	return m, tea.Batch(m.handleInfoTableUpdate(msg), m.handleFilterInputUpdate(msg))
}

func (m extDirNavModel) View() string {
	help := customExtDirTableHelp(m.showHelp)
	help.Width(m.extDirTable.Width() - 2)
	title := "Extended Dir: " + filepath.Base(m.dirPath)
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
		return lipgloss.JoinVertical(lipgloss.Center, c.Render(filter), status, m.extDirTable.View(), help.Render())
	}
	return lipgloss.JoinVertical(lipgloss.Center, title, status, m.extDirTable.View(), help.Render())
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

func (m *extDirNavModel) updateDimensions() {
	w := largeContainerW() - largeContainerStyle.GetHorizontalFrameSize()
	m.extDirTable.SetWidth(w + 2)
	m.filter.Width = (w * 60) / 100 // 60% of available width
	helpHeight := lipgloss.Height(customExtDirTableHelp(m.showHelp).String())
	statusBarHeight := extStatusBarStyle.GetHeight() + extStatusBarStyle.GetVerticalFrameSize()
	titleHeight := m.titleStyle.GetHeight() + m.titleStyle.GetVerticalFrameSize()
	m.extDirTable.SetHeight(extContainerWorkableH() - (titleHeight + statusBarHeight + helpHeight))
	m.extDirTable.SetColumns(getTableCols(m.extDirTable.Width()))
}

func (m *extDirNavModel) updateTitleStyleAsFocus(focus bool) {
	if focus {
		m.titleStyle = titleStyle.
			UnsetMarginBottom().
			Background(highlightColor).
			Foreground(subduedHighlightColor)
	} else {
		m.titleStyle = titleStyle.
			UnsetMarginBottom().
			Background(subduedGrayColor).
			Foreground(highlightColor)
	}
}

func (m *extDirNavModel) handleInfoTableUpdate(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.extDirTable, cmd = m.extDirTable.Update(msg)
	return cmd
}

// handle update while also evaluating if the filter is changed
func (m *extDirNavModel) handleFilterInputUpdate(msg tea.Msg) tea.Cmd {
	newModel, cmd := m.filter.Update(msg)
	m.filterChanged = m.filter.Value() != newModel.Value()
	m.filter = newModel
	return cmd
}

func (extDirNavModel) readDir(path string) tea.Cmd {
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
				errStr: "Unable to processed directory contents",
			}.cmd
		}

		dc := dirContents{parentDir: path, contents: make([]dirContent, len(entries))}

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
			size := humanize.Bytes(uint64(eInfo.Size()))

			if entry.IsDir() {
				name = entry.Name()
				filetype = dir
				size = "–––"
				dc.dirs++
			} else {
				dc.files++
			}

			if filetype == "" {
				filetype = nae
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

func (m *extDirNavModel) populateTable(contents []dirContent) {
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
		m.extDirTable.SetRows(rows)
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
	m.extDirTable.SetRows(rows)
}

func (m extDirNavModel) isValidTableShortcut() bool {
	return currentFocus == extension && m.extDirTable.Focused() && len(m.extDirTable.Rows()) > 0
}

func (m *extDirNavModel) selectAll(selection bool) {
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

func (m *extDirNavModel) selectSingle(selection bool) {
	sel := m.extDirTable.Cursor()
	if m.filterState != unfiltered {
		sel = m.dirContents.filteredContents[sel]
	}
	m.dirContents.contents[sel].selection = selection
	m.populateTable(m.dirContents.contents)
}

func (m *extDirNavModel) resetFilter() {
	m.filter.Reset()
	m.filter.Blur()
	m.filterState = unfiltered
}

func (m *extDirNavModel) applyFilter() {
	m.filter.Blur()
	m.filterState = filterApplied
	m.extDirTable.Focus()
}

func (m *extDirNavModel) handleFiltering() tea.Cmd {
	if !m.filterChanged {
		return nil
	}
	m.extDirTable.SetCursor(0) // reset cursor, if filter changed
	toFilter := make([]string, len(m.dirContents.contents))
	for i, content := range m.dirContents.contents {
		toFilter[i] = content.name
	}
	indices := filterDirContent(m.filter.Value(), toFilter)
	m.dirContents.filteredContents = indices
	m.populateTable(m.dirContents.contents)
	return nil
}

func (m *extDirNavModel) resetSelections() {
	for i := range m.dirContents.contents {
		m.dirContents.contents[i].selection = false
	}
}

func (m extDirNavModel) getStatus() string {
	// unfiltered
	status := fmt.Sprintf("%d Dir/s • %d File/s • %d Total",
		m.dirContents.dirs, m.dirContents.files, len(m.dirContents.contents))
	selectCount := m.getSelectionCount()
	if selectCount > 0 {
		status = fmt.Sprintf("%d Dir/s • %d File/s • %d Selected • %d Total",
			m.dirContents.dirs, m.dirContents.files, selectCount, len(m.dirContents.contents))
	}
	if utf8.RuneCountInString(m.filter.Value()) == 0 {
		return runewidth.Truncate(status, largeContainerW()-4, "…") // -4 for tail, extensionContainer & statusBar frame size
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
	return runewidth.Truncate(status, largeContainerW()-4, "…") // -4 for tail, extensionContainer & statusBar frame size
}

func (m extDirNavModel) getSelectionCount() int {
	count := 0
	for _, content := range m.dirContents.contents {
		if content.selection {
			count++
		}
	}
	return count
}

func (m extDirNavModel) selectedFilenames() (filenames []string, dirs, files int) {
	filenames = make([]string, 0, len(m.dirContents.contents))
	for _, c := range m.dirContents.contents {
		if !c.selection {
			continue
		}
		filename := c.name
		if c.ext == dir {
			dirs++
		} else {
			if c.ext != nae {
				filename += "." + c.ext
			}
			files++
		}
		filenames = append(filenames, filename)
	}
	return
}

func (m *extDirNavModel) confirmDiscardSelOnNewExtDir(msg extendDirMsg) tea.Cmd {
	selBtn := yup
	header := "ARE YOU SURE?"
	body := "All the selections will be lost..."
	yupFunc := func() tea.Cmd {
		m.focusOnExtend = msg.focus
		m.resetSelections()
		return m.readDir(msg.path)
	}
	return confirmDialogCmd(header, body, selBtn, yupFunc, nil, nil)
}

func (m *extDirNavModel) confirmDiacardSel(space extChild) tea.Cmd {
	// when filtering, we will not grant an extension switch
	if m.filterState != unfiltered {
		return nil
	}
	cmd := extensionChildSwitchMsg{space, true}.cmd
	if m.getSelectionCount() == 0 {
		return cmd
	}
	selBtn := yup
	header := "ARE YOU SURE?"
	body := "All the selections will be lost..."
	yupFunc := func() tea.Cmd {
		m.resetSelections()
		return cmd
	}
	return confirmDialogCmd(header, body, selBtn, yupFunc, nil, nil)
}

func (m *extDirNavModel) confirmSend() tea.Cmd {
	filenames, dirs, files := m.selectedFilenames()
	var dirStr, fileStr, space string
	if dirs > 0 {
		dirStr = fmt.Sprintf("%d Dir/s", dirs)
	}
	if files > 0 {
		fileStr = fmt.Sprintf("%d File/s", files)
	}
	if dirs > 0 && files > 0 {
		space = " and "
	}
	selBtn := yup
	header := "PROCEED?"
	body := fmt.Sprintf(`Selected “%s%s%s” will be processed as per preferences. To change preferences, press “esc” & “ctrl+p”.`,
		dirStr, space, fileStr)
	yupFunc := func() tea.Cmd {
		m.resetSelections()
		return processSelectionsMsg{
			parentPath: m.dirPath,
			filenames:  filenames,
			dirs:       dirs,
			files:      files,
		}.cmd
	}
	return confirmDialogCmd(header, body, selBtn, yupFunc, nil, nil)
}

func (m extDirNavModel) grantSpaceFocusSwitch() bool {
	return m.filterState == unfiltered
}

func (m *extDirNavModel) updateKeymap(disable bool) {
	m.disableKeymap = disable
}

func customExtDirTableHelp(show bool) *lipTable.Table {
	baseStyle := lipgloss.NewStyle()
	var rows [][]string
	if !show {
		rows = [][]string{{"?", "help"}}
	} else {
		rows = [][]string{
			{"shift+↓/ctrl+↓", "make selection"},
			{"shift+↑/ctrl+↑", "undo selection"},
			{"enter", "select/deselect at cursor"},
			{"enter (when filtering)", "apply filter"},
			{"ctrl+a/z", "select/deselect all"},
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
