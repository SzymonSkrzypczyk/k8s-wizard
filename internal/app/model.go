package app

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/k8s-wizard/internal/favourites"
	"github.com/k8s-wizard/internal/history"
	"github.com/k8s-wizard/internal/hotkeys"
	"github.com/k8s-wizard/internal/kubectl"
	"github.com/k8s-wizard/internal/ui"
)

// Model represents the application state
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
}

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

func (m Model) loadCommandHelp() tea.Cmd {
	return func() tea.Msg {
		helpCmd := strings.TrimSpace(m.currentCommand)
		if helpCmd == "" {
			helpCmd = "kubectl"
		}
		if !strings.HasSuffix(helpCmd, " --help") {
			helpCmd = helpCmd + " --help"
		}
		result, err := m.kubectlClient.ExecuteRaw(helpCmd)
		return commandHelpLoadedMsg{result: result, err: err}
	}
}

func (m Model) checkClusterConnectivity() tea.Cmd {
	return func() tea.Msg {
		result, err := m.kubectlClient.ExecuteRaw("kubectl cluster-info")
		return clusterConnectivityCheckedMsg{result: result, err: err}
	}
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

func (m Model) renameSavedOutputGroup(oldBase string, newBase string) tea.Cmd {
	return func() tea.Msg {
		oldBase = strings.TrimSpace(strings.TrimSuffix(oldBase, ".txt"))
		newBase = strings.TrimSpace(strings.TrimSuffix(newBase, ".txt"))
		if oldBase == "" || newBase == "" {
			return savedOutputRenamedMsg{err: fmt.Errorf("invalid name")}
		}
		if oldBase == newBase {
			return savedOutputRenamedMsg{err: nil}
		}

		dir := "saved_cmd"
		entries, err := os.ReadDir(dir)
		if err != nil {
			return savedOutputRenamedMsg{err: err}
		}

		versionRe := regexp.MustCompile(`^(.*)_v(\d+)$`)
		var renames [][2]string
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".txt") {
				continue
			}
			name := strings.TrimSuffix(entry.Name(), ".txt")
			base := name
			suffix := ""
			if matches := versionRe.FindStringSubmatch(name); matches != nil {
				base = matches[1]
				suffix = "_v" + matches[2]
			}
			if base != oldBase {
				continue
			}
			oldPath := fmt.Sprintf("%s/%s.txt", dir, name)
			newPath := fmt.Sprintf("%s/%s%s.txt", dir, newBase, suffix)
			renames = append(renames, [2]string{oldPath, newPath})
		}
		if len(renames) == 0 {
			return savedOutputRenamedMsg{err: fmt.Errorf("saved output '%s' not found", oldBase)}
		}

		for _, rn := range renames {
			if _, err := os.Stat(rn[1]); err == nil {
				return savedOutputRenamedMsg{err: fmt.Errorf("saved output '%s' already exists", newBase)}
			}
		}

		for _, rn := range renames {
			if err := os.Rename(rn[0], rn[1]); err != nil {
				return savedOutputRenamedMsg{err: err}
			}
		}
		if err := m.updateSavedOutputsIndexOnRename(oldBase, newBase); err != nil {
			return savedOutputRenamedMsg{err: err}
		}
		return savedOutputRenamedMsg{err: nil}
	}
}

