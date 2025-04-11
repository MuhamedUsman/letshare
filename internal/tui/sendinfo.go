package tui

import (
	"errors"
	"fmt"
	"github.com/MuhamedUsman/letshare/internal/tui/table"
	"github.com/MuhamedUsman/letshare/internal/util"
	tea "github.com/charmbracelet/bubbletea"
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
}

func initialSendInfoModel() sendInfoModel {
	t := table.New(
		table.WithStyles(customTableStyles),
		table.WithColumns(getTableCols(0)),
	)
	return sendInfoModel{infoTable: t}
}

func getTableCols(tableWidth int) []table.Column {
	cols := []string{"✓", "Name", "Ext", "Size"}
	subW := customTableStyles.Cell.GetHorizontalFrameSize() * len(cols)
	tableWidth -= subW
	selectionW := 1
	tableWidth -= selectionW
	nameW := (tableWidth * 62) / 100
	extW := (tableWidth * 18) / 100
	sizeW := (tableWidth * 20) / 100
	// terminals have rows x cols as integer values
	// this condition is to take into account the
	// precision loss from above division ops
	nameW += int(math.Abs(float64(tableWidth - (sizeW + extW + nameW))))
	return []table.Column{
		{cols[0], selectionW},
		{cols[1], nameW},
		{cols[2], extW},
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

		case "shift+down": // select row & move down
			if m.infoTable.Focused() {
				sel := m.infoTable.Cursor()
				m.dirContent[sel].selection = true
				m.populateTable(m.dirContent)
				m.infoTable.MoveDown(1)
			}

		case "shift+up": // undo selection & move up
			if m.infoTable.Focused() {
				sel := m.infoTable.Cursor()
				m.dirContent[sel].selection = false
				m.populateTable(m.dirContent)
				m.infoTable.MoveUp(1)
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

	case spaceTabSwitchMsg:
		if focusedTab(msg) == info {
			m.infoTable.Focus()
		} else {
			m.infoTable.Blur()
		}

	}
	return m, m.handleInfoTableUpdate(msg)
}

func (m sendInfoModel) View() string {
	return m.infoTable.View()
}

func (m *sendInfoModel) updateDimensions() {
	w := largeContainerW() - (infoContainerStyle.GetHorizontalFrameSize())
	m.infoTable.SetWidth(w)
	m.infoTable.SetHeight(infoContainerWorkableH())
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
			// only files
			if entry.IsDir() {
				continue
			}
			eInfo, _ := entry.Info()
			ext := filepath.Ext(entry.Name())
			name := strings.TrimSuffix(entry.Name(), ext)
			ext = strings.TrimPrefix(filepath.Ext(entry.Name()), ".")
			if ext == "" {
				ext = "–––"
			}
			c := dirContent{
				name: name,
				ext:  ext,
				size: util.UserFriendlyFilesize(eInfo.Size()),
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
