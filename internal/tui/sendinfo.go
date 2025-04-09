package tui

import (
	"github.com/MuhamedUsman/letshare/internal/tui/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"math"
)

type sendInfoModel struct {
	infoTable table.Model
}

func initialSendInfoModel() sendInfoModel {
	t := table.New(table.WithFocused(true))
	s := table.DefaultStyles()
	t.SetStyles(table.Styles{
		Header: s.Header.
			Foreground(highlightColor).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(subduedHighlightColor).
			BorderBottom(true),
		Selected: s.Selected.
			Background(subduedHighlightColor).
			Foreground(highlightColor).
			Italic(true).
			Bold(true),
		Cell: s.Cell.Foreground(midHighlightColor),
	})
	r := []table.Row{
		{"File One", "PDF", "10MB"},
		{"File Two", "MP4", "500MB"},
		{"File Three", "JPEG", "2.5MB"},
	}
	t.SetColumns(getTableCols(0))
	t.SetRows(r)
	return sendInfoModel{infoTable: t}
}

func getTableCols(tableWidth int) []table.Column {
	/*nameSpace, typeSpace, sizeSpace := 50, 21, 21
	nameW := (tableWidth * nameSpace) / 100
	typeW := (tableWidth * typeSpace) / 100
	sizeW := (tableWidth * sizeSpace) / 100
	return []table.Column{
		{"Name", nameW},
		{"Type", typeW},
		{"Size", sizeW},
	}*/
	tableWidth = tableWidth - 5
	nameCellW := (tableWidth * 60) / 100
	widthLeft := tableWidth - nameCellW
	typeCellW := int(math.Round(float64(widthLeft / 2)))
	sizeCellW := int(math.Round(float64(widthLeft / 2)))
	return []table.Column{
		{"Name", nameCellW},
		{"Type", typeCellW},
		{"Size", sizeCellW},
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

		}
	}
	return m, m.handleInfoTableUpdate(msg)
}

func (m sendInfoModel) View() string {
	return tableBaseStyle.Render(m.infoTable.View())
}

func (m *sendInfoModel) updateDimensions() {
	w := largeContainerW() - infoContainerStyle.GetHorizontalFrameSize()
	h := termH - (mainContainerStyle.GetVerticalFrameSize() + infoContainerStyle.GetVerticalFrameSize())
	m.infoTable.SetWidth(w)
	m.infoTable.SetHeight(h - 2)
	m.infoTable.SetColumns(getTableCols(m.infoTable.Width()))
}

func (m *sendInfoModel) handleInfoTableUpdate(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.infoTable, cmd = m.infoTable.Update(msg)
	return cmd
}

func applyCellStyle(cell string) string {
	return lipgloss.NewStyle().
		Foreground(highlightColor).
		Faint(true).
		Render(cell)
}
