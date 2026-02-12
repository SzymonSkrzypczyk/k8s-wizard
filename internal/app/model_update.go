package app

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/k8s-wizard/internal/hotkeys"
	"github.com/k8s-wizard/internal/ui"
)

// Init initializes the model (required by Bubble Tea).
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model (required by Bubble Tea).
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

	case resourceNamesLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		// Create list of resource names
		items := ui.StringsToItems(msg.names)
		title := fmt.Sprintf("Select %s", strings.TrimSuffix(m.selectedResource.String(), "s"))
		m.list = ui.NewList(items, title, m.width, m.height-4)
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
		// Preserve the full command output separately for saving, independent of viewport rendering
		m.currentOutputContent = output
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
		// Keep full help text available in case we later support saving it
		m.currentOutputContent = output
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
		// Track full connectivity output for consistency, even if we don't save it yet
		m.currentOutputContent = output
		m.currentScreen = ClusterConnectivityScreen
		return m, nil

	case contextSwitchedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = fmt.Errorf("✓ Switched context to %s", msg.newContext)
		return m.navigateToMainMenu(), nil

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

// handleKeyPress processes keyboard input.
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

// handleEnterKey processes the Enter key based on current screen.
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

	case ContextsNamespacesMenuScreen:
		return m.handleContextsAndNamespacesMenuSelection()

	case ContextsListScreen:
		return m.handleContextSelection()

	case NamespacesListScreen:
		return m.handleNamespaceSelection()

	case CustomCommandScreen:
		return m.handleCustomCommandInput()
	}

	return m, nil
}