func (m Model) deleteSavedOutputGroup(base string) tea.Cmd {
	return func() tea.Msg {
		base = strings.TrimSpace(strings.TrimSuffix(base, ".txt"))
		if base == "" {
			return savedOutputsLoadedMsg{files: nil, err: fmt.Errorf("invalid name")}
		}
		dir := "saved_cmd"
		entries, err := os.ReadDir(dir)
		if err != nil {
			return savedOutputsLoadedMsg{files: nil, err: err}
		}
		versionRe := regexp.MustCompile(`^(.*)_v(\d+)$`)
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".txt") {
				continue
			}
			name := strings.TrimSuffix(entry.Name(), ".txt")
			fileBase := name
			if matches := versionRe.FindStringSubmatch(name); matches != nil {
				fileBase = matches[1]
			}
			if fileBase != base {
				continue
			}
			_ = os.Remove(fmt.Sprintf("%s/%s.txt", dir, name))
		}
		_ = m.removeSavedOutputsIndexForBase(base)
		return m.loadSavedOutputsCmd()()
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
		} else {
			output = "Output:\n" + output
		}

		m.viewport.SetContent(output)
		m.currentScreen = CommandOutputScreen
		return m, nil

	case commandHelpLoadedMsg:
		output := msg.result.Output
		if msg.result.Error != "" {
			output = "Error:\n" + msg.result.Error + "\n\nHelp Output:\n" + output
		} else {
			output = "Help Output:\n" + output
		}
		m.viewport.SetContent(output)
		m.currentScreen = CommandHelpScreen
		return m, nil

	case clusterConnectivityCheckedMsg:
		output := msg.result.Output
		if msg.result.Error != "" {
			output = "Error:\n" + msg.result.Error + "\n\nCluster Connectivity:\n" + output
		} else {
			if strings.Contains(output, "Unable to connect to the server") {
				output = "Cluster Connectivity:\n\n❌ Cannot connect to the Kubernetes cluster.\n\n" + output
			} else {
				// Show a concise connected status and include basic info
				lines := strings.Split(output, "\n")
				var summary []string
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line != "" && !strings.HasPrefix(line, "Further debugging") && !strings.HasPrefix(line, "To further debug") {
						summary = append(summary, line)
					}
				}
				output = "Cluster Connectivity:\n\n✅ Connected to the Kubernetes cluster.\n\n" + strings.Join(summary, "\n")
			}
		}
		m.viewport.SetContent(output)
		m.currentScreen = ClusterConnectivityScreen
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

	case savedOutputRenamedMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		return m.loadSavedOutputs()

	case outputSavedMsg:
		if msg.err != nil {
			m.err = fmt.Errorf("Failed to save output: %v", msg.err)
			return m, nil
		}
		// Show success message and return to main menu
		m.err = fmt.Errorf("✓ Output saved to: %s", msg.filename)
		return m.navigateToMainMenu(), nil

	case savedOutputsLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m.navigateToMainMenu(), nil
		}

		versionRe := regexp.MustCompile(`^(.*)_v(\d+)$`)
		grouped := make(map[string][]string)
		for _, f := range msg.files {
			base := f
			if matches := versionRe.FindStringSubmatch(f); matches != nil {
				if matches[1] != "" {
					base = matches[1]
				}
			}
			grouped[base] = append(grouped[base], f)
		}

		for base, versions := range grouped {
			sort.Slice(versions, func(i, j int) bool {
				vi := 1
				vj := 1
				if matches := versionRe.FindStringSubmatch(versions[i]); matches != nil {
					if v, err := strconv.Atoi(matches[2]); err == nil {
						vi = v
					}
				}
				if matches := versionRe.FindStringSubmatch(versions[j]); matches != nil {
					if v, err := strconv.Atoi(matches[2]); err == nil {
						vj = v
					}
				}
				return vi < vj
			})
			grouped[base] = versions
		}
		m.savedOutputsByBase = grouped

		if m.savedOutputsReturnScreen == SavedOutputVersionsScreen && m.savedOutputsReturnBase != "" {
			base := m.savedOutputsReturnBase
			idx := m.savedOutputsReturnVersionIdx
			m.savedOutputsReturnScreen = 0
			m.savedOutputsReturnBase = ""
			m.savedOutputsReturnVersionIdx = 0
			if _, ok := m.savedOutputsByBase[base]; ok {
				m.selectedSavedOutputVersionIdx = idx
				return m.navigateToSavedOutputVersions(base), nil
			}
		}
		m.savedOutputsReturnScreen = 0
		m.savedOutputsReturnBase = ""
		m.savedOutputsReturnVersionIdx = 0
		return m.navigateToSavedOutputsGroups(), nil
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
		s.WriteString(fmt.Sprintf("Command: %s\n\n", m.currentCommand))
		s.WriteString(m.viewport.View())
		s.WriteString("\n\nPress 's' to save output | 'q' to return to main menu | ↑↓ to scroll")

	case CommandHelpScreen:
		s.WriteString("Command Help\n")
		s.WriteString(strings.Repeat("─", m.width) + "\n")
		s.WriteString(fmt.Sprintf("Command: %s --help\n\n", m.currentCommand))
		s.WriteString(m.viewport.View())
		s.WriteString("\n\nPress 'Esc' to go back | ↑↓ to scroll")

	case HotkeyBindScreen:
		s.WriteString("Bind Hotkey\n")
		s.WriteString(strings.Repeat("─", m.width) + "\n")
		s.WriteString("Press F1-F12 to bind the selected favourite\n\n")
		s.WriteString(fmt.Sprintf("Favourite: %s\n", m.hotkeyBindingFavourite.Name))
		s.WriteString(fmt.Sprintf("Command: %s\n\n", m.hotkeyBindingFavourite.Command))
		s.WriteString("Press Esc to cancel")

	case HotkeysListScreen:
		s.WriteString(m.list.View())

	case ClusterConnectivityScreen:
		s.WriteString("Cluster Connectivity\n")
		s.WriteString(strings.Repeat("─", m.width) + "\n")
		s.WriteString(m.viewport.View())
		s.WriteString("\n\nPress 'Esc' to go back | ↑↓ to scroll")

	case CommandHistoryScreen:
		s.WriteString(m.list.View())

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

	case RenameSavedOutputScreen:
		s.WriteString("Rename Saved Output\n")
		s.WriteString(strings.Repeat("─", m.width) + "\n")
		s.WriteString("Enter new name (without extension):\n\n")
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

	case SavedOutputViewScreen:
		s.WriteString("Saved Output: " + m.selectedSavedOutput + "\n")
		s.WriteString(strings.Repeat("─", m.width) + "\n")
		s.WriteString(m.viewport.View())
		s.WriteString("\n\nPress 'd' to delete | 'q' or 'Esc' to go back | ↑↓ to scroll")

	case SaveOutputNameScreen:
		s.WriteString("Save Output\n")
		s.WriteString(strings.Repeat("─", m.width) + "\n")
		s.WriteString("Enter name for saved output (without extension):\n\n")
		s.WriteString(m.textInput.View())
		s.WriteString("\n\nPress Enter to save, Esc to cancel")

	case SavedOutputsListScreen:
		s.WriteString(m.list.View())

	case SavedOutputVersionsScreen:
		s.WriteString(m.renderSavedOutputVersionsTable())

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

	// Global hotkeys (F1-F12) – ignore while typing into a text input screen
	if m.hotkeyStore != nil && !m.isTextInputScreen() {
		if hk, ok := m.tryParseHotkey(msg.String()); ok {
			// If we're currently binding a hotkey, bind instead of executing
			if m.hotkeyBindingPending {
				binding := hotkeys.Binding{Key: hk, Name: m.hotkeyBindingFavourite.Name, Command: m.hotkeyBindingFavourite.Command}
				if err := m.hotkeyStore.Set(binding); err != nil {
					m.err = err
					m.hotkeyBindingPending = false
					return m.navigateToFavouritesList(), nil
				}
				m.err = fmt.Errorf("✓ Bound %s to %s", hk, m.hotkeyBindingFavourite.Name)
				m.hotkeyBindingPending = false
				return m.navigateToFavouritesList(), nil
			}

			if binding, ok := m.hotkeyStore.Get(hk); ok {
				m.currentCommand = binding.Command
				return m, m.executeCommand()
			}
		}
	}

	switch msg.String() {
	case "ctrl+c", "q":
		if m.currentScreen == MainMenuScreen {
			return m, tea.Quit
		}
		// Return to main menu from other screens
		return m.navigateToMainMenu(), nil

	case "esc":
		if m.currentScreen == HotkeyBindScreen {
			m.hotkeyBindingPending = false
			return m.navigateToFavouritesList(), nil
		}
		if m.currentScreen == RenameSavedOutputScreen {
			if m.renamingSavedOutputIsGroup {
				return m.loadSavedOutputsToVersions(m.renamingSavedOutput)
			}
			return m.loadSavedOutputs()
		}
		// Go back to previous screen
		return m.navigateBack(), nil

	case "enter":
		return m.handleEnterKey()

	case " ":
		// Space bar toggles flags in flags selection screen
		if m.currentScreen == FlagsSelectionScreen {
			return m.toggleFlag(), nil
		}

	case "left":
		if m.currentScreen == SavedOutputVersionsScreen {
			versions := m.savedOutputsByBase[m.selectedSavedOutputBase]
			if len(versions) == 0 {
				return m, nil
			}
			if m.selectedSavedOutputVersionIdx <= 0 {
				m.selectedSavedOutputVersionIdx = len(versions) - 1
			} else {
				m.selectedSavedOutputVersionIdx--
			}
			return m, nil
		}

	case "right":
		if m.currentScreen == SavedOutputVersionsScreen {
			versions := m.savedOutputsByBase[m.selectedSavedOutputBase]
			if len(versions) == 0 {
				return m, nil
			}
			if m.selectedSavedOutputVersionIdx >= len(versions)-1 {
				m.selectedSavedOutputVersionIdx = 0
			} else {
				m.selectedSavedOutputVersionIdx++
			}
			return m, nil
		}

	case "d":
		// Delete favourite if in favourites list
		if m.currentScreen == FavouritesListScreen && m.favStore != nil {
			idx := m.list.Index()
			if idx >= 0 && idx < len(m.favStore.List()) {
				return m, m.deleteFavourite(idx)
			}
		}
		// Delete hotkey binding if in hotkeys list
		if m.currentScreen == HotkeysListScreen && m.hotkeyStore != nil {
			selected := m.list.SelectedItem()
			if selected != nil {
				key := selected.(ui.SimpleItem).Title()
				if strings.HasPrefix(strings.ToUpper(key), "F") {
					if err := m.hotkeyStore.Delete(key); err != nil {
						m.err = err
						return m, nil
					}
					m.err = fmt.Errorf("✓ Unbound %s", strings.ToUpper(key))
					return m.navigateToHotkeysList(), nil
				}
			}
		}
		// Delete currently viewed saved output version
		if m.currentScreen == SavedOutputViewScreen {
			if strings.TrimSpace(m.selectedSavedOutput) != "" {
				base := m.selectedSavedOutputBase
				if base == "" {
					versionRe := regexp.MustCompile(`^(.*)_v(\d+)$`)
					base = m.selectedSavedOutput
					if matches := versionRe.FindStringSubmatch(base); matches != nil {
						if matches[1] != "" {
							base = matches[1]
						}
					}
				}
				m.savedOutputsReturnScreen = SavedOutputVersionsScreen
				m.savedOutputsReturnBase = base
				return m, m.deleteSavedOutput(m.selectedSavedOutput)
			}
		}
		// Delete saved output group if in saved outputs list
		if m.currentScreen == SavedOutputsListScreen {
			selected := m.list.SelectedItem()
			if selected != nil {
				base := selected.(ui.SimpleItem).Title()
				if base != "No saved outputs" {
					return m, m.deleteSavedOutputGroup(base)
				}
			}
		}
		if m.currentScreen == SavedOutputVersionsScreen {
			versions := m.savedOutputsByBase[m.selectedSavedOutputBase]
			if len(versions) == 0 {
				return m, nil
			}
			idx := m.selectedSavedOutputVersionIdx
			if idx < 0 {
				idx = 0
			}
			if idx >= len(versions) {
				idx = len(versions) - 1
			}
			filename := versions[idx]
			m.savedOutputsReturnScreen = SavedOutputVersionsScreen
			m.savedOutputsReturnBase = m.selectedSavedOutputBase
			m.savedOutputsReturnVersionIdx = idx
			return m, m.deleteSavedOutput(filename)
		}

	case "s":
		// Save output if in command output screen
		if m.currentScreen == CommandOutputScreen {
			baseName, ok, err := m.getSavedOutputBaseNameForCommand(m.currentCommand)
			if err != nil {
				m.err = err
				return m, nil
			}
			if ok {
				m.currentOutputContent = m.viewport.View()
				return m, m.saveOutput(baseName)
			}
			return m.navigateToSaveOutputName(), nil
		}
		// Save favourite from history list
		if m.currentScreen == CommandHistoryScreen && m.historyStore != nil && m.favStore != nil {
			idx := m.list.Index()
			entry, ok := m.historyStore.Get(idx)
			if ok {
				m.currentCommand = entry.Command
				return m.navigateToSaveFavourite(), nil
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
		if m.currentScreen == SavedOutputsListScreen {
			selected := m.list.SelectedItem()
			if selected != nil {
				base := selected.(ui.SimpleItem).Title()
				if base != "No saved outputs" && base != "Loading..." {
					return m.navigateToRenameSavedOutputGroup(base), nil
				}
			}
		}
		if m.currentScreen == SavedOutputVersionsScreen {
			if m.selectedSavedOutputBase != "" {
				return m.navigateToRenameSavedOutputGroup(m.selectedSavedOutputBase), nil
			}
		}

	case "h":
		// Start hotkey bind flow from favourites list
		if m.currentScreen == FavouritesListScreen && m.favStore != nil && m.hotkeyStore != nil {
			idx := m.list.Index()
			fav, ok := m.favStore.Get(idx)
			if ok {
				m.hotkeyBindingFavourite = fav
				m.hotkeyBindingPending = true
				m.previousScreen = m.currentScreen
				m.currentScreen = HotkeyBindScreen
				return m, nil
			}
		}
	}

	// Pass other keys to the active component
	switch m.currentScreen {
	case SaveFavouriteScreen, RenameFavouriteScreen, RenameSavedOutputScreen, NamespaceInputScreen, SaveOutputNameScreen:
		m.textInput, cmd = m.textInput.Update(msg)
	case CommandOutputScreen, SavedOutputViewScreen:
		m.viewport, cmd = ui.UpdateViewport(m.viewport, msg)
	case CommandHelpScreen:
		m.viewport, cmd = ui.UpdateViewport(m.viewport, msg)
	case ClusterConnectivityScreen:
		m.viewport, cmd = ui.UpdateViewport(m.viewport, msg)
	case SavedOutputVersionsScreen:
		cmd = nil
	case HotkeyBindScreen:
		cmd = nil
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

	case RenameSavedOutputScreen:
		return m.handleRenameSavedOutput()

	case NamespaceInputScreen:
		return m.handleNamespaceInput()

	case SavedOutputsListScreen:
		return m.handleSavedOutputSelection()

	case SavedOutputVersionsScreen:
		return m.handleSavedOutputVersionSelection()

	case SaveOutputNameScreen:
		return m.handleSaveOutputName()

	case CommandHistoryScreen:
		return m.handleCommandHistorySelection()
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

// Navigation handlers

func (m Model) navigateToMainMenu() Model {
	items := []list.Item{
		ui.NewSimpleItem("Run Command", "Execute kubectl commands"),
		ui.NewSimpleItem("Favourites", "View and run saved commands"),
		ui.NewSimpleItem("Command History", "View and re-run previous commands"),
		ui.NewSimpleItem("Saved Outputs", "View previously saved outputs"),
		ui.NewSimpleItem("Hotkeys", "Manage hotkey bindings"),
		ui.NewSimpleItem("Check Cluster Connectivity", "Verify connection to Kubernetes cluster"),
		ui.NewSimpleItem("Exit", "Quit the application"),
	}
	m.list = ui.NewList(items, "Kubernetes Wizard", m.width, m.height-4)
	m.previousScreen = m.currentScreen
	m.currentScreen = MainMenuScreen
	m.err = nil
	return m
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

func (m Model) navigateToCommandHistory() Model {
	items := []list.Item{}
	if m.historyStore == nil {
		items = []list.Item{ui.NewSimpleItem("History unavailable", "")}
		m.list = ui.NewList(items, "Command History", m.width, m.height-4)
		m.previousScreen = m.currentScreen
		m.currentScreen = CommandHistoryScreen
		return m
	}

	entries := m.historyStore.List()
	if len(entries) == 0 {
		items = []list.Item{ui.NewSimpleItem("No command history", "")}
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
		ui.NewSimpleItem("Help", "Show --help output"),
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

func (m Model) navigateToRenameSavedOutput(filename string) Model {
	m.renamingSavedOutput = filename
	m.renamingSavedOutputIsGroup = false
	m.textInput.SetValue(filename)
	m.textInput.Placeholder = "Enter new name"
	m.textInput.Focus()
	m.previousScreen = m.currentScreen
	m.currentScreen = RenameSavedOutputScreen
	return m
}

func (m Model) navigateToRenameSavedOutputGroup(base string) Model {
	m.renamingSavedOutput = base
	m.renamingSavedOutputIsGroup = true
	m.textInput.SetValue(base)
	m.textInput.Placeholder = "Enter new name"
	m.textInput.Focus()
	m.previousScreen = m.currentScreen
	m.currentScreen = RenameSavedOutputScreen
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
	case CommandHelpScreen:
		return m.navigateToCommandPreview()
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
	case "Command History":
		return m.navigateToCommandHistory(), nil
	case "Saved Outputs":
		return m.loadSavedOutputs()
	case "Hotkeys":
		return m.navigateToHotkeysList(), nil
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
	case "Help":
		return m, m.loadCommandHelp()
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

// Command execution

func (m Model) fetchPodNames() tea.Cmd {
	return func() tea.Msg {
		names, err := m.kubectlClient.ListPodNames()
		return podNamesLoadedMsg{names: names, err: err}
	}
}

func (m Model) executeCommand() tea.Cmd {
	return func() tea.Msg {
		// Add to history
		if m.historyStore != nil && strings.TrimSpace(m.currentCommand) != "" {
			_ = m.historyStore.Add(m.currentCommand)
		}
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

func (m Model) renameSavedOutput(oldName string, newName string) tea.Cmd {
	return func() tea.Msg {
		oldPath := fmt.Sprintf("saved_cmd/%s.txt", oldName)
		newPath := fmt.Sprintf("saved_cmd/%s.txt", newName)

		if oldName == newName {
			return savedOutputRenamedMsg{err: nil}
		}
		if _, err := os.Stat(newPath); err == nil {
			return savedOutputRenamedMsg{err: fmt.Errorf("saved output '%s' already exists", newName)}
		}
		if err := os.Rename(oldPath, newPath); err != nil {
			return savedOutputRenamedMsg{err: err}
		}
		if err := m.updateSavedOutputsIndexOnRename(oldName, newName); err != nil {
			return savedOutputRenamedMsg{err: err}
		}
		return savedOutputRenamedMsg{err: nil}
	}
}

func (m Model) loadSavedOutputsIndex() (map[string]string, error) {
	indexPath := "saved_cmd/index.json"

	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}

	if len(data) == 0 {
		return map[string]string{}, nil
	}

	var index map[string]string
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, err
	}
	if index == nil {
		index = map[string]string{}
	}
	return index, nil
}

func (m Model) saveSavedOutputsIndex(index map[string]string) error {
	data, err := json.Marshal(index)
	if err != nil {
		return err
	}

	if _, statErr := os.Stat("saved_cmd"); os.IsNotExist(statErr) {
		if err := os.Mkdir("saved_cmd", 0755); err != nil {
			return err
		}
	}

	return os.WriteFile("saved_cmd/index.json", data, 0644)
}

func (m Model) removeSavedOutputsIndexForBase(baseName string) error {
	baseName = strings.TrimSpace(strings.TrimSuffix(baseName, ".txt"))
	if baseName == "" {
		return nil
	}
	index, err := m.loadSavedOutputsIndex()
	if err != nil {
		return err
	}
	changed := false
	for cmd, base := range index {
		if base == baseName {
			delete(index, cmd)
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return m.saveSavedOutputsIndex(index)
}

func (m Model) savedOutputGroupExists(baseName string) (bool, error) {
	baseName = strings.TrimSpace(strings.TrimSuffix(baseName, ".txt"))
	if baseName == "" {
		return false, nil
	}

	dir := "saved_cmd"
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	versionRe := regexp.MustCompile(`^(.*)_v(\d+)$`)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".txt") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".txt")
		fileBase := name
		if matches := versionRe.FindStringSubmatch(name); matches != nil {
			fileBase = matches[1]
		}
		if fileBase == baseName {
			return true, nil
		}
	}

	return false, nil
}

func (m Model) getSavedOutputBaseNameForCommand(command string) (string, bool, error) {
	if strings.TrimSpace(command) == "" {
		return "", false, nil
	}

	index, err := m.loadSavedOutputsIndex()
	if err != nil {
		return "", false, err
	}
	name, ok := index[command]
	if !ok {
		return "", false, nil
	}
	name = strings.TrimSpace(strings.TrimSuffix(name, ".txt"))
	if name == "" {
		return "", false, nil
	}

	exists, err := m.savedOutputGroupExists(name)
	if err != nil {
		return "", false, err
	}
	if !exists {
		_ = m.removeSavedOutputsIndexForBase(name)
		return "", false, nil
	}
	return name, true, nil
}

func (m Model) setSavedOutputBaseNameForCommand(command string, baseName string) error {
	command = strings.TrimSpace(command)
	if command == "" {
		return nil
	}

	baseName = strings.TrimSpace(strings.TrimSuffix(baseName, ".txt"))
	if baseName == "" {
		return nil
	}

	index, err := m.loadSavedOutputsIndex()
	if err != nil {
		return err
	}
	index[command] = baseName
	return m.saveSavedOutputsIndex(index)
}

func (m Model) updateSavedOutputsIndexOnRename(oldName string, newName string) error {
	oldName = strings.TrimSpace(strings.TrimSuffix(oldName, ".txt"))
	newName = strings.TrimSpace(strings.TrimSuffix(newName, ".txt"))
	if oldName == "" || newName == "" {
		return nil
	}

	versionRe := regexp.MustCompile(`^(.*)_v(\d+)$`)
	if matches := versionRe.FindStringSubmatch(newName); matches != nil {
		if matches[1] != "" {
			newName = matches[1]
		}
	}

	index, err := m.loadSavedOutputsIndex()
	if err != nil {
		return err
	}

	changed := false
	for cmd, base := range index {
		if base == oldName {
			index[cmd] = newName
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return m.saveSavedOutputsIndex(index)
}

func (m Model) navigateToSaveOutputName() Model {
	m.currentOutputContent = m.viewport.View()
	m.textInput.SetValue("")
	m.textInput.Placeholder = "Enter name (e.g. pods-output)"
	m.textInput.Focus()
	m.previousScreen = m.currentScreen
	m.currentScreen = SaveOutputNameScreen
	return m
}

func (m Model) navigateToSavedOutputsList() Model {
	m.list = ui.NewList([]list.Item{
		ui.NewSimpleItem("Loading...", ""),
	}, "Saved Outputs", m.width, m.height-4)
	m.previousScreen = m.currentScreen
	m.currentScreen = SavedOutputsListScreen
	return m
}

func (m Model) navigateToSavedOutputsGroups() Model {
	items := []list.Item{}
	if len(m.savedOutputsByBase) == 0 {
		items = []list.Item{ui.NewSimpleItem("No saved outputs", "")}
	} else {
		bases := make([]string, 0, len(m.savedOutputsByBase))
		for base := range m.savedOutputsByBase {
			bases = append(bases, base)
		}
		sort.Strings(bases)
		for _, base := range bases {
			items = append(items, ui.NewSimpleItem(base, fmt.Sprintf("%d versions", len(m.savedOutputsByBase[base]))))
		}
	}
	m.list = ui.NewList(items, "Saved Outputs (Enter=versions, 'd'=delete, 'r'=rename)", m.width, m.height-4)
	m.previousScreen = m.currentScreen
	m.currentScreen = SavedOutputsListScreen
	return m
}

func (m Model) navigateToSavedOutputVersions(base string) Model {
	m.selectedSavedOutputBase = base
	versions := m.savedOutputsByBase[base]
	if m.selectedSavedOutputVersionIdx < 0 {
		m.selectedSavedOutputVersionIdx = 0
	}
	if len(versions) > 0 {
		if m.selectedSavedOutputVersionIdx >= len(versions) {
			m.selectedSavedOutputVersionIdx = len(versions) - 1
		}
	} else {
		m.selectedSavedOutputVersionIdx = 0
	}
	items := []list.Item{}
	if len(versions) == 0 {
		items = []list.Item{ui.NewSimpleItem("No versions", "")}
	} else {
		versionRe := regexp.MustCompile(`^(.*)_v(\d+)$`)
		for _, v := range versions {
			n := 1
			if matches := versionRe.FindStringSubmatch(v); matches != nil {
				if parsed, err := strconv.Atoi(matches[2]); err == nil {
					n = parsed
				}
			}
			items = append(items, ui.NewSimpleItem(v, fmt.Sprintf("v%d", n)))
		}
	}
	m.list = ui.NewList(items, fmt.Sprintf("Saved Outputs: %s (Enter=view, 'd'=delete)", base), m.width, m.height-4)
	m.previousScreen = m.currentScreen
	m.currentScreen = SavedOutputVersionsScreen
	return m
}

func (m Model) navigateToSavedOutputView(filename string, content string) Model {
	m.selectedSavedOutput = filename
	m.viewport.SetContent(content)
	m.previousScreen = m.currentScreen
	m.currentScreen = SavedOutputViewScreen
	return m
}

func (m Model) navigateToCommandOutput() Model {
	m.currentScreen = CommandOutputScreen
	return m
}

func (m Model) handleSaveOutputName() (tea.Model, tea.Cmd) {
	name := m.textInput.Value()
	if name == "" {
		return m, nil
	}

	return m, m.saveOutput(name)
}

func (m Model) handleSavedOutputSelection() (tea.Model, tea.Cmd) {
	selected := m.list.SelectedItem()
	if selected == nil {
		return m, nil
	}

	title := selected.(ui.SimpleItem).Title()
	if title == "No saved outputs" || title == "Loading..." {
		return m, nil
	}
	return m.navigateToSavedOutputVersions(title), nil
}

func (m Model) handleSavedOutputVersionSelection() (tea.Model, tea.Cmd) {
	versions := m.savedOutputsByBase[m.selectedSavedOutputBase]
	if len(versions) == 0 {
		return m, nil
	}
	idx := m.selectedSavedOutputVersionIdx
	if idx < 0 {
		idx = 0
	}
	if idx >= len(versions) {
		idx = len(versions) - 1
	}
	return m.viewSavedOutput(versions[idx])
}

func (m Model) renderSavedOutputVersionsTable() string {
	versions := m.savedOutputsByBase[m.selectedSavedOutputBase]
	if len(versions) == 0 {
		return "No versions"
	}

	idx := m.selectedSavedOutputVersionIdx
	if idx < 0 {
		idx = 0
	}
	if idx >= len(versions) {
		idx = len(versions) - 1
	}

	versionRe := regexp.MustCompile(`^(.*)_v(\d+)$`)
	labels := make([]string, 0, len(versions))
	for _, v := range versions {
		n := 1
		if matches := versionRe.FindStringSubmatch(v); matches != nil {
			if parsed, err := strconv.Atoi(matches[2]); err == nil {
				n = parsed
			}
		}
		labels = append(labels, fmt.Sprintf("v%d", n))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Saved Outputs: %s\n", m.selectedSavedOutputBase))
	sb.WriteString(strings.Repeat("─", m.width) + "\n")

	for i, lbl := range labels {
		cell := lbl
		if i == idx {
			cell = "[" + cell + "]"
		}
		if i > 0 {
			sb.WriteString("  ")
		}
		sb.WriteString(cell)
	}

	sb.WriteString("\n\n")
	sb.WriteString("←→ select | Enter view | d delete | r rename | Esc back")
	sb.WriteString("\n")
	sb.WriteString("Selected: " + versions[idx])
	return sb.String()
}

func (m Model) loadSavedOutputs() (tea.Model, tea.Cmd) {
	m = m.navigateToSavedOutputsList()
	return m, m.loadSavedOutputsCmd()
}

func (m Model) loadSavedOutputsToVersions(base string) (tea.Model, tea.Cmd) {
	m.savedOutputsReturnScreen = SavedOutputVersionsScreen
	m.savedOutputsReturnBase = base
	m = m.navigateToSavedOutputsList()
	return m, m.loadSavedOutputsCmd()
}

func (m Model) loadSavedOutputsCmd() tea.Cmd {
	return func() tea.Msg {
		dir := "saved_cmd"
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.Mkdir(dir, 0755); err != nil {
				return savedOutputsLoadedMsg{files: nil, err: err}
			}
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			return savedOutputsLoadedMsg{files: nil, err: err}
		}

		var files []string
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".txt") {
				// Strip .txt suffix for display
				name := strings.TrimSuffix(entry.Name(), ".txt")
				files = append(files, name)
			}
		}

		return savedOutputsLoadedMsg{files: files, err: nil}
	}
}

func (m Model) saveOutput(name string) tea.Cmd {
	return func() tea.Msg {
		content := m.currentOutputContent
		dir := "saved_cmd"

		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.Mkdir(dir, 0755); err != nil {
				return outputSavedMsg{filename: "", err: err}
			}
		}

		trimmedName := strings.TrimSpace(strings.TrimSuffix(name, ".txt"))
		if trimmedName == "" {
			return outputSavedMsg{filename: "", err: fmt.Errorf("output name cannot be empty")}
		}

		versionRe := regexp.MustCompile(`^(.*)_v(\d+)$`)
		baseName := trimmedName
		if matches := versionRe.FindStringSubmatch(trimmedName); matches != nil {
			if matches[1] != "" {
				baseName = matches[1]
			}
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			return outputSavedMsg{filename: "", err: err}
		}

		maxVersion := 0
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".txt") {
				continue
			}

			existing := strings.TrimSuffix(entry.Name(), ".txt")
			if existing == baseName {
				if maxVersion < 1 {
					maxVersion = 1
				}
				continue
			}
			if matches := versionRe.FindStringSubmatch(existing); matches != nil {
				if matches[1] != baseName {
					continue
				}
				v, convErr := strconv.Atoi(matches[2])
				if convErr != nil {
					continue
				}
				if v > maxVersion {
					maxVersion = v
				}
			}
		}

		targetName := baseName
		if maxVersion > 0 {
			targetName = fmt.Sprintf("%s_v%d", baseName, maxVersion+1)
		}

		filename := fmt.Sprintf("%s.txt", targetName)
		filepath := fmt.Sprintf("%s/%s", dir, filename)

		err = os.WriteFile(filepath, []byte(content), 0644)
		if err != nil {
			return outputSavedMsg{filename: "", err: err}
		}

		if err := m.setSavedOutputBaseNameForCommand(m.currentCommand, baseName); err != nil {
			return outputSavedMsg{filename: "", err: err}
		}

		return outputSavedMsg{filename: filename, err: nil}
	}
}

func (m Model) deleteSavedOutput(filename string) tea.Cmd {
	return func() tea.Msg {
		filepath := fmt.Sprintf("saved_cmd/%s.txt", filename)
		err := os.Remove(filepath)
		if err != nil {
			return savedOutputsLoadedMsg{files: nil, err: err}
		}

		versionRe := regexp.MustCompile(`^(.*)_v(\d+)$`)
		base := strings.TrimSpace(strings.TrimSuffix(filename, ".txt"))
		if matches := versionRe.FindStringSubmatch(base); matches != nil {
			if matches[1] != "" {
				base = matches[1]
			}
		}
		if base != "" {
			exists, checkErr := m.savedOutputGroupExists(base)
			if checkErr != nil {
				return savedOutputsLoadedMsg{files: nil, err: checkErr}
			}
			if !exists {
				_ = m.removeSavedOutputsIndexForBase(base)
			}
		}

		return m.loadSavedOutputsCmd()()
	}
}

func (m Model) viewSavedOutput(filename string) (tea.Model, tea.Cmd) {
	filePath := fmt.Sprintf("saved_cmd/%s.txt", filename)
	content, err := os.ReadFile(filePath)
	if err != nil {
		m.err = err
		return m, nil
	}

	m = m.navigateToSavedOutputView(filename, string(content))
	return m, nil
}
