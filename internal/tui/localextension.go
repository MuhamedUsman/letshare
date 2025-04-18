package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"path/filepath"
)

// localExtensionModel manages extensions for the local Space component.
// the local Space currently contains a single model (dirNavigationModel),
// which can be extended with additional functionality. this extension model
// provides a container for these extensions (currently only extendDirModel).
// localSpaceModel(dirNavigationModel) ->  localExtensionModel(extendDirModel)
// tea.Cmd and tea.Msg should be used to communicate between the models and their extensions
// This architectural pattern separates core navigation (dirNavigationModel) from
// extended functionality (extendDirModel), making the codebase more maintainable
// and extensible. future extensions can be added to this model without modifying
// the core navigation logic.
type localExtensionModel struct {
	extendDir     extendDirModel
	disableKeymap bool
}

func initialLocalExtensionModel() localExtensionModel {
	return localExtensionModel{
		extendDir: initialExtendDirModel(),
	}
}

func (m localExtensionModel) Init() tea.Cmd {
	return m.extendDir.Init()
}

func (m localExtensionModel) Update(msg tea.Msg) (localExtensionModel, tea.Cmd) {
	m.extendDir.disableKeymap = m.disableKeymap
	var cmd tea.Cmd
	m.extendDir, cmd = m.extendDir.Update(msg)
	return m, cmd
}

func (m localExtensionModel) View() string {
	return m.extendDir.View()
}

func (m localExtensionModel) getTitle() string {
	return "Extended Dir: " + filepath.Base(m.extendDir.dirPath)
}

func (m localExtensionModel) grantExtensionSwitch(space focusedSpace) tea.Cmd {
	// ask child models to grant extension switch
	return m.extendDir.grantExtensionSwitch(space)
}

func (m localExtensionModel) grantSpaceFocusSwitch(space focusedSpace) tea.Cmd {
	// ask child models to grant space focus switch
	return m.extendDir.grantSpaceFocusSwitch(space)
}

func (m *localExtensionModel) updateKeymap(disable bool) {
	m.disableKeymap = disable
	m.extendDir.updateKeymap(disable)
}
