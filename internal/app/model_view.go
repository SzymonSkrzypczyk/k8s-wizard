package app

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// View renders the UI (required by Bubble Tea).
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

	case CustomCommandScreen:
		s.WriteString("Custom Command\n")
		s.WriteString(strings.Repeat("─", m.width) + "\n")
		s.WriteString("Enter kubectl arguments (without the leading 'kubectl') or a full kubectl command:\n\n")
		s.WriteString(m.textInput.View())
		s.WriteString("\n\nPress Enter to preview, Esc to cancel")

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
