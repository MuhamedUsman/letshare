package tui

import (
	"github.com/MuhamedUsman/letshare/internal/tui/table"
	tea "github.com/charmbracelet/bubbletea"
)

type sendInfoModel struct {
	infoTable table.Model
}

func initialSendInfoModel() sendInfoModel {
	r := []table.Row{
		{"File One", "PDF", "10MB"},
		{"File Two", "MP4", "500MB"},
		{"File Three", "JPEG", "2.5MB"},
	}
	t := table.New(
		table.WithStyles(customTableStyles),
		table.WithColumns(getTableCols(0)),
		table.WithRows(r),
	)
	return sendInfoModel{infoTable: t}
}

func getTableCols(tableWidth int) []table.Column {
	cols := []string{"Name", "Type", "Size"}
	subW := customTableStyles.Cell.GetHorizontalFrameSize() * len(cols)
	tableWidth = tableWidth - subW
	smallCellW := (tableWidth * 20) / 100
	largeCellW := tableWidth - smallCellW*2
	return []table.Column{
		{cols[0], largeCellW},
		{cols[1], smallCellW},
		{cols[2], smallCellW},
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

		case "tab", "shift+tab":
			if currentFocus == info && extendSpace == send {
				m.infoTable.Focus()
			} else {
				m.infoTable.Blur()
			}

		}

	case extentDirMsg:

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

/*func (sendInfoModel) readDir(path string) tea.Cmd {
	return func() tea.Msg {
		entries, err := os.ReadDir(path)
		if err != nil {
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
		}

	}
}*/
