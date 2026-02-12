package app

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/k8s-wizard/internal/ui"
)

// Contexts & namespaces navigation and selection.

func (m Model) navigateToContextsAndNamespacesMenu() Model {
	items := []list.Item{
		ui.NewSimpleItem("Switch Context", "Switch the current kube context"),
		ui.NewSimpleItem("Set Default Namespace", "Choose a default namespace for commands"),
		ui.NewSimpleItem("Back to Main Menu", "Return to the main menu"),
	}
	m.list = ui.NewList(items, "Contexts & Namespaces", m.width, m.height-4)
	m.previousScreen = m.currentScreen
	m.currentScreen = ContextsNamespacesMenuScreen
	return m
}

func (m Model) navigateToContextsList() Model {
	items := []list.Item{}

	currentCtx, err := m.kubectlClient.GetCurrentContext()
	if err != nil {
		m.err = err
	}

	contexts, listErr := m.kubectlClient.ListContexts()
	if listErr != nil {
		m.err = listErr
		items = []list.Item{
			ui.NewSimpleItem("Unable to load contexts", listErr.Error()),
		}
	} else if len(contexts) == 0 {
		items = []list.Item{
			ui.NewSimpleItem("No contexts found", "Configure kubeconfig to add contexts"),
		}
	} else {
		for _, name := range contexts {
			desc := ""
			if name == currentCtx {
				desc = "(current)"
			}
			items = append(items, ui.NewSimpleItem(name, desc))
		}
	}

	m.list = ui.NewList(items, "Kube Contexts (Enter=switch)", m.width, m.height-4)
	m.previousScreen = m.currentScreen
	m.currentScreen = ContextsListScreen
	return m
}

func (m Model) navigateToNamespacesList() Model {
	items := []list.Item{}

	namespaces, err := m.kubectlClient.ListNamespaceNames()
	if err != nil {
		m.err = err
		items = []list.Item{
			ui.NewSimpleItem("Unable to load namespaces", err.Error()),
		}
	} else if len(namespaces) == 0 {
		items = []list.Item{
			ui.NewSimpleItem("No namespaces found", "Create namespaces to select a default"),
		}
	} else {
		for _, ns := range namespaces {
			desc := ""
			if ns == m.defaultNamespace {
				desc = "(current default)"
			}
			items = append(items, ui.NewSimpleItem(ns, desc))
		}
	}

	m.list = ui.NewList(items, "Namespaces (Enter=set default)", m.width, m.height-4)
	m.previousScreen = m.currentScreen
	m.currentScreen = NamespacesListScreen
	return m
}

func (m Model) handleContextsAndNamespacesMenuSelection() (tea.Model, tea.Cmd) {
	selected := m.list.SelectedItem()
	if selected == nil {
		return m, nil
	}

	title := selected.(ui.SimpleItem).Title()

	switch title {
	case "Switch Context":
		return m.navigateToContextsList(), nil
	case "Set Default Namespace":
		return m.navigateToNamespacesList(), nil
	case "Back to Main Menu":
		return m.navigateToMainMenu(), nil
	}

	return m, nil
}

func (m Model) handleContextSelection() (tea.Model, tea.Cmd) {
	selected := m.list.SelectedItem()
	if selected == nil {
		return m, nil
	}

	title := selected.(ui.SimpleItem).Title()
	if title == "Unable to load contexts" || title == "No contexts found" {
		return m, nil
	}

	return m, m.switchContext(title)
}

func (m Model) handleNamespaceSelection() (tea.Model, tea.Cmd) {
	selected := m.list.SelectedItem()
	if selected == nil {
		return m, nil
	}

	title := selected.(ui.SimpleItem).Title()
	if title == "Unable to load namespaces" || title == "No namespaces found" {
		return m, nil
	}

	m.defaultNamespace = title
	m.err = fmt.Errorf("✓ Default namespace set to %s", title)
	return m.navigateToContextsAndNamespacesMenu(), nil
}

func (m Model) switchContext(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.kubectlClient.UseContext(name)
		return contextSwitchedMsg{newContext: name, err: err}
	}
}
