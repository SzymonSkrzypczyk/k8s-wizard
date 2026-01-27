package ui

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// NewViewport creates a new viewport for displaying scrollable content
func NewViewport(width, height int) viewport.Model {
	vp := viewport.New(width, height)
	return vp
}

// UpdateViewport is a helper to update a viewport model
func UpdateViewport(vp viewport.Model, msg tea.Msg) (viewport.Model, tea.Cmd) {
	newVp, cmd := vp.Update(msg)
	return newVp, cmd
}

// SetViewportContent sets the content of a viewport
func SetViewportContent(vp *viewport.Model, content string) {
	vp.SetContent(content)
}
