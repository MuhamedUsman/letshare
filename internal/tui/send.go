package tui

import (
	"errors"
	"fmt"
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/mattn/go-runewidth"
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

type sendModel struct {
	// directory List: children dirs in a parent dirAction
	dirList    list.Model
	curDirPath string
	showHelp   bool
}

func initialSendModel() sendModel {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}
	return sendModel{
		curDirPath: wd,
		dirList:    newDirList(),
	}
}

func (m sendModel) Init() tea.Cmd {
	return m.readDir(m.curDirPath, noop)
}

func (m sendModel) Update(msg tea.Msg) (sendModel, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.updateDimensions()

	case tea.KeyMsg:
		if currentFocus == send {
			m.setTitleStylesAsFocus()
			m.dirList.KeyMap = list.DefaultKeyMap() // enable list keymaps
			switch msg.String() {

			case "enter":
				if m.dirList.FilterState() != list.Filtering && m.dirList.SelectedItem() != nil {
					selDir := m.dirList.SelectedItem().FilterValue()
					selDir = filepath.Join(m.curDirPath, selDir) // Dir in
					return m, m.readDir(selDir, in)
				}

			case "backspace":
				if m.dirList.FilterState() == list.Unfiltered {
					dirOut := filepath.Dir(m.curDirPath) // Dir out
					if m.curDirPath == dirOut {
						return m, m.createDirListStatusMsg("Drive Root!", highlightColor)
					}
					return m, m.readDir(dirOut, out)
				}

			case "space":
				selPath := m.dirList.SelectedItem().FilterValue()
				selPath = filepath.Join(m.curDirPath, selPath)
				return m, extentDirMsg(selPath).cmd

			case "?":
				m.showHelp = !m.showHelp
				m.updateDimensions()

			}
		} else {
			m.setTitleStylesAsFocus()
			m.dirList.KeyMap = list.KeyMap{} // disable list keymap
		}

	case dirEntryMsg:
		if msg.action != noop {
			m.curDirPath = m.getCurDirPath(msg.action)
			m.dirList.Title = m.getDirListTitle(m.curDirPath)
		}
		m.dirList.ResetSelected()
		return m, m.populateDirList(msg.entries)

	case fsErrMsg:
		return m, m.createDirListStatusMsg(string(msg), redBrightColor)

	case errMsg:
		log.Fatal(msg)

	}
	return m, m.handleDirListUpdate(msg)
}

func (m sendModel) View() string {
	if len(m.dirList.Items()) == 0 {
		m.dirList.SetShowStatusBar(false)
	} else {
		m.dirList.SetShowStatusBar(true)
	}
	ht := customDirListHelpTable(m.showHelp).Width(m.dirList.Width())
	tail := "..."
	subW := m.dirList.Styles.TitleBar.GetHorizontalFrameSize() // subtract Width
	subW += lipgloss.Width(tail)
	m.dirList.Title = runewidth.Truncate(m.dirList.Title, m.dirList.Width()-subW, "...")
	s := lipgloss.JoinVertical(lipgloss.Top, m.dirList.View(), ht.Render())
	return smallContainerStyle.Render(s)
}

