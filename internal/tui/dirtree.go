package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"os"
)

type stack struct {
	s    []string
	push func(string)
	pop  func() string
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
	}
}

type dirTree struct {
	dirStack stack
}

func newDirTree() dirTree {
	return dirTree{dirStack: newStack()}
}

func (m dirTree) Init() tea.Cmd {
	return nil
}

func (m dirTree) Update(msg tea.Msg) (dirTree, tea.Cmd) {
	return m, nil
}

func (m dirTree) View() string {
	return ""
}

func hasNestedDir(path string) bool {
	entries, _ := os.ReadDir(path)
	for _, entry := range entries {
		if entry.IsDir() {
			return true
		}
	}
	return false
}
