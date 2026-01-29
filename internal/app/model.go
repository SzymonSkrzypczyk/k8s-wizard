package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/k8s-wizard/internal/favourites"
	"github.com/k8s-wizard/internal/kubectl"
	"github.com/k8s-wizard/internal/ui"
)

// Model represents the application state
type Model struct {
	// Core dependencies
	kubectlClient *kubectl.Client
	favStore      *favourites.Store

	// Current screen and navigation state
	currentScreen  Screen
	previousScreen Screen

	// User selections throughout the wizard
	selectedResource     ResourceType
	selectedAction       Action
	selectedResourceName string
	selectedFlags        []string // Selected command flags
	customNamespace      string   // Custom namespace value
	needsNamespaceInput  bool     // Whether namespace input is needed
	currentCommand       string
	renamingFavouriteIdx int // Index of favourite being renamed

	// UI components
	list      list.Model
	viewport  viewport.Model
	textInput textinput.Model

	// Terminal dimensions
	width  int
	height int

	// Error state
	err error
}

// NewModel creates and initializes a new application model
func NewModel() Model {
	// Initialize kubectl client
	kubectlClient := kubectl.NewClient()

	// Initialize favourites store
	favStore, err := favourites.NewStore()
	if err != nil {
		// If we can't load favourites, continue anyway
		// The error will be shown in the UI
		favStore = nil
	}

	// Create initial list for main menu
	mainMenuItems := []list.Item{
		ui.NewSimpleItem("Run Command", "Execute kubectl commands"),
		ui.NewSimpleItem("Favourites", "View and run saved commands"),
		ui.NewSimpleItem("Exit", "Quit the application"),
	}

	initialList := ui.NewList(mainMenuItems, "Kubernetes Wizard", 0, 0)

	// Create text input for naming favourites
	ti := textinput.New()
	ti.Placeholder = "Enter favourite name"
	ti.CharLimit = 50

	return Model{
		kubectlClient: kubectlClient,
		favStore:      favStore,
		currentScreen: MainMenuScreen,
		list:          initialList,
		textInput:     ti,
		viewport:      ui.NewViewport(0, 0),
		err:           err,
	}
}

// Init initializes the model (required by Bubble Tea)
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model (required by Bubble Tea)
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Update list dimensions
		m.list.SetSize(msg.Width, msg.Height-4)

		// Update viewport dimensions
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 4

		return m, nil

	case podNamesLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		// Create list of pod names
		items := ui.StringsToItems(msg.names)
		m.list = ui.NewList(items, "Select Pod", m.width, m.height-4)
		m.currentScreen = ResourceNameSelectionScreen
		return m, nil

	case commandExecutedMsg:
		// Display command output
		output := msg.result.Output
		if msg.result.Error != "" {
			output = "Error:\n" + msg.result.Error + "\n\nOutput:\n" + output
		}

		m.viewport.SetContent(output)
		m.currentScreen = CommandOutputScreen
		return m, nil

	case favouriteSavedMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		// Return to main menu after saving
		return m.navigateToMainMenu(), nil

	case favouriteDeletedMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		// Refresh favourites list
		return m.navigateToFavouritesList(), nil

	case favouriteRenamedMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		// Refresh favourites list
		return m.navigateToFavouritesList(), nil
	}

	return m, nil
}

// View renders the UI (required by Bubble Tea)
func (m Model) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	var s strings.Builder

	// Show error if present
	if m.err != nil {
		s.WriteString(fmt.Sprintf("⚠️  Error: %v\n\n", m.err))
	}

	// Render current screen
	switch m.currentScreen {
	case CommandOutputScreen:
		s.WriteString("Command Output\n")
		s.WriteString(strings.Repeat("─", m.width) + "\n")
		s.WriteString(m.viewport.View())
		s.WriteString("\n\nPress 'q' to return to main menu")

	case SaveFavouriteScreen:
		s.WriteString("Save as Favourite\n")
		s.WriteString(strings.Repeat("─", m.width) + "\n")
		s.WriteString(fmt.Sprintf("Command: %s\n\n", m.currentCommand))
		s.WriteString(m.textInput.View())
		s.WriteString("\n\nPress Enter to save, Esc to cancel")

	case RenameFavouriteScreen:
		s.WriteString("Rename Favourite\n")
		s.WriteString(strings.Repeat("─", m.width) + "\n")
		s.WriteString("Enter new name:\n\n")
		s.WriteString(m.textInput.View())
		s.WriteString("\n\nPress Enter to save, Esc to cancel")

	case NamespaceInputScreen:
		s.WriteString("Custom Namespace\n")
		s.WriteString(strings.Repeat("─", m.width) + "\n")
		s.WriteString("Enter namespace name:\n\n")
		s.WriteString(m.textInput.View())
		s.WriteString("\n\nPress Enter to continue, Esc to cancel")

	case CommandPreviewScreen:
		s.WriteString("Command Preview\n")
		s.WriteString(strings.Repeat("─", m.width) + "\n")
		s.WriteString(fmt.Sprintf("Command: %s\n\n", m.currentCommand))
		s.WriteString(m.list.View())

	default:
		s.WriteString(m.list.View())
	}

	// Add context-sensitive help text at the bottom
	if m.currentScreen == MainMenuScreen {
		s.WriteString("\n\nPress 'q' to quit")
	} else {
		s.WriteString("\n\nPress 'Esc' to go back | 'q' to quit")
	}

	return s.String()
}

