package tui

import (
	"errors"
	"fmt"
	"github.com/MuhamedUsman/letshare/internal/tui/table"
	"github.com/MuhamedUsman/letshare/internal/util"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	lipTable "github.com/charmbracelet/lipgloss/table" // lipTable -> lipglossTable
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"
)

type dirContent struct {
	selection       bool
	name, ext, size string
}

type dirContentsMsg = []dirContent

type sendInfoModel struct {
	infoTable  table.Model
	tableFocus bool
	dirPath    string
	dirContent []dirContent
	// Toggle for help display
	showHelp bool
}

func initialSendInfoModel() sendInfoModel {
	t := table.New(
		table.WithStyles(customTableStyles),
		table.WithColumns(getTableCols(0)),
	)
	return sendInfoModel{infoTable: t}
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
			if m.infoTable.Focused() {
				sel := m.infoTable.Cursor()
				m.dirContent[sel].selection = !m.dirContent[sel].selection
				m.populateTable(m.dirContent)
			}

		case "shift+down", "ctrl+down": // select row & move down
			if m.infoTable.Focused() {
				sel := m.infoTable.Cursor()
				m.dirContent[sel].selection = true
				m.populateTable(m.dirContent)
				m.infoTable.MoveDown(1)
			}

		case "shift+up", "ctrl+up": // undo selection & move up
			if m.infoTable.Focused() {
				sel := m.infoTable.Cursor()
				m.dirContent[sel].selection = false
				m.populateTable(m.dirContent)
				m.infoTable.MoveUp(1)
			}

		case "ctrl+a":
			if m.infoTable.Focused() {
				for i := range m.dirContent {
					m.dirContent[i].selection = true
				}
				m.populateTable(m.dirContent)
			}

		case "ctrl+z":
			if m.infoTable.Focused() {
				for i := range m.dirContent {
					m.dirContent[i].selection = false
				}
				m.populateTable(m.dirContent)
			}

		case "?":
			if m.infoTable.Focused() {
				m.showHelp = !m.showHelp
				m.updateDimensions()
			}

		}

	case extendDirMsg:
		m.tableFocus = msg.focus
		return m, m.readDir(msg.path)

	case dirContentsMsg:
		m.dirContent = msg
		m.populateTable(msg)
		if m.tableFocus {
			m.infoTable.Focus()
		} else {
			m.infoTable.Blur()
		}
		// if table is focused, then info space also needs to be focused
		return m, extendSpaceMsg{send, m.tableFocus}.cmd

	case spaceFocusSwitchMsg:
		if focusedTab(msg) == info {
			m.infoTable.Focus()
		} else {
			m.infoTable.Blur()
		}

	}
	return m, m.handleInfoTableUpdate(msg)
}

func (m sendInfoModel) View() string {
	help := customInfoTableHelp(m.showHelp)
	help.Width(m.infoTable.Width())
	return lipgloss.JoinVertical(lipgloss.Top, m.infoTable.View(), help.Render())
}

func (m *sendInfoModel) updateDimensions() {
	w := largeContainerW() - (infoContainerStyle.GetHorizontalFrameSize())
	m.infoTable.SetWidth(w + 2)
	helpHeight := lipgloss.Height(customInfoTableHelp(m.showHelp).String())
	m.infoTable.SetHeight(infoContainerWorkableH() - helpHeight)
	m.infoTable.SetColumns(getTableCols(m.infoTable.Width()))
}

func (m *sendInfoModel) handleInfoTableUpdate(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.infoTable, cmd = m.infoTable.Update(msg)
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
		dirContents := make([]dirContent, 0, len(entries))
		for _, entry := range entries {
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
			}
			if filetype == "" {
				filetype = "–––"
			}
			c := dirContent{
				name: name,
				ext:  filetype,
				size: size,
			}
			dirContents = append(dirContents, c)
		}
		return dirContents
	}
}

func (m *sendInfoModel) populateTable(msg dirContentsMsg) {
	rows := make([]table.Row, len(msg))
	for i, content := range msg {
		sel := ""
		if content.selection {
			sel = "✓"
		}
		rows[i] = table.Row{sel, content.name, content.ext, content.size}
	}
	m.infoTable.SetRows(rows)
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
