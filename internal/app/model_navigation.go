package app

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/k8s-wizard/internal/ui"
)

// Navigation handlers for the main application flow.

func (m Model) navigateToMainMenu() Model {
	items := []list.Item{
		ui.NewSimpleItem("Run Command", "Execute kubectl commands"),
		ui.NewSimpleItem("Custom Command", "Build an advanced kubectl command"),
		ui.NewSimpleItem("Cluster Info", "View cluster information and metrics"),
		ui.NewSimpleItem("Favourites", "View and run saved commands"),
		ui.NewSimpleItem("Command History", "View and re-run previous commands"),
		ui.NewSimpleItem("Saved Outputs", "View previously saved outputs"),
		ui.NewSimpleItem("Hotkeys", "Manage hotkey bindings"),
		ui.NewSimpleItem("Contexts & Namespaces", "Manage kube contexts and default namespace"),
		ui.NewSimpleItem("Check Cluster Connectivity", "Verify connection to Kubernetes cluster"),
		ui.NewSimpleItem("Exit", "Quit the application"),
	}
	m.list = ui.NewList(items, "Kubernetes Wizard", m.width, m.height-4)

	// Reset wizard selections when returning to the main menu to avoid stale state
	m.selectedResource = 0
	m.selectedAction = 0
	m.selectedResourceName = ""
	m.selectedFlags = nil
	m.customNamespace = ""
	m.needsNamespaceInput = false
	m.currentCommand = ""

	m.previousScreen = m.currentScreen
	m.currentScreen = MainMenuScreen
	m.err = nil
	return m
}

func (m Model) navigateToCommandHistory() Model {
	items := []list.Item{}
	if m.historyStore == nil {
		items = []list.Item{
			ui.NewSimpleItem("History unavailable", "Command history could not be loaded"),
		}
		m.list = ui.NewList(items, "Command History", m.width, m.height-4)
		m.previousScreen = m.currentScreen
		m.currentScreen = CommandHistoryScreen
		return m
	}

	entries := m.historyStore.List()
	if len(entries) == 0 {
		items = []list.Item{
			ui.NewSimpleItem("No command history", "Run some commands to see them here"),
		}
	} else {
		for _, entry := range entries {
			timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")
			items = append(items, ui.NewSimpleItem(entry.Command, timestamp))
		}
	}
	m.list = ui.NewList(items, "Command History (Enter=run, 's'=save as favourite, Esc=back)", m.width, m.height-4)
	m.previousScreen = m.currentScreen
	m.currentScreen = CommandHistoryScreen
	return m
}

func (m Model) navigateToCustomCommand() Model {
	// Reuse the text input to capture a free-form kubectl command.
	m.textInput.SetValue("")
	m.textInput.Placeholder = "e.g. get pods -n default"
	m.textInput.Focus()
	m.previousScreen = m.currentScreen
	m.currentScreen = CustomCommandScreen
	m.currentCommand = ""
	return m
}

func (m Model) navigateToResourceSelection() Model {
	// Starting a new command flow: reset selections from any previous run
	m.selectedResource = 0
	m.selectedAction = 0
	m.selectedResourceName = ""
	m.selectedFlags = nil
	m.customNamespace = ""
	m.needsNamespaceInput = false
	m.currentCommand = ""

	items := []list.Item{
		ui.NewSimpleItem("Pods", "Manage pods"),
		ui.NewSimpleItem("Deployments", "Manage deployments"),
		ui.NewSimpleItem("Services", "Inspect services"),
		ui.NewSimpleItem("Nodes", "Inspect cluster nodes"),
		ui.NewSimpleItem("ConfigMaps", "Inspect configuration data"),
		ui.NewSimpleItem("Secrets", "Inspect secrets (careful: may show sensitive data)"),
		ui.NewSimpleItem("Ingress", "Inspect ingress resources"),
	}
	m.list = ui.NewList(items, "Select Resource Type", m.width, m.height-4)
	m.previousScreen = m.currentScreen
	m.currentScreen = ResourceSelectionScreen
	return m
}

