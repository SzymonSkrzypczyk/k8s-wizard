package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/k8s-wizard/internal/favourites"
	"github.com/k8s-wizard/internal/ui"
)

// Helpers and navigation for favourites and hotkeys.

func (m Model) isTextInputScreen() bool {
	switch m.currentScreen {
	case SaveFavouriteScreen, RenameFavouriteScreen, RenameSavedOutputScreen, NamespaceInputScreen, SaveOutputNameScreen:
		return true
	default:
		return false
	}
}

func (m Model) tryParseHotkey(key string) (string, bool) {
	key = strings.TrimSpace(strings.ToUpper(key))
	switch key {
	case "F1", "F2", "F3", "F4", "F5", "F6", "F7", "F8", "F9", "F10", "F11", "F12":
		return key, true
	default:
		return "", false
	}
}

func (m Model) navigateToHotkeysList() Model {
	items := []list.Item{}
	if m.hotkeyStore == nil {
		items = []list.Item{ui.NewSimpleItem("Hotkeys unavailable", "")}
		m.list = ui.NewList(items, "Hotkeys", m.width, m.height-4)
		m.previousScreen = m.currentScreen
		m.currentScreen = HotkeysListScreen
		return m
	}

	keys := []string{"F1", "F2", "F3", "F4", "F5", "F6", "F7", "F8", "F9", "F10", "F11", "F12"}
	for _, k := range keys {
		if b, ok := m.hotkeyStore.Get(k); ok {
			items = append(items, ui.NewSimpleItem(k, b.Name))
		} else {
			items = append(items, ui.NewSimpleItem(k, "(unbound)"))
		}
	}
	if len(items) == 0 {
		items = []list.Item{ui.NewSimpleItem("No hotkeys bound", "")}
	}
	m.list = ui.NewList(items, "Hotkeys ('d'=unbind, Esc=back)", m.width, m.height-4)
	m.previousScreen = m.currentScreen
	m.currentScreen = HotkeysListScreen
	return m
}

func (m Model) navigateToFavouritesList() Model {
	if m.favStore == nil {
		m.err = fmt.Errorf("favourites store not available")
		return m.navigateToMainMenu()
	}

	favs := m.favStore.List()
	items := make([]list.Item, len(favs))
	for i, fav := range favs {
		items[i] = ui.NewSimpleItem(fav.Name, fav.Command)
	}

	if len(items) == 0 {
		items = []list.Item{
			ui.NewSimpleItem("No favourites saved", "Save a command from the preview screen or history to see it here"),
		}
	}

	m.list = ui.NewList(items, "Favourites (Enter=run, 'd'=delete, 'r'=rename, 'h'=bind hotkey)", m.width, m.height-4)
	m.previousScreen = m.currentScreen
	m.currentScreen = FavouritesListScreen
	return m
}

func (m Model) saveFavourite(fav favourites.Favourite) tea.Cmd {
	return func() tea.Msg {
		err := m.favStore.Add(fav)
		return favouriteSavedMsg{err: err}
	}
}

func (m Model) deleteFavourite(idx int) tea.Cmd {
	return func() tea.Msg {
		err := m.favStore.Delete(idx)
		return favouriteDeletedMsg{err: err}
	}
}

func (m Model) renameFavourite(idx int, newName string) tea.Cmd {
	return func() tea.Msg {
		err := m.favStore.Rename(idx, newName)
		return favouriteRenamedMsg{err: err}
	}
}