// handleKeyPress processes keyboard input
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "ctrl+c", "q":
		if m.currentScreen == MainMenuScreen {
			return m, tea.Quit
		}
		// Return to main menu from other screens
		return m.navigateToMainMenu(), nil

	case "esc":
		// Go back to previous screen
		return m.navigateBack(), nil

	case "enter":
		return m.handleEnterKey()

	case " ":
		// Space bar toggles flags in flags selection screen
		if m.currentScreen == FlagsSelectionScreen {
			return m.toggleFlag(), nil
		}

	case "d":
		// Delete favourite if in favourites list
		if m.currentScreen == FavouritesListScreen && m.favStore != nil {
			idx := m.list.Index()
			if idx >= 0 && idx < len(m.favStore.List()) {
				return m, m.deleteFavourite(idx)
			}
		}

	case "r":
		// Rename favourite if in favourites list
		if m.currentScreen == FavouritesListScreen && m.favStore != nil {
			idx := m.list.Index()
			if idx >= 0 && idx < len(m.favStore.List()) {
				return m.navigateToRenameFavourite(idx), nil
			}
		}
	}

	// Pass other keys to the active component
	switch m.currentScreen {
	case SaveFavouriteScreen, RenameFavouriteScreen, NamespaceInputScreen:
		m.textInput, cmd = m.textInput.Update(msg)
	case CommandOutputScreen:
		m.viewport, cmd = ui.UpdateViewport(m.viewport, msg)
	default:
		m.list, cmd = ui.UpdateList(m.list, msg)
	}

	return m, cmd
}

// handleEnterKey processes the Enter key based on current screen
func (m Model) handleEnterKey() (tea.Model, tea.Cmd) {
	switch m.currentScreen {
	case MainMenuScreen:
		return m.handleMainMenuSelection()

	case ResourceSelectionScreen:
		return m.handleResourceSelection()

	case ActionSelectionScreen:
		return m.handleActionSelection()

	case ResourceNameSelectionScreen:
		return m.handleResourceNameSelection()

	case FlagsSelectionScreen:
		return m.handleFlagsSelection()

	case CommandPreviewScreen:
		return m.handleCommandPreviewSelection()

	case FavouritesListScreen:
		return m.handleFavouriteSelection()

	case SaveFavouriteScreen:
		return m.handleSaveFavourite()

	case RenameFavouriteScreen:
		return m.handleRenameFavourite()

	case NamespaceInputScreen:
		return m.handleNamespaceInput()
	}

	return m, nil
}

// Navigation handlers

func (m Model) navigateToMainMenu() Model {
	items := []list.Item{
		ui.NewSimpleItem("Run Command", "Execute kubectl commands"),
		ui.NewSimpleItem("Favourites", "View and run saved commands"),
		ui.NewSimpleItem("Exit", "Quit the application"),
	}
	m.list = ui.NewList(items, "Kubernetes Wizard", m.width, m.height-4)
	m.previousScreen = m.currentScreen
	m.currentScreen = MainMenuScreen
	m.err = nil
	return m
}

func (m Model) navigateToResourceSelection() Model {
	items := []list.Item{
		ui.NewSimpleItem("Pods", "Manage pods"),
		ui.NewSimpleItem("Deployments", "Manage deployments"),
	}
	m.list = ui.NewList(items, "Select Resource Type", m.width, m.height-4)
	m.previousScreen = m.currentScreen
	m.currentScreen = ResourceSelectionScreen
	return m
}

func (m Model) navigateToActionSelection() Model {
	var items []list.Item

	if m.selectedResource == ResourcePods {
		items = []list.Item{
			ui.NewSimpleItem("Get", "List all pods"),
			ui.NewSimpleItem("Describe", "Describe a specific pod"),
			ui.NewSimpleItem("Logs", "View logs from a pod"),
		}
	} else {
		items = []list.Item{
			ui.NewSimpleItem("Get", "List all deployments"),
		}
	}

	m.list = ui.NewList(items, "Select Action", m.width, m.height-4)
	m.previousScreen = m.currentScreen
	m.currentScreen = ActionSelectionScreen
	return m
}