func (m Model) navigateToActionSelection() Model {
	var items []list.Item

	switch m.selectedResource {
	case ResourcePods:
		items = []list.Item{
			ui.NewSimpleItem("Get", "List all pods"),
			ui.NewSimpleItem("Describe", "Describe a specific pod"),
			ui.NewSimpleItem("Logs", "View logs from a pod"),
			ui.NewSimpleItem("Exec", "Execute shell in a pod"),
			ui.NewSimpleItem("Port Forward", "Forward local port to pod"),
			ui.NewSimpleItem("Edit", "Edit pod YAML"),
			ui.NewSimpleItem("Delete", "Delete a pod"),
		}
	case ResourceDeployments:
		items = []list.Item{
			ui.NewSimpleItem("Get", "List all deployments"),
			ui.NewSimpleItem("Describe", "Describe a specific deployment"),
			ui.NewSimpleItem("Logs", "View logs for a deployment"),
			ui.NewSimpleItem("Exec", "Execute shell in a deployment pod"),
			ui.NewSimpleItem("Port Forward", "Forward local port to deployment"),
			ui.NewSimpleItem("Edit", "Edit deployment YAML"),
			ui.NewSimpleItem("Delete", "Delete a deployment"),
		}
	case ResourceServices:
		items = []list.Item{
			ui.NewSimpleItem("Get", "List all services"),
			ui.NewSimpleItem("Describe", "Describe a specific service"),
			ui.NewSimpleItem("Port Forward", "Forward local port to service"),
			ui.NewSimpleItem("Edit", "Edit service YAML"),
			ui.NewSimpleItem("Delete", "Delete a service"),
		}
	case ResourceNodes:
		items = []list.Item{
			ui.NewSimpleItem("Get", "List all nodes"),
			ui.NewSimpleItem("Describe", "Describe a specific node"),
			ui.NewSimpleItem("Edit", "Edit node YAML"),
			ui.NewSimpleItem("Delete", "Delete a node"),
		}
	case ResourceConfigMaps:
		items = []list.Item{
			ui.NewSimpleItem("Get", "List all configmaps"),
			ui.NewSimpleItem("Describe", "Describe a specific configmap"),
			ui.NewSimpleItem("Edit", "Edit configmap YAML"),
			ui.NewSimpleItem("Delete", "Delete a configmap"),
		}
	case ResourceSecrets:
		items = []list.Item{
			ui.NewSimpleItem("Get", "List all secrets"),
			ui.NewSimpleItem("Describe", "Describe a specific secret (may reveal sensitive data)"),
			ui.NewSimpleItem("Extract Field", "Pick a field to decode and view"),
			ui.NewSimpleItem("Edit", "Edit secret YAML"),
			ui.NewSimpleItem("Delete", "Delete a secret"),
		}
	case ResourceIngress:
		items = []list.Item{
			ui.NewSimpleItem("Get", "List all ingress resources"),
			ui.NewSimpleItem("Describe", "Describe a specific ingress"),
			ui.NewSimpleItem("Edit", "Edit ingress YAML"),
			ui.NewSimpleItem("Delete", "Delete an ingress"),
		}
	default:
		items = []list.Item{
			ui.NewSimpleItem("Get", "List resources"),
		}
	}

	m.list = ui.NewList(items, "Select Action", m.width, m.height-4)
	m.previousScreen = m.currentScreen
	m.currentScreen = ActionSelectionScreen
	return m
}

func (m Model) navigateToPortInput() Model {
	m.textInput.SetValue("")
	m.textInput.Placeholder = "Enter ports (e.g. 8080:80)"
	m.textInput.Focus()
	m.previousScreen = m.currentScreen
	m.currentScreen = PortInputScreen
	return m
}

func (m Model) navigateToDeleteConfirmation() Model {
	items := []list.Item{
		ui.NewSimpleItem("Cancel", "Go back without deleting"),
		ui.NewSimpleItem("Confirm Delete", fmt.Sprintf("Permanently delete %s %s", getResourceShortName(m.selectedResource), m.selectedResourceName)),
	}
	title := fmt.Sprintf("⚠️  CONFIRM DELETION: %s %s", getResourceShortName(m.selectedResource), m.selectedResourceName)
	m.list = ui.NewList(items, title, m.width, m.height-4)
	m.previousScreen = m.currentScreen
	m.currentScreen = DeleteConfirmationScreen
	return m
}

