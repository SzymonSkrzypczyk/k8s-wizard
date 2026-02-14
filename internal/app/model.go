package app

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/SzymonSkrzypczyk/k8s-wizard/internal/favourites"
	"github.com/SzymonSkrzypczyk/k8s-wizard/internal/history"
	"github.com/SzymonSkrzypczyk/k8s-wizard/internal/hotkeys"
	"github.com/SzymonSkrzypczyk/k8s-wizard/internal/kubectl"
	"github.com/SzymonSkrzypczyk/k8s-wizard/internal/ui"
)

// Model represents the application state.
type Model struct {
	// Core dependencies
	kubectlClient *kubectl.Client
	favStore      *favourites.Store
	hotkeyStore   *hotkeys.Store
	historyStore  *history.Store

	// Current screen and navigation state
	currentScreen  Screen
	previousScreen Screen

	// User selections throughout the wizard
	selectedResource              ResourceType
	selectedAction                Action
	selectedResourceName          string
	selectedFlags                 []string // Selected command flags
	customNamespace               string   // Custom namespace value
	needsNamespaceInput           bool     // Whether namespace input is needed
	currentCommand                string
	renamingFavouriteIdx          int    // Index of favourite being renamed
	currentOutputContent          string // Current output content to be saved
	selectedSavedOutput           string // Selected saved output filename
	renamingSavedOutput           string // Saved output being renamed
	renamingSavedOutputIsGroup    bool
	selectedSavedOutputBase       string
	selectedSavedOutputVersionIdx int
	savedOutputsByBase            map[string][]string
	savedOutputsReturnScreen      Screen
	savedOutputsReturnBase        string
	savedOutputsReturnVersionIdx  int

	hotkeyBindingPending   bool
	hotkeyBindingFavourite favourites.Favourite

	// UI components
	list      list.Model
	viewport  viewport.Model
	textInput textinput.Model

	// Terminal dimensions
	width  int
	height int

	// Error state
	err error

	// Default namespace applied to commands when no explicit namespace flag is chosen
	defaultNamespace string

	// Ready indicates if the TUI is initialized with terminal dimensions
	ready bool
}

// NewModel creates and initializes a new application model.
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

	// Initialize hotkey store
	hotkeyStore, hotkeyErr := hotkeys.NewStore()
	if hotkeyErr != nil {
		hotkeyStore = nil
		if err == nil {
			err = hotkeyErr
		}
	}

	// Initialize history store
	historyStore, historyErr := history.NewStore()
	if historyErr != nil {
		historyStore = nil
		if err == nil {
			err = historyErr
		}
	}

	// Create initial list for main menu
	mainMenuItems := []list.Item{
		ui.NewSimpleItem("Run Command", "Execute kubectl commands"),
		ui.NewSimpleItem("Favourites", "View and run saved commands"),
		ui.NewSimpleItem("Command History", "View and re-run previous commands"),
		ui.NewSimpleItem("Saved Outputs", "View previously saved outputs"),
		ui.NewSimpleItem("Hotkeys", "Manage hotkey bindings"),
		ui.NewSimpleItem("Check Cluster Connectivity", "Verify connection to Kubernetes cluster"),
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
		hotkeyStore:   hotkeyStore,
		historyStore:  historyStore,
		currentScreen: MainMenuScreen,
		list:          initialList,
		textInput:     ti,
		viewport:      ui.NewViewport(0, 0),
		err:           err,
	}
}
// GetKubectlClient returns the internal kubectl client.
func (m Model) GetKubectlClient() *kubectl.Client {
	return m.kubectlClient
}
