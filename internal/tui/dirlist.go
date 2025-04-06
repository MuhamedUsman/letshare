package tui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

type dirItem string

func (i dirItem) FilterValue() string {
	return string(i)
}

func (i dirItem) Title() string {
	return string(i)
}

func (i dirItem) Description() string {
	return ""
}

type dirListModel struct {
	dirStack stack
	// directory List: children dirs in a parent dir
	dirList list.Model
}

func initialDirTreeModel() dirListModel {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}
	s := newStack()
	s.push(wd)
	return dirListModel{
		dirStack: s,
		dirList:  newDirList(),
	}
}

func (m dirListModel) Init() tea.Cmd {
	return m.readDir(m.dirStack.peek())
}

func (m dirListModel) Update(msg tea.Msg) (dirListModel, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.updateDimensions()

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			selDirPath := m.dirList.SelectedItem().FilterValue()      // name of dir
			selDirPath = filepath.Join(m.dirStack.peek(), selDirPath) // peek -> absolute path of parent dir
			m.dirStack.push(selDirPath)
			return m, m.readDir(selDirPath)

		case ">":
			return m, m.readDir(m.dirStack.pop())

		case "/":
			m.dirList.SetDelegate(customDelegate())

		}

	case dirEntryMsg:
		// return m, m.populateDirList(s)

	}

	return m, m.handleDirListUpdate(msg)
}

func (m dirListModel) View() string {
	return m.dirList.View()
}

func (dirListModel) readDir(dir string) tea.Cmd {
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

func newDirList() list.Model {
	s := []list.Item{
		dirItem("Tokyo"),
		dirItem("Delhi"),
		dirItem("Shanghai"),
		dirItem("Dhaka"),
		dirItem("São Paulo"),
		dirItem("Mexico City"),
		dirItem("Cairo"),
		dirItem("Beijing"),
		dirItem("Mumbai"),
		dirItem("Osaka"),
		dirItem("Chongqing"),
		dirItem("Karachi"),
		dirItem("Istanbul"),
		dirItem("Kinshasa"),
		dirItem("Lagos"),
		dirItem("Buenos Aires"),
		dirItem("Kolkata"),
		dirItem("Manila"),
		dirItem("Tianjin"),
		dirItem("Guangzhou"),
		dirItem("Rio De Janeiro"),
		dirItem("Lahore"),
		dirItem("Bangalore"),
		dirItem("Shenzhen"),
		dirItem("Moscow"),
		dirItem("Chennai"),
		dirItem("Bogota"),
		dirItem("Paris"),
		dirItem("Jakarta"),
		dirItem("Lima"),
		dirItem("Bangkok"),
		dirItem("Hyderabad"),
		dirItem("Seoul"),
		dirItem("Nagoya"),
		dirItem("London"),
		dirItem("Chengdu"),
		dirItem("Nanjing"),
		dirItem("Tehran"),
		dirItem("Ho Chi Minh City"),
		dirItem("Luanda"),
		dirItem("Wuhan"),
		dirItem("Xi An Shaanxi"),
		dirItem("Ahmedabad"),
		dirItem("Kuala Lumpur"),
		dirItem("New York City"),
		dirItem("Hangzhou"),
		dirItem("Surat"),
		dirItem("Suzhou"),
		dirItem("Hong Kong"),
		dirItem("Riyadh"),
	}
	l := list.New(s, customDelegate(), 0, 0)
	l.Title = "Directories"
	l.SetStatusBarItemName("Dir", "Dirs")
	l.DisableQuitKeybindings()

	l.Styles.Title = l.Styles.Title.
		Background(highlightColor).
		Foreground(bgColor).
		Italic(true)

	l.Styles.StatusBar = l.Styles.StatusBar.
		Foreground(highlightColor).
		Italic(true).
		Faint(true)

	l.Styles.StatusBarFilterCount = l.Styles.StatusBarFilterCount.
		Foreground(highlightColor).
		Italic(true).
		Faint(true)

	l.Styles.DividerDot = l.Styles.DividerDot.
		Foreground(highlightColor).
		Faint(true)

	l.Styles.StatusEmpty = l.Styles.StatusEmpty.
		Foreground(highlightColor).
		Italic(true).
		Faint(true)

	l.Styles.NoItems = l.Styles.NoItems.
		Foreground(highlightColor).
		Faint(true)

	c := cursor.New()
	c.Style = lipgloss.NewStyle().Foreground(highlightColor)

	f := textinput.New()
	f.PromptStyle = l.Styles.FilterPrompt.Foreground(highlightColor)
	f.TextStyle = lipgloss.NewStyle().Foreground(highlightColor)
	f.Placeholder = "Directory Name"
	f.PlaceholderStyle = lipgloss.NewStyle().
		Foreground(highlightColor).
		Faint(true)
	f.Cursor = c
	f.Prompt = "🔎 "

	l.FilterInput = f

	return l
}

func customDelegate() list.ItemDelegate {
	d := list.NewDefaultDelegate()
	d.ShowDescription = false
	d.SetSpacing(0)
	d.SetHeight(2)

	d.Styles.NormalTitle = d.Styles.NormalTitle.
		Foreground(highlightColor).
		Faint(true)

	d.Styles.SelectedTitle = d.Styles.SelectedTitle.
		Foreground(highlightColor).
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(highlightColor).
		Bold(true).
		Italic(true)

	d.Styles.DimmedTitle = d.Styles.DimmedTitle.
		Foreground(highlightColor).
		Faint(true)

	d.Styles.FilterMatch = d.Styles.FilterMatch.
		Foreground(highlightColor)

	return d
}

func (m *dirListModel) updateDimensions() {
	h := termH - (mainContainerStyle.GetVerticalFrameSize() + sendContainerStyle.GetVerticalFrameSize())
	m.dirList.SetHeight(h - 1)
	m.dirList.SetWidth(smallContainerW())
}

func (m *dirListModel) populateDirList(dirs []string) tea.Cmd {
	items := make([]list.Item, len(dirs))
	for i, dir := range dirs {
		items[i] = dirItem(dir)
	}
	return m.dirList.SetItems(items)
}

func (m *dirListModel) handleDirListUpdate(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.dirList, cmd = m.dirList.Update(msg)
	return cmd
}