func (m Model) navigateToCommandPreview() Model {
	items := []list.Item{
		ui.NewSimpleItem("Execute", "Run the command"),
		ui.NewSimpleItem("Save as Favourite", "Save for later use"),
		ui.NewSimpleItem("Back", "Return to previous screen"),
	}
	m.list = ui.NewList(items, "Command Preview", m.width, m.height-4)
	m.previousScreen = m.currentScreen
	m.currentScreen = CommandPreviewScreen
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
			ui.NewSimpleItem("No favourites saved", ""),
		}
	}

	m.list = ui.NewList(items, "Favourites (Enter=run, 'd'=delete, 'r'=rename)", m.width, m.height-4)
	m.previousScreen = m.currentScreen
	m.currentScreen = FavouritesListScreen
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
			ui.NewSimpleItem("[ ] -o wide", "Show additional columns"),
			ui.NewSimpleItem("[ ] -o yaml", "Output in YAML format"),
			ui.NewSimpleItem("[ ] -o json", "Output in JSON format"),
			ui.NewSimpleItem("[ ] --show-labels", "Show labels"),
			ui.NewSimpleItem("[ ] -A", "All namespaces"),
			ui.NewSimpleItem("[ ] -n <namespace>", "Specify custom namespace"),
			ui.NewSimpleItem("---", ""),
			ui.NewSimpleItem("Done (Continue)", "Proceed with selected flags"),
		}
	case ActionDescribe:
		items = []list.Item{
			ui.NewSimpleItem("[ ] --show-events=true", "Show events"),
			ui.NewSimpleItem("[ ] -n <namespace>", "Specify custom namespace"),
			ui.NewSimpleItem("---", ""),
			ui.NewSimpleItem("Done (Continue)", "Proceed with selected flags"),
		}
	case ActionLogs:
		items = []list.Item{
			ui.NewSimpleItem("[ ] -f", "Follow log output"),
			ui.NewSimpleItem("[ ] --tail=100", "Show last 100 lines"),
			ui.NewSimpleItem("[ ] --tail=50", "Show last 50 lines"),
			ui.NewSimpleItem("[ ] --since=1h", "Show logs from last hour"),
			ui.NewSimpleItem("[ ] --since=5m", "Show logs from last 5 minutes"),
			ui.NewSimpleItem("[ ] --previous", "Show logs from previous container"),
			ui.NewSimpleItem("[ ] -n <namespace>", "Specify custom namespace"),
			ui.NewSimpleItem("---", ""),
			ui.NewSimpleItem("Done (Continue)", "Proceed with selected flags"),
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
		if m.selectedAction == ActionGet {
			return m.navigateToActionSelection()
		}
		return m.navigateToResourceSelection()
	case CommandPreviewScreen:
		return m.navigateToFlagsSelection()
	case FavouritesListScreen:
		return m.navigateToMainMenu()
	case SaveFavouriteScreen:
		return m.navigateToCommandPreview()
	case RenameFavouriteScreen:
		return m.navigateToFavouritesList()
	case NamespaceInputScreen:
		return m.navigateToFlagsSelection()
	default:
		return m.navigateToMainMenu()
	}
}

// Selection handlers

func (m Model) handleMainMenuSelection() (tea.Model, tea.Cmd) {
	selected := m.list.SelectedItem()
	if selected == nil {
		return m, nil
	}

	title := selected.(ui.SimpleItem).Title()

	switch title {
	case "Run Command":
		return m.navigateToResourceSelection(), nil
	case "Favourites":
		return m.navigateToFavouritesList(), nil
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
		// Need to fetch pod names
		return m, m.fetchPodNames()

	case "Logs":
		m.selectedAction = ActionLogs
		// Need to fetch pod names
		return m, m.fetchPodNames()
	}

	return m, nil
}

func (m Model) handleResourceNameSelection() (tea.Model, tea.Cmd) {
	selected := m.list.SelectedItem()
	if selected == nil {
		return m, nil
	}

	m.selectedResourceName = selected.(ui.SimpleItem).Title()

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
		// Build command with selected flags
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

// toggleFlag toggles the selection state of the current flag
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
	case "Save as Favourite":
		return m.navigateToSaveFavourite(), nil
	case "Back":
		return m.navigateBack(), nil
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

// Command execution

func (m Model) fetchPodNames() tea.Cmd {
	return func() tea.Msg {
		names, err := m.kubectlClient.ListPodNames()
		return podNamesLoadedMsg{names: names, err: err}
	}
}

func (m Model) executeCommand() tea.Cmd {
	return func() tea.Msg {
		// Use the ExecuteRaw method which validates cluster context and runs the command
		result, err := m.kubectlClient.ExecuteRaw(m.currentCommand)
		return commandExecutedMsg{result: result, err: err}
	}
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
