package ui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// SimpleItem implements list.Item for simple string lists
type SimpleItem struct {
	title string
	desc  string
}

func (i SimpleItem) Title() string       { return i.title }
func (i SimpleItem) Description() string { return i.desc }
func (i SimpleItem) FilterValue() string { return i.title }

// NewSimpleItem creates a new simple list item
func NewSimpleItem(title, desc string) SimpleItem {
	return SimpleItem{title: title, desc: desc}
}

// NewList creates a new list with the given items and title
func NewList(items []list.Item, title string, width, height int) list.Model {
	l := list.New(items, list.NewDefaultDelegate(), width, height)
	l.Title = title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	return l
}

// StringsToItems converts a slice of strings to list items
func StringsToItems(strings []string) []list.Item {
	items := make([]list.Item, len(strings))
	for i, s := range strings {
		items[i] = NewSimpleItem(s, "")
	}
	return items
}

// Update is a helper to update a list model
func UpdateList(l list.Model, msg tea.Msg) (list.Model, tea.Cmd) {
	newList, cmd := l.Update(msg)
	return newList, cmd
}
