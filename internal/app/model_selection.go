package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/k8s-wizard/internal/favourites"
	"github.com/k8s-wizard/internal/ui"
)

// Selection handlers for the main command flow.

func (m Model) handleMainMenuSelection() (tea.Model, tea.Cmd) {
	selected := m.list.SelectedItem()
	if selected == nil {
		return m, nil
	}

	title := selected.(ui.SimpleItem).Title()

	switch title {
	case "Run Command":
		return m.navigateToResourceSelection(), nil
	case "Custom Command":
		return m.navigateToCustomCommand(), nil
	case "Cluster Info":
		m = m.navigateToClusterInfo()
		return m, m.loadClusterInfo()
	case "Favourites":
		return m.navigateToFavouritesList(), nil
	case "Command History":
		return m.navigateToCommandHistory(), nil
	case "Saved Outputs":
		return m.loadSavedOutputs()
	case "Hotkeys":
		return m.navigateToHotkeysList(), nil
	case "Contexts & Namespaces":
		return m.navigateToContextsAndNamespacesMenu(), nil
	case "Check Cluster Connectivity":
		return m, m.checkClusterConnectivity()
	case "Exit":
		return m, tea.Quit
	}

	return m, nil
}

func (m Model) handleResourceSelection() (tea.Model, tea.Cmd) {
	selected := m.list.SelectedItem()
	if selected == nil {
		return m, nil
	}

	title := selected.(ui.SimpleItem).Title()

	switch title {
	case "Pods":
		m.selectedResource = ResourcePods
	case "Deployments":
		m.selectedResource = ResourceDeployments
	case "Services":
		m.selectedResource = ResourceServices
	case "Nodes":
		m.selectedResource = ResourceNodes
	case "ConfigMaps":
		m.selectedResource = ResourceConfigMaps
	case "Secrets":
		m.selectedResource = ResourceSecrets
	case "Ingress":
		m.selectedResource = ResourceIngress
	default:
		return m, nil
	}

	return m.navigateToActionSelection(), nil
}

func (m Model) handleActionSelection() (tea.Model, tea.Cmd) {
	selected := m.list.SelectedItem()
	if selected == nil {
		return m, nil
	}

	title := selected.(ui.SimpleItem).Title()

	switch title {
	case "Get":
		m.selectedAction = ActionGet
		// For 'get' commands, go to flags selection
		return m.navigateToFlagsSelection(), nil

	case "Describe":
		m.selectedAction = ActionDescribe
		// Need to fetch resource names for selection
		return m, m.fetchResourceNames()

	case "Logs":
		m.selectedAction = ActionLogs
		// Need to fetch names for the selected resource
		if m.selectedResource == ResourcePods {
			return m, m.fetchPodNames()
		}
		return m, m.fetchResourceNames()

	case "Extract Field":
		m.selectedAction = ActionExtractField
		// Need to fetch resource names for selection
		return m, m.fetchResourceNames()

	case "Edit":
		m.selectedAction = ActionEdit
		return m, m.fetchResourceNames()

	case "Delete":
		m.selectedAction = ActionDelete
		return m, m.fetchResourceNames()

	case "Exec":
		m.selectedAction = ActionExec
		return m, m.fetchResourceNames()

	case "Port Forward":
		m.selectedAction = ActionPortForward
		return m, m.fetchResourceNames()

	case "Top (Metrics)":
		m.selectedAction = ActionTop
		return m.navigateToFlagsSelection(), nil
	}

	return m, nil
}

func (m Model) handleResourceNameSelection() (tea.Model, tea.Cmd) {
	selected := m.list.SelectedItem()
	if selected == nil {
		return m, nil
	}

	m.selectedResourceName = selected.(ui.SimpleItem).Title()

	if m.selectedAction == ActionExtractField {
		return m, m.fetchSecretKeys()
	}

	if m.selectedAction == ActionDelete {
		return m.navigateToDeleteConfirmation(), nil
	}

	if m.selectedAction == ActionPortForward {
		return m.navigateToPortInput(), nil
	}

	// Go to flags selection
	return m.navigateToFlagsSelection(), nil
}

