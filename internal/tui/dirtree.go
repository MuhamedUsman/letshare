package tui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"log"
	"os"
	"path/filepath"
)

type dirEntryMsg = []string

type stack struct {
	s    []string
	push func(string)
	pop  func() string
	peek func() string
}

func newStack() stack {
	s := make([]string, 0)
	return stack{
		push: func(dir string) { s = append(s, dir) },
		pop: func() string {
			if len(s) == 0 {
				return ""
			}
			r := s[len(s)-1]
			s = s[:len(s)-1]
			return r
		},
		peek: func() string {
			if len(s) == 0 {
				return ""
			}
			return s[len(s)-1]
		},
	}
}

type dirTreeModel struct {
	dirStack stack
	// current dir entries: children entries at specific parent
	curDE    []string
	dirTable table.Model
}

func initialDirTreeModel() dirTreeModel {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}
	s := newStack()
	s.push(wd)
	return dirTreeModel{
		dirStack: s,
		dirTable: newDirTable(),
	}
}

func (m dirTreeModel) Init() tea.Cmd {
	return m.readDir(m.dirStack.peek())
}

func (m dirTreeModel) Update(msg tea.Msg) (dirTreeModel, tea.Cmd) {
	log.Println(m.dirTable.Rows())
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.updateDirTableDim()

	case tea.KeyMsg:
		switch msg.String() {
		case "right":
			selDirPath := m.dirTable.SelectedRow()[0]
			selDirPath = filepath.Join(m.dirStack.peek(), selDirPath) // peek -> absolute path of parent dir
			m.dirStack.push(selDirPath)
			return m, m.readDir(selDirPath)

		case "left":
			return m, m.readDir(m.dirStack.pop())

		case "esc":
			m.dirTable.SetRows(nil)
			return m, tea.Quit
		}

	case dirEntryMsg:
		m.populateDirTable(msg)

	}
	return m, m.handleTableModelUpdate(msg)
}

func (m dirTreeModel) View() string {
	return m.dirTable.View()
}

func newDirTable() table.Model {
	cols := []table.Column{
		{Title: "Directories", Width: 30},
	}
	t := table.New(table.WithColumns(cols), table.WithFocused(true))
	s := table.DefaultStyles()
	s.Header = s.Header.
		PaddingBottom(1).
		Foreground(fgColor).
		Bold(false)
	s.Cell = s.Cell.PaddingLeft(2)
	s.Selected = s.Selected.
		//BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(yellowColor).
		//BorderLeft(true).
		UnsetPaddingLeft().
		Foreground(yellowColor).
		Bold(true).
		Italic(true)
	t.SetStyles(s)
	return t
}

func (m *dirTreeModel) handleTableModelUpdate(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.dirTable, cmd = m.dirTable.Update(msg)
	return cmd
}

func (dirTreeModel) readDir(dir string) tea.Cmd {
	return func() tea.Msg {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return errMsg{
				err:    fmt.Errorf("reading directory %q: %v", dir, err),
				errStr: "Unable to read directory contents",
			}.cmd
		}
		dirEntries := make([]string, 0)
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			dirEntries = append(dirEntries, e.Name())
		}
		return dirEntries
	}
}

func (m *dirTreeModel) updateDirTableDim() {
	m.dirTable.SetWidth(max(0, termW-mainContainerStyle.GetHorizontalFrameSize()))
	m.dirTable.SetHeight(max(0, termH-mainContainerStyle.GetVerticalFrameSize()))
}

func (m *dirTreeModel) populateDirTable(de []string) {
	rows := make([]table.Row, 0)
	for _, e := range de {
		r := table.Row{e}
		rows = append(rows, r)
	}
	m.dirTable.SetRows(rows)
}

func hasNestedDir(path string) bool {
	entries, _ := os.ReadDir(path)
	for _, entry := range entries {
		if entry.IsDir() {
			return true
		}
	}
	return false
}
