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
	tea "github.com/charmbracelet/bubbletea"
	"github.com/SzymonSkrzypczyk/k8s-wizard/internal/ui"
)

// Saved outputs navigation, storage, and helpers.

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

func (m Model) navigateToSaveOutputName() Model {
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
		items = []list.Item{
			ui.NewSimpleItem("No saved outputs", "Save command output from the Command Output screen to see it here"),
		}
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
	// When viewing a saved output, keep its full content in sync as well
	m.currentOutputContent = content
	m.previousScreen = m.currentScreen
	m.currentScreen = SavedOutputViewScreen
	return m
}

func (m Model) navigateToCommandOutput() Model {
	m.currentScreen = CommandOutputScreen
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
