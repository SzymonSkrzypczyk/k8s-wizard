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

	case CommandPreviewScreen:
		s.WriteString("Command Preview\n")
		s.WriteString(strings.Repeat("─", m.width) + "\n")
		s.WriteString(fmt.Sprintf("Command: %s\n\n", m.currentCommand))
		s.WriteString(m.list.View())

	default:
		s.WriteString(m.list.View())
	}

	s.WriteString("\n\nPress 'q' to quit")

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
	case SaveFavouriteScreen, RenameFavouriteScreen:
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

	case CommandPreviewScreen:
		return m.handleCommandPreviewSelection()

	case FavouritesListScreen:
		return m.handleFavouriteSelection()

	case SaveFavouriteScreen:
		return m.handleSaveFavourite()

	case RenameFavouriteScreen:
		return m.handleRenameFavourite()
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

func (m Model) navigateToSaveFavourite() Model {
	m.textInput.SetValue("")
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
	case CommandPreviewScreen:
		if m.selectedAction == ActionGet {
			return m.navigateToActionSelection()
		}
		return m.navigateToResourceSelection()
	case FavouritesListScreen:
		return m.navigateToMainMenu()
	case SaveFavouriteScreen:
		return m.navigateToCommandPreview()
	case RenameFavouriteScreen:
		return m.navigateToFavouritesList()
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
		// For 'get' commands, we can go directly to preview
		m.currentCommand = buildCommand(m.selectedResource, m.selectedAction, "")
		return m.navigateToCommandPreview(), nil

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
	m.currentCommand = buildCommand(m.selectedResource, m.selectedAction, m.selectedResourceName)

	return m.navigateToCommandPreview(), nil
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

// Command execution

func (m Model) fetchPodNames() tea.Cmd {
	return func() tea.Msg {
		names, err := m.kubectlClient.ListPodNames()
		return podNamesLoadedMsg{names: names, err: err}
	}
}

func (m Model) executeCommand() tea.Cmd {
	return func() tea.Msg {
		var result kubectl.CommandResult
		var err error

		// Execute based on current selections
		switch m.selectedAction {
		case ActionGet:
			if m.selectedResource == ResourcePods {
				result, err = m.kubectlClient.GetPods()
			} else {
				result, err = m.kubectlClient.GetDeployments()
			}
		case ActionDescribe:
			result, err = m.kubectlClient.DescribePod(m.selectedResourceName)
		case ActionLogs:
			result, err = m.kubectlClient.GetPodLogs(m.selectedResourceName)
		}

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