func (m Model) handleFlagsSelection() (tea.Model, tea.Cmd) {
	selected := m.list.SelectedItem()
	if selected == nil {
		return m, nil
	}

	title := selected.(ui.SimpleItem).Title()

	// Check if user selected "Done"
	if title == "Done (Continue)" {
		// Check if namespace input is needed
		if m.needsNamespaceInput {
			return m.navigateToNamespaceInput(), nil
		}

		// If a default namespace is configured and the user hasn't explicitly
		// chosen a namespace or all-namespaces flag, apply it implicitly.
		if m.defaultNamespace != "" && !m.hasExplicitNamespaceFlag() {
			m.selectedFlags = append(m.selectedFlags, "-n "+m.defaultNamespace)
		}

		// Build command with selected flags (including any implicit namespace)
		m.currentCommand = buildCommand(m.selectedResource, m.selectedAction, m.selectedResourceName, m.selectedFlags)
		// Navigate to command preview
		return m.navigateToCommandPreview(), nil
	}

	// Ignore separator
	if title == "---" {
		return m, nil
	}

	// Toggle flag selection (space bar will call this via handleKeyPress)
	return m.toggleFlag(), nil
}

// toggleFlag toggles the selection state of the current flag.
func (m Model) toggleFlag() Model {
	selected := m.list.SelectedItem()
	if selected == nil {
		return m
	}

	title := selected.(ui.SimpleItem).Title()

	// Ignore Done and separator
	if title == "Done (Continue)" || title == "---" {
		return m
	}

	// Extract flag from title (remove checkbox)
	var flag string
	if len(title) > 4 {
		flag = title[4:] // Remove "[ ] " or "[x] "
	}

	// Special handling for namespace flag
	if flag == "-n <namespace>" {
		// Get current index in list
		idx := m.list.Index()
		items := m.list.Items()

		// Toggle namespace flag
		var newTitle string
		if m.needsNamespaceInput {
			// Deselect namespace
			m.needsNamespaceInput = false
			m.customNamespace = ""
			newTitle = "[ ] -n <namespace>"

			// Remove any existing -n flag from selectedFlags
			for i, f := range m.selectedFlags {
				if len(f) >= 2 && f[:2] == "-n" {
					m.selectedFlags = append(m.selectedFlags[:i], m.selectedFlags[i+1:]...)
					break
				}
			}
		} else {
			// Select namespace (will prompt for input later)
			m.needsNamespaceInput = true
			newTitle = "[x] -n <namespace>"
		}

		// Update list item
		if idx >= 0 && idx < len(items) {
			desc := items[idx].(ui.SimpleItem).Description()
			items[idx] = ui.NewSimpleItem(newTitle, desc)
			m.list.SetItems(items)
		}

		return m
	}

	// Check if flag is already selected
	flagIndex := -1
	for i, f := range m.selectedFlags {
		if f == flag {
			flagIndex = i
			break
		}
	}

	// Get current index in list
	idx := m.list.Index()

	// Toggle flag
	var newTitle string
	if flagIndex >= 0 {
		// Remove flag
		m.selectedFlags = append(m.selectedFlags[:flagIndex], m.selectedFlags[flagIndex+1:]...)
		newTitle = "[ ] " + flag
	} else {
		// Add flag
		m.selectedFlags = append(m.selectedFlags, flag)
		newTitle = "[x] " + flag
	}

	// Update list item
	items := m.list.Items()
	if idx >= 0 && idx < len(items) {
		desc := items[idx].(ui.SimpleItem).Description()
		items[idx] = ui.NewSimpleItem(newTitle, desc)
		m.list.SetItems(items)
	}

	return m
}