func (m Model) navigateToCommandPreview() Model {
	items := []list.Item{
		ui.NewSimpleItem("Execute", "Run the command"),
		ui.NewSimpleItem("Help", "Show --help output"),
		ui.NewSimpleItem("Save as Favourite", "Save for later use"),
		ui.NewSimpleItem("Back", "Return to previous screen"),
	}
	m.list = ui.NewList(items, "Command Preview", m.width, m.height-4)
	m.previousScreen = m.currentScreen
	m.currentScreen = CommandPreviewScreen
	return m
}

func (m Model) navigateToFlagsSelection() Model {
	// Reset selected flags and namespace
	m.selectedFlags = []string{}
	m.customNamespace = ""
	m.needsNamespaceInput = false

	// Build list of common flags based on action
	var items []list.Item

	switch m.selectedAction {
	case ActionGet:
		items = []list.Item{
			ui.NewSimpleItem("Done (Continue)", "Proceed with selected flags"),
			ui.NewSimpleItem("---", ""),
			ui.NewSimpleItem("[ ] -o wide", "Show additional columns"),
			ui.NewSimpleItem("[ ] -o yaml", "Output in YAML format"),
			ui.NewSimpleItem("[ ] -o json", "Output in JSON format"),
			ui.NewSimpleItem("[ ] --show-labels", "Show labels"),
			ui.NewSimpleItem("[ ] -A", "All namespaces"),
			ui.NewSimpleItem("[ ] -n <namespace>", "Specify custom namespace"),
		}
	case ActionDescribe:
		items = []list.Item{
			ui.NewSimpleItem("Done (Continue)", "Proceed with selected flags"),
			ui.NewSimpleItem("---", ""),
			ui.NewSimpleItem("[ ] --show-events=true", "Show events"),
			ui.NewSimpleItem("[ ] -n <namespace>", "Specify custom namespace"),
		}
	case ActionLogs:
		items = []list.Item{
			ui.NewSimpleItem("Done (Continue)", "Proceed with selected flags"),
			ui.NewSimpleItem("---", ""),
			ui.NewSimpleItem("[ ] -f", "Follow log output"),
			ui.NewSimpleItem("[ ] --tail=100", "Show last 100 lines"),
			ui.NewSimpleItem("[ ] --tail=50", "Show last 50 lines"),
			ui.NewSimpleItem("[ ] --since=1h", "Show logs from last hour"),
			ui.NewSimpleItem("[ ] --since=5m", "Show logs from last 5 minutes"),
			ui.NewSimpleItem("[ ] --previous", "Show logs from previous container"),
			ui.NewSimpleItem("[ ] -n <namespace>", "Specify custom namespace"),
		}
	}

	m.list = ui.NewList(items, "Select Flags (Space to toggle, Enter when done)", m.width, m.height-4)
	m.previousScreen = m.currentScreen
	m.currentScreen = FlagsSelectionScreen
	return m
}

func (m Model) navigateToNamespaceInput() Model {
	m.textInput.SetValue("")
	m.textInput.Placeholder = "Enter namespace name"
	m.textInput.Focus()
	m.previousScreen = m.currentScreen
	m.currentScreen = NamespaceInputScreen
	return m
}

func (m Model) navigateToSecretFieldSelection(keys []string) Model {
	items := []list.Item{
		ui.NewSimpleItem("Custom JSONPath", "Enter a custom JSONPath (e.g. .metadata.labels)"),
		ui.NewSimpleItem("---", ""),
	}

	for _, k := range keys {
		var description string
		if strings.HasPrefix(k, "data.") {
			description = "Extract and decode this base64-encoded data field"
		} else if strings.HasPrefix(k, "stringData.") {
			description = "Extract this plain text field"
		} else if strings.HasPrefix(k, "metadata.") {
			description = "Extract this metadata field"
		} else {
			description = "Extract this field"
		}
		items = append(items, ui.NewSimpleItem(k, description))
	}

	m.list = ui.NewList(items, "Select Field to Extract", m.width, m.height-4)
	m.previousScreen = m.currentScreen
	m.currentScreen = SecretFieldSelectionScreen
	return m
}

