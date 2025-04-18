package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

type extensionSpaceModel struct {
	localExtension           localExtensionModel
	titleStyle               lipgloss.Style
	extendedSpace            focusedSpace
	dirToExtend              string
	hideTitle, disableKeymap bool
}

func initialExtensionSpaceModel() extensionSpaceModel {
	return extensionSpaceModel{
		localExtension: initialLocalExtensionModel(),
		titleStyle:     titleStyle,
	}
}

func (m extensionSpaceModel) Init() tea.Cmd {
	return m.localExtension.Init()
}

func (m extensionSpaceModel) Update(msg tea.Msg) (extensionSpaceModel, tea.Cmd) {

	switch msg := msg.(type) {

	case tea.KeyMsg:
		if m.disableKeymap {
			return m, nil
		}
		switch msg.String() {

		case "esc":
			// this must be run in a sequence, we need to get grant access on
			// current state of the localExtension model before it handles esc key msg
			return m, tea.Sequence(m.localExtension.grantExtensionSwitch(home), m.handleChildModelUpdate(msg))

		case "backspace":
			return m, tea.Sequence(m.localExtension.grantSpaceFocusSwitch(local), m.handleChildModelUpdate(msg))

		}

	case extendSpaceMsg:
		m.updateTitleStyleAsFocus(msg.focus)
		m.extendedSpace = msg.space
		if msg.space == home {
			m.disableKeymap = true
		}
		if msg.focus {
			currentFocus = extension
			return m, spaceFocusSwitchMsg(extension).cmd
		}

	case spaceFocusSwitchMsg:
		if focusedSpace(msg) == extension {
			m.updateTitleStyleAsFocus(true)
		} else {
			m.updateTitleStyleAsFocus(false)
		}

	case hideInfoSpaceTitle:
		m.hideTitle = bool(msg)

	}
	return m, m.handleChildModelUpdate(msg)
}

func (m extensionSpaceModel) View() string {
	title := "Home Space"
	infoContent := banner.Height(infoContainerWorkableH(!m.hideTitle)).Render() // !m.hideTitle == showTitle
	if m.extendedSpace == local {
		title = m.localExtension.getTitle()
		tail := "…"
		w := largeContainerW() - (lipgloss.Width(tail) + titleStyle.GetHorizontalPadding() + 1) // +1 experimental
		title = runewidth.Truncate(title, w, tail)
		infoContent = m.localExtension.View()
	}
	if m.extendedSpace == remote {
		title = "Extended Remote Space"
		infoContent = ""
	}
	title = m.titleStyle.Render(title)
	style := infoContainerStyle.
		Width(largeContainerW()).
		Height(termH - (mainContainerStyle.GetVerticalFrameSize()))
	if m.hideTitle {
		return style.Render(infoContent)
	}
	return style.Render(title, infoContent)
}

func (m *extensionSpaceModel) handleChildModelUpdate(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.localExtension, cmd = m.localExtension.Update(msg)
	return cmd
}

func (m *extensionSpaceModel) updateTitleStyleAsFocus(focus bool) {
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

func (m *extensionSpaceModel) updateKeymap(disable bool) {
	m.disableKeymap = disable
	m.localExtension.updateKeymap(disable || m.extendedSpace == home)
}