func (m Model) handleCommandPreviewSelection() (tea.Model, tea.Cmd) {
	selected := m.list.SelectedItem()
	if selected == nil {
		return m, nil
	}

	title := selected.(ui.SimpleItem).Title()

	switch title {
	case "Execute":
		return m, m.executeCommand()
	case "Help":
		return m, m.loadCommandHelp()
	case "Save as Favourite":
		return m.navigateToSaveFavourite(), nil
	case "Back":
		return m.navigateBack(), nil
	}

	return m, nil
}

func (m Model) handleCommandHistorySelection() (tea.Model, tea.Cmd) {
	if m.historyStore == nil {
		return m, nil
	}
	idx := m.list.Index()
	entry, ok := m.historyStore.Get(idx)
	if ok {
		m.currentCommand = entry.Command
		return m, m.executeCommand()
	}
	return m, nil
}

func (m Model) handleFavouriteSelection() (tea.Model, tea.Cmd) {
	if m.favStore == nil {
		return m, nil
	}

	idx := m.list.Index()
	fav, ok := m.favStore.Get(idx)
	if !ok {
		return m, nil
	}

	// Check if user pressed 'd' to delete
	// This is handled in the key handler, so here we just execute
	m.currentCommand = fav.Command
	return m, m.executeCommand()
}

func (m Model) handleSaveFavourite() (tea.Model, tea.Cmd) {
	name := m.textInput.Value()
	if name == "" {
		return m, nil
	}

	if m.favStore == nil {
		m.err = fmt.Errorf("favourites store not available")
		return m.navigateToMainMenu(), nil
	}

	fav := favourites.NewFavourite(name, m.currentCommand)
	return m, m.saveFavourite(fav)
}

func (m Model) handleRenameFavourite() (tea.Model, tea.Cmd) {
	newName := m.textInput.Value()
	if newName == "" {
		return m, nil
	}

	if m.favStore == nil {
		m.err = fmt.Errorf("favourites store not available")
		return m.navigateToMainMenu(), nil
	}

	return m, m.renameFavourite(m.renamingFavouriteIdx, newName)
}

func (m Model) handleRenameSavedOutput() (tea.Model, tea.Cmd) {
	newName := strings.TrimSpace(m.textInput.Value())
	if newName == "" {
		return m, nil
	}
	newName = strings.TrimSuffix(newName, ".txt")
	if m.renamingSavedOutputIsGroup {
		m.savedOutputsReturnScreen = SavedOutputVersionsScreen
		m.savedOutputsReturnBase = newName
		return m, m.renameSavedOutputGroup(m.renamingSavedOutput, newName)
	}
	return m, m.renameSavedOutput(m.renamingSavedOutput, newName)
}

func (m Model) handleNamespaceInput() (tea.Model, tea.Cmd) {
	namespace := m.textInput.Value()
	if namespace == "" {
		return m, nil
	}

	// Store the namespace value
	m.customNamespace = namespace

	// Add the namespace flag to selected flags
	m.selectedFlags = append(m.selectedFlags, "-n "+namespace)

	// Build command with all flags including namespace
	m.currentCommand = buildCommand(m.selectedResource, m.selectedAction, m.selectedResourceName, m.selectedFlags)

	// Navigate to command preview
	return m.navigateToCommandPreview(), nil
}

func (m Model) hasExplicitNamespaceFlag() bool {
	for _, f := range m.selectedFlags {
		if f == "-A" || strings.HasPrefix(f, "-n ") || strings.HasPrefix(f, "-n=") {
			return true
		}
	}
	return false
}