func (m Model) navigateToSaveFavourite() Model {
	m.textInput.SetValue("")
	m.textInput.Placeholder = "Enter favourite name"
	m.textInput.Focus()
	m.previousScreen = m.currentScreen
	m.currentScreen = SaveFavouriteScreen
	return m
}

func (m Model) navigateToRenameFavourite(idx int) Model {
	if m.favStore == nil {
		return m
	}

	fav, ok := m.favStore.Get(idx)
	if !ok {
		return m
	}

	m.renamingFavouriteIdx = idx
	m.textInput.SetValue(fav.Name)
	m.textInput.Focus()
	m.previousScreen = m.currentScreen
	m.currentScreen = RenameFavouriteScreen
	return m
}

func (m Model) navigateBack() Model {
	switch m.currentScreen {
	case ResourceSelectionScreen:
		return m.navigateToMainMenu()
	case ActionSelectionScreen:
		return m.navigateToResourceSelection()
	case ResourceNameSelectionScreen:
		return m.navigateToActionSelection()
	case FlagsSelectionScreen:
		// Always return to the action selection from flags to keep navigation consistent
		return m.navigateToActionSelection()
	case CommandPreviewScreen:
		return m.navigateToFlagsSelection()
	case CommandHelpScreen:
		return m.navigateToCommandPreview()
	case ClusterInfoScreen:
		return m.navigateToMainMenu()
	case ClusterConnectivityScreen:
		return m.navigateToMainMenu()
	case CommandHistoryScreen:
		return m.navigateToMainMenu()
	case HotkeysListScreen:
		return m.navigateToMainMenu()
	case HotkeyBindScreen:
		m.hotkeyBindingPending = false
		return m.navigateToFavouritesList()
	case FavouritesListScreen:
		return m.navigateToMainMenu()
	case SaveFavouriteScreen:
		return m.navigateToCommandPreview()
	case RenameFavouriteScreen:
		return m.navigateToFavouritesList()
	case SecretFieldSelectionScreen:
		return m.navigateToActionSelection()
	case NamespaceInputScreen:
		return m.navigateToFlagsSelection()
	case SavedOutputsListScreen:
		return m.navigateToMainMenu()
	case SavedOutputVersionsScreen:
		return m.navigateToSavedOutputsGroups()
	case SavedOutputViewScreen:
		if m.previousScreen == SavedOutputVersionsScreen && m.selectedSavedOutputBase != "" {
			return m.navigateToSavedOutputVersions(m.selectedSavedOutputBase)
		}
		return m.navigateToSavedOutputsGroups()
	case RenameSavedOutputScreen:
		if m.renamingSavedOutputIsGroup {
			return m.navigateToSavedOutputVersions(m.renamingSavedOutput)
		}
		return m.navigateToSavedOutputsGroups()
	case SaveOutputNameScreen:
		return m.navigateToCommandOutput()
	case ContextsNamespacesMenuScreen:
		return m.navigateToMainMenu()
	case ContextsListScreen:
		return m.navigateToContextsAndNamespacesMenu()
	case NamespacesListScreen:
		return m.navigateToContextsAndNamespacesMenu()
	case PortInputScreen:
		return m.navigateToActionSelection()
	default:
		return m.navigateToMainMenu()
	}
}

// navigateToClusterInfo navigates to the cluster info screen and triggers data loading
func (m Model) navigateToClusterInfo() Model {
	m.previousScreen = m.currentScreen
	m.currentScreen = ClusterInfoScreen

	// Initialize viewport with loading message
	m.viewport = ui.NewViewport(m.width, m.height-6)
	m.viewport.SetContent("Loading cluster information...\n\nThis may take a few moments.")

	return m
}
