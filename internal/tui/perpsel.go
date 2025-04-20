package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"strings"
)

// prepQue(Prep Question) is a question to be asked the user
type prepQue struct {
	// title and body for the question
	title, body string
	// the choice the user made
	check bool // true -> Checked | false -> Unchecked
}

// perpSelModel(Prepare Selections Model) prepares selected files for sharing
// the intent is to ask the user if they want to zip files as a single zip file
// and to isolate files to a separate directory before sharing them.
type prepSelModel struct {
	// we hold on to processMsg to be able to send it for processing
	// once the user has made their choices
	processMsg processSelectionsMsg
	prepQues   []prepQue
	cursor     int
	titleStyle lipgloss.Style
}

func initialPrepSelModel() prepSelModel {
	q := []prepQue{
		{
			title: "ZIP Files?",
			body:  "Share selected files as a single zip.",
		},
		{
			title: "Isolate Files?",
			body:  "Copy selected files to a separate directory before sharing.",
		},
	}
	return prepSelModel{prepQues: q}
}

func (m prepSelModel) Init() tea.Cmd {
	return nil
}

func (m prepSelModel) Update(msg tea.Msg) (prepSelModel, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
		}

	case spaceFocusSwitchMsg:
		m.updateTitleStyleAsFocus(true)

	case processSelectionsMsg:
		m.processMsg = msg
		currentFocus = local
		return m, spaceFocusSwitchCmd
	}
	return m, nil
}

func (m prepSelModel) View() string {
	title := m.titleStyle.Margin(0, 2).Render("Sharing Preferences")
	status := m.getStatus()
	status = prepSelStatusBarStyle.Render(status)
	sb := new(strings.Builder)
	for i, q := range m.prepQues {
		if i == m.cursor {
			sb.WriteString(m.renderActiveQue(q))
		} else {
			sb.WriteString(m.renderInactiveQue(q))
		}
		sb.WriteString("\n")
	}
	return smallContainerStyle.Width(smallContainerW()).Render(lipgloss.JoinVertical(lipgloss.Left, title, status, sb.String()))
}

func (prepSelModel) renderActiveQue(q prepQue) string {
	titleS := prepQuesTitleStyle.
		Background(highlightColor).
		Foreground(subduedHighlightColor).
		Faint(true)
	bodyS := prepQuesBodyStyle.
		Foreground(highlightColor)
	if !q.check {
		titleS = titleS.Strikethrough(true)
		bodyS = bodyS.Strikethrough(true)
	}
	title := truncateRenderedTitle(q.title)
	ques := lipgloss.JoinVertical(lipgloss.Left, titleS.Render(title), bodyS.Render(q.body))
	return prepQuesContainerStyle.
		Width(smallContainerW() - smallContainerStyle.GetHorizontalFrameSize()).
		Render(ques)
}

func (prepSelModel) renderInactiveQue(q prepQue) string {
	titleS := prepQuesTitleStyle
	bodyS := prepQuesBodyStyle
	if !q.check {
		titleS = titleS.Strikethrough(true)
		bodyS = bodyS.Strikethrough(true)
	}
	title := truncateRenderedTitle(q.title)
	ques := lipgloss.JoinVertical(lipgloss.Left, titleS.Render(title), bodyS.Render(q.body))
	return prepQuesContainerStyle.
		BorderStyle(lipgloss.HiddenBorder()).
		Width(smallContainerW() - smallContainerStyle.GetHorizontalFrameSize()).
		Render(ques)
}

func (m *prepSelModel) updateTitleStyleAsFocus(focus bool) {
	if focus {
		m.titleStyle = titleStyle.
			Background(highlightColor).
			Foreground(subduedHighlightColor)
	} else {
		m.titleStyle = titleStyle.
			Background(subduedGrayColor).
			Foreground(highlightColor)
	}
}

func (m prepSelModel) getStatus() string {
	s := fmt.Sprintf("%d Dirs • %d Files • %d Selected",
		m.processMsg.dirs, m.processMsg.files, len(m.processMsg.filenames))
	subW := smallContainerStyle.GetHorizontalFrameSize() + titleStyle.GetHorizontalFrameSize()
	return runewidth.Truncate(s, smallContainerW()-subW, "…")
}

func truncateRenderedTitle(title string) string {
	tail := "…"
	subW := smallContainerStyle.GetVerticalFrameSize() +
		prepQuesContainerStyle.GetHorizontalFrameSize() +
		prepQuesTitleStyle.GetHorizontalFrameSize() +
		lipgloss.Width(tail)
	titleW := smallContainerW() - subW
	return runewidth.Truncate(title, titleW, tail)
}
