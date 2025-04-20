package tui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	lipTable "github.com/charmbracelet/lipgloss/table"
	"github.com/mattn/go-runewidth"
	"os"
	"strconv"
)

type preferenceType int

const (
	option preferenceType = iota
	input
)

type preferenceSection int

const (
	sharing preferenceSection = iota
	receiving
)

var prefSecNames = []string{
	"Sharing",
	"Receiving",
}

func (ps preferenceSection) string() string {
	if int(ps) < 0 || int(ps) >= len(prefSecNames) {
		return "unknown preference section " + strconv.Itoa(int(ps))
	}
	return prefSecNames[ps]
}

type preferenceQue struct {
	title, desc string
	pType       preferenceType
	pSec        preferenceSection
}

type preferenceModel struct {
	preferenceQues          []preferenceQue
	vp                      viewport.Model
	titleStyle              lipgloss.Style
	showHelp, disableKeymap bool
}

func initialPreferenceModel() preferenceModel {
	// Load some text for our viewport
	content, err := os.ReadFile("internal/tui/artichoke.md")
	if err != nil {
		fmt.Println("could not load file:", err)
		os.Exit(1)
	}
	preferenceQues := []preferenceQue{
		{
			title: "ZIP FILES?",
			desc:  "Share selected files as a single zip.",
			pType: option,
			pSec:  sharing,
		},
		{
			title: "ISOLATE FILES?",
			desc:  "Copy selected files to a separate directory before sharing.",
			pType: option,
			pSec:  sharing,
		},
	}
	vp := viewport.New(0, 0)
	vp.SetContent(string(content))
	vp.Style = vp.Style.Padding(1, 2, 0, 2)
	return preferenceModel{
		preferenceQues: preferenceQues,
		vp:             vp,
	}
}

func (m preferenceModel) Init() tea.Cmd {
	return m.vp.Init()
}

func (m preferenceModel) Update(msg tea.Msg) (preferenceModel, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		if m.disableKeymap {
			return m, nil
		}
		switch msg.String() {

		case "?":
			m.showHelp = !m.showHelp
			m.updateViewPortDimensions()

		}
	case tea.WindowSizeMsg:
		m.updateViewPortDimensions()

	case spaceFocusSwitchMsg:
		m.updateTitleStyleAsFocus(currentFocus == extension)

	}
	return m, m.handleViewportUpdate(msg)
}

func (m preferenceModel) View() string {
	title := m.renderTitle("Preferences")
	status := m.renderStatusBar()
	help := customPreferenceHelp(m.showHelp)
	help.Width(largeContainerW())
	c := lipgloss.JoinVertical(lipgloss.Center, title, status, m.vp.View(), help.Render())
	return c
}

func (m *preferenceModel) handleViewportUpdate(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return cmd
}

func (m *preferenceModel) updateViewPortDimensions() {
	titleH := m.titleStyle.GetHeight() + m.titleStyle.GetVerticalFrameSize()
	statusBarH := extStatusBarStyle.GetHeight() + extStatusBarStyle.GetVerticalFrameSize()
	helpH := lipgloss.Height(customPreferenceHelp(m.showHelp).String())
	viewportFrameH := m.vp.Style.GetVerticalFrameSize()
	h := extContainerWorkableH() - (titleH + statusBarH + helpH + viewportFrameH)
	w := largeContainerW() - largeContainerStyle.GetHorizontalFrameSize()
	m.vp.Width, m.vp.Height = w, h
	if m.vp.PastBottom() {
		m.vp.GotoBottom()
	}
}

func (m *preferenceModel) updateTitleStyleAsFocus(focus bool) {
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

func (m preferenceModel) renderTitle(title string) string {
	tail := "…"
	w := largeContainerW() - (lipgloss.Width(tail) + titleStyle.GetHorizontalPadding() + lipgloss.Width(tail))
	title = runewidth.Truncate(title, w, tail)
	return m.titleStyle.Render(title)
}

func (m preferenceModel) renderStatusBar() string {
	scrolled := m.vp.ScrollPercent() * 100
	s := fmt.Sprintf("Scrolled: %06.2f%%", scrolled)
	return extStatusBarStyle.Render(s)
}

func (m *preferenceModel) updateKeymap(disable bool) {
	m.disableKeymap = disable
}

func customPreferenceHelp(show bool) *lipTable.Table {
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
			{"f/child", "page down"},
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
