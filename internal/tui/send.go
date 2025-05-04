package tui

import (
	"errors"
	"github.com/MuhamedUsman/letshare/internal/client"
	"github.com/MuhamedUsman/letshare/internal/zipr"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

type selections struct {
	rootPath    string
	filenames   []string
	dirs, files int
}
type sendModel struct {
	selections    *selections
	progress      progress.Model
	titleStyle    lipgloss.Style
	disableKeymap bool
}

func (m sendModel) capturesKeyEvent(msg tea.KeyMsg) bool {
	return false
}

func initialSendModel() sendModel {
	return sendModel{
		titleStyle: titleStyle.Margin(0, 2),
	}
}

func (m sendModel) Init() tea.Cmd {
	return nil
}

func (m sendModel) Update(msg tea.Msg) (sendModel, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		if m.disableKeymap {
			return m, nil
		}
		switch msg.String() {

		}

	case spaceFocusSwitchMsg:
		if currentFocus == local {
			m.titleStyle = m.titleStyle.
				Background(highlightColor).
				Foreground(subduedHighlightColor)
		} else {
			m.titleStyle = m.titleStyle.
				Background(subduedGrayColor).
				Foreground(highlightColor)
		}

	case processSelectionsMsg:
		m.selections = &selections{
			rootPath:  msg.parentPath,
			filenames: msg.filenames,
			dirs:      msg.dirs,
			files:     msg.files,
		}
		return m, localChildSwitchMsg{child: send, focus: true}.cmd
	}

	return m, nil
}

func (m sendModel) View() string {
	s := m.titleStyle.Render("Local Space")
	s = runewidth.Truncate(s, runewidth.StringWidth(s), "…")
	return smallContainerStyle.Width(smallContainerW()).Render(s)
}

func (m *sendModel) handleUpdate(msg tea.Msg) tea.Cmd {
	return nil
}

func (m *sendModel) updateKeymap(disable bool) {
	m.disableKeymap = disable
}

func (m sendModel) processFiles(msg processSelectionsMsg) tea.Cmd {
	cfg, err := client.GetConfig()
	if errors.Is(err, client.ErrNoConfig) {
		cfg, err = client.LoadConfig()
	}
	return func() tea.Msg {
		if err != nil {
			return errMsg{err: err, fatal: true}
		}

		if cfg.Share.ZipFiles {
			progressChan := make(chan uint64)
			zipr.New(progressChan, zipr.Deflate)
			//zipr.(progressChan, cfg.Share.SharedZipName, m.selections.rootPath, m.selections.filenames...)
		}
		return nil
	}
}
