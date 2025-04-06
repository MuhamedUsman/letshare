package tui

import (
	"errors"
	"fmt"
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type dirAction = int

const (
	noop dirAction = iota
	in
	out
)

type dirEntryMsg struct {
	entries []string
	action  dirAction
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

type dirListKeyMap struct {
	dirIn  key.Binding
	dirOut key.Binding
	up     key.Binding
	down   key.Binding
}

type dirListModel struct {
	// directory List: children dirs in a parent dirAction
	dirList    list.Model
	curDirPath string
}

func initialDirListModel() dirListModel {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}
	return dirListModel{
		dirList:    newDirList(),
		curDirPath: wd,
	}
}

func (m dirListModel) Init() tea.Cmd {
	return m.readDir(m.curDirPath, noop)
}

func (m dirListModel) Update(msg tea.Msg) (dirListModel, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.updateDimensions()

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			selDir := m.dirList.SelectedItem().FilterValue()
			selDir = filepath.Join(m.curDirPath, selDir) // Dir in
			return m, m.readDir(selDir, in)

		case "backspace":
			dirOut := filepath.Dir(m.curDirPath) // Dir out
			if m.curDirPath == dirOut {
				return m, m.createDirListStatusMsg("It's the top!")
			}
			return m, m.readDir(dirOut, out)

		case "/":
			m.dirList.SetDelegate(customDelegate())

		}

	case dirEntryMsg:
		if msg.action != noop {
			m.curDirPath = m.getCurDirPath(msg.action)
			m.dirList.Title = m.getDirListTitle(m.curDirPath)
		}
		m.dirList.Select(0)
		return m, m.populateDirList(msg.entries)

	case permDeniedMsg:
		return m, m.createDirListStatusMsg("Perm Denied!")

	case errMsg:
		log.Fatal(msg)

	}

	return m, m.handleDirListUpdate(msg)
}

func (m dirListModel) View() string {
	if len(m.dirList.Items()) == 0 {
		m.dirList.SetShowStatusBar(false)
	} else {
		m.dirList.SetShowStatusBar(true)
	}
	return m.dirList.View()
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
	l.Title = "Local"
	l.SetStatusBarItemName("Dir", "Dirs")
	l.DisableQuitKeybindings()

	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("ctrl+left", "ctrl+l"),
				key.WithHelp("ctrl+←/ctrl+l", "out of dirAction"),
			),
			key.NewBinding(
				key.WithKeys("ctrl+right", "ctrl+h"),
				key.WithHelp("ctrl+⇾/ctrl+h", "into dirAction"),
			),
		}
	}

	l.Styles.Title = l.Styles.Title.
		Background(highlightColor).
		Foreground(subduedGrayColor).
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
		PaddingLeft(2).
		Italic(true).
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
	// +1 for some buggy behaviour of pagination when transitioning from filtering to normal list after hitting esc
	h := termH - (mainContainerStyle.GetVerticalFrameSize() + sendContainerStyle.GetVerticalFrameSize() + 1)
	w := smallContainerW() - sendContainerStyle.GetHorizontalFrameSize()
	m.dirList.SetSize(w, h)
}

func (dirListModel) readDir(dir string, action dirAction) tea.Cmd {
	return func() tea.Msg {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if errors.Is(err, fs.ErrPermission) {
				return permDeniedCmd()
			}
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
		return dirEntryMsg{
			entries: dirEntries,
			action:  action,
		}
	}
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

// getDirListTitle creates a simplified display title for the current directory path,
// showing only the volume name and the final directory name. Must be called after
// updating dirListModel.curDirPath.
func (dirListModel) getDirListTitle(curDirPath string) string {
	volName := filepath.VolumeName(curDirPath)
	selDir := filepath.ToSlash(filepath.Base(curDirPath))
	c := strings.Count(curDirPath, string(os.PathSeparator))
	if c == 1 {
		return fmt.Sprintf("%s%s", volName, selDir)
	}
	if selDir == "/" {
		return fmt.Sprintf("%s%s", volName, selDir)
	}
	return fmt.Sprintf("%s/…/%s", volName, selDir)
}

func (m dirListModel) getCurDirPath(da dirAction) string {
	switch da {
	case in:
		selDir := m.dirList.SelectedItem().FilterValue()
		return filepath.Join(m.curDirPath, selDir)
	case out:
		return filepath.Dir(m.curDirPath)
	default:
		return ""
	}
}

func (m *dirListModel) createDirListStatusMsg(s string) tea.Cmd {
	style := lipgloss.NewStyle().
		Foreground(redBrightColor).
		Italic(true)
	return m.dirList.NewStatusMessage(style.Render(s))
}