func (m Model) handleSecretFieldSelection() (tea.Model, tea.Cmd) {
	selected := m.list.SelectedItem()
	if selected == nil {
		return m, nil
	}

	title := selected.(ui.SimpleItem).Title()

	if title == "Custom JSONPath" {
		return m.navigateToCustomCommand(), nil
	}

	if title == "---" {
		return m, nil
	}

	// Build the command to extract the field
	// Different handling for data fields (base64 encoded) vs metadata fields (plain text)
	var templateStr string

	if strings.HasPrefix(title, "data.") {
		// Data fields are base64 encoded and need decoding
		fieldName := strings.TrimPrefix(title, "data.")
		// Escape field name for shell - replace single quotes with '\''
		escapedFieldName := strings.ReplaceAll(fieldName, "'", "'\\''")
		templateStr = fmt.Sprintf("{{index .data \"%s\" | base64decode}}", escapedFieldName)
	} else if strings.HasPrefix(title, "stringData.") {
		// StringData fields are plain text
		fieldName := strings.TrimPrefix(title, "stringData.")
		escapedFieldName := strings.ReplaceAll(fieldName, "'", "'\\''")
		templateStr = fmt.Sprintf("{{index .stringData \"%s\"}}", escapedFieldName)
	} else if strings.HasPrefix(title, "metadata.labels.") {
		// Label fields
		fieldName := strings.TrimPrefix(title, "metadata.labels.")
		escapedFieldName := strings.ReplaceAll(fieldName, "'", "'\\''")
		templateStr = fmt.Sprintf("{{index .metadata.labels \"%s\"}}", escapedFieldName)
	} else if strings.HasPrefix(title, "metadata.annotations.") {
		// Annotation fields
		fieldName := strings.TrimPrefix(title, "metadata.annotations.")
		escapedFieldName := strings.ReplaceAll(fieldName, "'", "'\\''")
		templateStr = fmt.Sprintf("{{index .metadata.annotations \"%s\"}}", escapedFieldName)
	} else if title == "metadata.name" {
		templateStr = "{{.metadata.name}}"
	} else if title == "metadata.namespace" {
		templateStr = "{{.metadata.namespace}}"
	} else if title == "metadata.type" {
		templateStr = "{{.type}}"
	} else {
		// Fallback: try as a data field with base64 decode
		escapedTitle := strings.ReplaceAll(title, "'", "'\\''")
		templateStr = fmt.Sprintf("{{index .data \"%s\" | base64decode}}", escapedTitle)
	}

	m.currentCommand = fmt.Sprintf("kubectl get secret %s -o go-template='%s'", m.selectedResourceName, templateStr)

	if m.customNamespace != "" {
		m.currentCommand += " -n " + m.customNamespace
	} else if m.defaultNamespace != "" && !m.hasExplicitNamespaceFlag() {
		m.currentCommand += " -n " + m.defaultNamespace
	}

	return m.navigateToCommandPreview(), nil
}

func (m Model) handleDeleteConfirmationSelection() (tea.Model, tea.Cmd) {
	selected := m.list.SelectedItem()
	if selected == nil {
		return m, nil
	}

	title := selected.(ui.SimpleItem).Title()

	if title == "Confirm Delete" {
		m.currentCommand = buildCommand(m.selectedResource, m.selectedAction, m.selectedResourceName, m.selectedFlags)
		return m, m.executeCommand()
	}

	// Cancel - go back to name selection
	return m, m.fetchResourceNames()
}

func (m Model) handlePortInput() (tea.Model, tea.Cmd) {
	ports := m.textInput.Value()
	if ports == "" {
		return m, nil
	}

	// Build command with ports
	m.currentCommand = buildCommand(m.selectedResource, m.selectedAction, m.selectedResourceName, m.selectedFlags)
	m.currentCommand += " " + ports

	// Navigate to command preview
	return m.navigateToCommandPreview(), nil
}

func (m Model) handleCustomCommandInput() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.textInput.Value())
	if input == "" {
		return m, nil
	}

	// Allow users to type either full "kubectl ..." or just the arguments.
	if strings.HasPrefix(input, "kubectl ") {
		m.currentCommand = input
	} else {
		m.currentCommand = "kubectl " + input
	}

	return m.navigateToCommandPreview(), nil
}