func newDirList() list.Model {
	l := list.New(nil, customDelegate(), 0, 0)
	l.Title = "Local Space"
	l.SetStatusBarItemName("Dir", "Dirs")
	l.DisableQuitKeybindings()
	l.SetShowHelp(false)

	b := []key.Binding{key.NewBinding(key.WithKeys("space")), key.NewBinding(key.WithKeys("backspace"))}
	l.AdditionalFullHelpKeys = func() []key.Binding { return b }
	l.AdditionalShortHelpKeys = func() []key.Binding { return b }

	l.Help.Styles = customDirListHelpStyles(l.Help.Styles)

	l.Styles.HelpStyle = l.Styles.HelpStyle.Padding(1, 0, 0, 1)

	l.Styles.Title = l.Styles.Title.
		Background(highlightColor).
		Foreground(subduedHighlightColor).
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

func customDirListHelpStyles(s help.Styles) help.Styles {
	s.FullSeparator = s.FullSeparator.Foreground(highlightColor).Faint(true)
	s.ShortSeparator = s.ShortSeparator.Foreground(highlightColor).Faint(true)
	s.ShortKey = s.ShortKey.Foreground(highlightColor).Faint(true)
	s.FullKey = s.FullKey.Foreground(highlightColor).Faint(true)
	s.FullDesc = s.FullDesc.Foreground(subduedHighlightColor)
	s.ShortDesc = s.ShortDesc.Foreground(subduedHighlightColor)
	return s
}

func customDelegate() list.ItemDelegate {
	d := list.NewDefaultDelegate()
	d.ShowDescription = false
	d.SetSpacing(0)
	d.SetHeight(2)

	d.Styles.NormalTitle = d.Styles.NormalTitle.
		Foreground(midHighlightColor)

	d.Styles.SelectedTitle = d.Styles.SelectedTitle.
		Foreground(highlightColor).
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(highlightColor).
		Bold(true).
		Italic(true)

	d.Styles.DimmedTitle = d.Styles.DimmedTitle.
		Foreground(midHighlightColor)

	d.Styles.FilterMatch = d.Styles.FilterMatch.
		Foreground(highlightColor)

	return d
}

func customDirListHelpTable(show bool) *table.Table {
	baseStyle := lipgloss.NewStyle().Margin(0, 1)
	var rows [][]string
	if !show {
		rows = [][]string{{"?", "help"}}
	} else {
		rows = [][]string{
			{"/", "filter"},
			{"space", "extend directory"},
			{"enter", "into directory"},
			{"backspace", "out of directory"},
			{"←||→", "shuffle pages"},
			{"esc", "exit filtering"},
			{"?", "hide help"},
		}
	}
	return table.New().
		Border(lipgloss.HiddenBorder()).
		BorderBottom(false).
		Wrap(false).
		StyleFunc(func(row, col int) lipgloss.Style {
			switch col {
			case 0:
				return baseStyle.Foreground(highlightColor).Faint(true) // key style
			case 1:
				return baseStyle.Foreground(subduedHighlightColor) // desc style
			default:
				return lipgloss.Style{}
			}
		}).Rows(rows...)

}

func (m *sendModel) updateDimensions() {
	// sub '1' height for some buggy behaviour of pagination when transitioning from filtering to normal list state,
	// if the pagination will be visible afterward, it adds '1' height to the list till the next update is called
	helpHeight := lipgloss.Height(customDirListHelpTable(m.showHelp).String())
	h := termH - (mainContainerStyle.GetVerticalFrameSize() + smallContainerStyle.GetVerticalFrameSize() + helpHeight + 1)
	w := smallContainerW() - smallContainerStyle.GetHorizontalFrameSize()
	m.dirList.SetSize(w, h)
	// the width of titleBar gets +1 when the title txt overflows so explicitly constraining it
	w = w - m.dirList.Styles.TitleBar.GetHorizontalFrameSize()
	m.dirList.Styles.TitleBar = m.dirList.Styles.TitleBar.MaxWidth(w)
}

func (sendModel) readDir(dir string, action dirAction) tea.Cmd {
	return func() tea.Msg {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if errors.Is(err, fs.ErrPermission) {
				return fsErrMsg("Perm Denied!")
			}
			if errors.Is(err, fs.ErrNotExist) {
				return fsErrMsg("No such dir!")
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

func (m *sendModel) populateDirList(dirs []string) tea.Cmd {
	items := make([]list.Item, len(dirs))
	for i, dir := range dirs {
		items[i] = dirItem(dir)
	}
	return m.dirList.SetItems(items)
}

func (m *sendModel) handleDirListUpdate(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.dirList, cmd = m.dirList.Update(msg)
	return cmd
}

// getDirListTitle creates a simplified display title for the current directory path,
// showing only the volume name and the final directory name. Must be called after
// updating sendModel.curDirPath.
func (sendModel) getDirListTitle(curDirPath string) string {
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

func (m sendModel) getCurDirPath(da dirAction) string {
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

func (m *sendModel) createDirListStatusMsg(s string, c lipgloss.AdaptiveColor) tea.Cmd {
	style := lipgloss.NewStyle().
		Foreground(c).
		Italic(true)
	return m.dirList.NewStatusMessage(style.Render(s))
}

func (m *sendModel) setTitleStylesAsFocus() {
	s := m.dirList.Styles.Title.
		Background(subduedGrayColor).
		Foreground(highlightColor)
	if currentFocus == send {
		s = m.dirList.Styles.Title.
			Background(highlightColor).
			Foreground(subduedHighlightColor)
	}
	m.dirList.Styles.Title = s
}
