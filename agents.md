# Development Guidelines for Kubernetes Wizard

This document provides comprehensive guidelines for AI agents and developers working on the Kubernetes Wizard project. It covers best practices for Go development, Bubble Tea TUI patterns, and project-specific conventions.

## Table of Contents

1. [Project Overview](#project-overview)
2. [Go Best Practices](#go-best-practices)
3. [Bubble Tea Architecture](#bubble-tea-architecture)
4. [Project Structure Guidelines](#project-structure-guidelines)
5. [Code Style and Conventions](#code-style-and-conventions)
6. [Testing Guidelines](#testing-guidelines)
7. [Common Patterns](#common-patterns)
8. [Troubleshooting](#troubleshooting)

---

## Project Overview

### Technology Stack
- **Language**: Go 1.21+
- **TUI Framework**: [Bubble Tea](https://github.com/charmbracelet/bubbletea) (Elm architecture)
- **UI Components**: [Bubbles](https://github.com/charmbracelet/bubbles) (list, viewport, textinput)
- **Styling**: [Lipgloss](https://github.com/charmbracelet/lipgloss)
- **External Tool**: kubectl (Kubernetes CLI)

### Architecture Pattern
The project follows the **Elm Architecture** (Model-View-Update):
- **Model**: Application state (`internal/app/model.go`)
- **View**: Rendering logic (`internal/app/model_view.go`)
- **Update**: State transitions (`internal/app/model_update.go`)
- **Messages**: Events that trigger updates (`internal/app/messages.go`)

---

## Go Best Practices

### Package Organization

```go
// ✅ GOOD: Clear package purpose, single responsibility
package kubectl

type Client struct {
    // ...
}

func (c *Client) Execute(args ...string) (CommandResult, error) {
    // ...
}
```

```go
// ❌ BAD: Mixed concerns in one package
package app

// Don't mix kubectl logic with UI logic in the same file
```

### Error Handling

```go
// ✅ GOOD: Explicit error handling, user-friendly messages
func (s *Store) Load() ([]Favourite, error) {
    data, err := os.ReadFile(s.path)
    if err != nil {
        if os.IsNotExist(err) {
            return []Favourite{}, nil // Empty list is valid
        }
        return nil, fmt.Errorf("failed to read favourites: %w", err)
    }
    // ...
}
```

```go
// ❌ BAD: Swallowing errors or panicking
func (s *Store) Load() []Favourite {
    data, _ := os.ReadFile(s.path) // Don't ignore errors
    // ...
}
```

### Struct Design

```go
// ✅ GOOD: Clear field names, grouped by purpose, documented
type Model struct {
    // Core dependencies
    kubectlClient *kubectl.Client
    favStore      *favourites.Store
    
    // Current screen and navigation state
    currentScreen  Screen
    previousScreen Screen
    
    // User selections throughout the wizard
    selectedResource ResourceType
    selectedAction   Action
    
    // UI components
    list      list.Model
    viewport  viewport.Model
    textInput textinput.Model
}
```

### Constants and Enums

```go
// ✅ GOOD: Use iota for enums, provide String() method
type Screen int

const (
    MainMenuScreen Screen = iota
    ResourceSelectionScreen
    ActionSelectionScreen
)

func (s Screen) String() string {
    switch s {
    case MainMenuScreen:
        return "Main Menu"
    case ResourceSelectionScreen:
        return "Resource Selection"
    default:
        return "Unknown"
    }
}
```

---

## Bubble Tea Architecture

### The Elm Architecture Flow

```
User Input → Message → Update(Model, Message) → New Model → View(Model) → Display
     ↑                                                                        ↓
     └────────────────────────────────────────────────────────────────────────┘
```

### Model Structure

The Model is the **single source of truth** for application state:

```go
type Model struct {
    // Dependencies (injected, immutable)
    kubectlClient *kubectl.Client
    favStore      *favourites.Store
    
    // State (mutable through Update)
    currentScreen Screen
    selectedResource ResourceType
    
    // UI Components (managed by Bubble Tea)
    list list.Model
    viewport viewport.Model
}
```

**Key Principles:**
- All state lives in the Model
- State changes only through Update()
- No global variables
- Dependencies are injected

### Update Method Pattern

```go
// ✅ GOOD: Clear message handling, returns commands
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    
    case tea.KeyMsg:
        return m.handleKeyPress(msg)
    
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        return m, nil
    
    case commandResultMsg:
        return m.handleCommandResult(msg)
    
    default:
        return m.updateComponents(msg)
    }
}
```

**Best Practices:**
- Handle one message type per case
- Delegate to helper methods for complex logic
- Always return (Model, Cmd)
- Update UI components last

### View Method Pattern

```go
// ✅ GOOD: Screen-based rendering, clear separation
func (m Model) View() string {
    if m.err != nil {
        return m.renderError()
    }
    
    switch m.currentScreen {
    case MainMenuScreen:
        return m.renderMainMenu()
    case ResourceSelectionScreen:
        return m.renderResourceSelection()
    case CommandOutputScreen:
        return m.renderCommandOutput()
    default:
        return "Unknown screen"
    }
}
```

**Best Practices:**
- One render method per screen
- Handle errors at the top level
- Use Lipgloss for consistent styling
- Keep view logic simple (no business logic)

### Custom Messages

```go
// ✅ GOOD: Descriptive message types for async operations
type commandResultMsg struct {
    result kubectl.CommandResult
    err    error
}

type resourceListMsg struct {
    resources []string
    err       error
}

// Command that returns a message
func executeCommandCmd(client *kubectl.Client, cmd string) tea.Cmd {
    return func() tea.Msg {
        result, err := client.Execute(cmd)
        return commandResultMsg{result: result, err: err}
    }
}
```

**When to Use Custom Messages:**
- Async operations (file I/O, command execution)
- Background tasks
- Delayed actions
- External events

### Commands (tea.Cmd)

```go
// ✅ GOOD: Commands for side effects
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if msg.String() == "enter" {
            // Return a command to execute kubectl
            return m, executeCommandCmd(m.kubectlClient, m.currentCommand)
        }
    }
    return m, nil
}
```

**Command Types:**
- `tea.Cmd`: Function that returns a `tea.Msg`
- `tea.Batch()`: Execute multiple commands
- `tea.Sequence()`: Execute commands in order
- `nil`: No side effects

---

## Project Structure Guidelines

### File Organization

The project uses **file splitting by concern** for the main model:

```
internal/app/
├── model.go                      # Core model struct and initialization
├── model_commands.go             # Command execution logic
├── model_contexts_namespaces.go  # Context/namespace operations
├── model_favourites_hotkeys.go   # Favourites and hotkeys management
├── model_navigation.go           # Screen navigation logic
├── model_saved_outputs.go        # Saved outputs management
├── model_selection.go            # Resource/action selection
├── model_update.go               # Main Update method
├── model_view.go                 # Main View method
├── messages.go                   # Custom message types
└── navigation.go                 # Screen/Resource/Action enums
```

**When to Split Files:**
- File exceeds ~500 lines
- Distinct functional area (commands, navigation, etc.)
- Can be understood independently
- Clear naming convention (model_*.go)

### Package Structure

```
internal/
├── app/          # Main application logic (Bubble Tea model)
├── kubectl/      # kubectl command execution wrapper
├── favourites/   # Favourites data model and persistence
├── hotkeys/      # Hotkey bindings data model and persistence
├── history/      # Command history data model and persistence
└── ui/           # Reusable UI components
```

**Package Guidelines:**
- One clear responsibility per package
- Minimal dependencies between packages
- Public API in package root
- Internal helpers in separate files

### Dependency Direction

```
cmd/kube-wizard
    ↓
internal/app (depends on all below)
    ↓
internal/kubectl, internal/favourites, internal/hotkeys, internal/history, internal/ui
```

**Rules:**
- `cmd` depends on `internal/app` only
- `internal/app` orchestrates all other packages
- Other `internal/*` packages are independent
- No circular dependencies

---

## Code Style and Conventions

### Naming Conventions

```go
// ✅ GOOD: Clear, descriptive names
type ResourceType int
const ResourcePods ResourceType = 0

func (m Model) navigateToResourceSelection() (Model, tea.Cmd)
func (m Model) handleActionSelection() (Model, tea.Cmd)
func (c *Client) ExecuteCommand(args ...string) (CommandResult, error)

// ❌ BAD: Unclear abbreviations
type RT int
const RP RT = 0

func (m Model) navToRes() (Model, tea.Cmd)
func (m Model) handleAS() (Model, tea.Cmd)
```

**Conventions:**
- Use full words, not abbreviations (except common ones: `msg`, `cmd`, `err`)
- Methods start with lowercase (private) or uppercase (public)
- Constants use CamelCase
- Enums use descriptive prefixes (ResourcePods, ActionGet)

### Function Organization

```go
// ✅ GOOD: Logical grouping with comments
// Navigation methods

func (m Model) navigateToMainMenu() (Model, tea.Cmd) { }
func (m Model) navigateToResourceSelection() (Model, tea.Cmd) { }
func (m Model) navigateToActionSelection() (Model, tea.Cmd) { }

// Command execution methods

func (m Model) executeCommand() tea.Cmd { }
func (m Model) handleCommandResult(msg commandResultMsg) (Model, tea.Cmd) { }

// UI rendering methods

func (m Model) renderMainMenu() string { }
func (m Model) renderResourceSelection() string { }
```

### Comments and Documentation

```go
// ✅ GOOD: Package documentation
// Package kubectl provides a wrapper around kubectl command execution.
// It handles command building, execution, and result parsing.
package kubectl

// ✅ GOOD: Function documentation
// Execute runs a kubectl command with the given arguments and returns the result.
// It captures both stdout and stderr, and returns an error if the command fails.
func (c *Client) Execute(args ...string) (CommandResult, error) {
    // ...
}

// ✅ GOOD: Complex logic explanation
// Build the command with flags. We need to handle namespace specially
// because it can be either a flag (-n namespace) or part of the resource
// name (pod/name -n namespace).
```

### Error Messages

```go
// ✅ GOOD: User-friendly, actionable error messages
return fmt.Errorf("failed to load favourites from %s: %w", s.path, err)
return fmt.Errorf("kubectl command failed: %s", stderr)
return fmt.Errorf("no pods found in namespace %s", namespace)

// ❌ BAD: Generic or technical error messages
return fmt.Errorf("error: %v", err)
return errors.New("failed")
```

---

## Testing Guidelines

### Unit Testing

```go
// ✅ GOOD: Table-driven tests
func TestBuildCommand(t *testing.T) {
    tests := []struct {
        name         string
        resource     ResourceType
        action       Action
        resourceName string
        flags        []string
        want         string
    }{
        {
            name:     "get pods",
            resource: ResourcePods,
            action:   ActionGet,
            flags:    []string{},
            want:     "kubectl get pods",
        },
        {
            name:         "describe pod with namespace",
            resource:     ResourcePods,
            action:       ActionDescribe,
            resourceName: "my-pod",
            flags:        []string{"-n", "default"},
            want:         "kubectl describe pod my-pod -n default",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := buildCommand(tt.resource, tt.action, tt.resourceName, tt.flags)
            if got != tt.want {
                t.Errorf("buildCommand() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Testing Bubble Tea Models

```go
// ✅ GOOD: Test state transitions
func TestModelUpdate(t *testing.T) {
    m := NewModel()
    
    // Test navigation
    m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
    if m.currentScreen != ResourceSelectionScreen {
        t.Errorf("Expected ResourceSelectionScreen, got %v", m.currentScreen)
    }
    
    // Test selection
    m.selectedResource = ResourcePods
    m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
    if m.currentScreen != ActionSelectionScreen {
        t.Errorf("Expected ActionSelectionScreen, got %v", m.currentScreen)
    }
}
```

### Mocking External Dependencies

```go
// ✅ GOOD: Interface for testability
type KubectlExecutor interface {
    Execute(args ...string) (CommandResult, error)
}

type Model struct {
    kubectlClient KubectlExecutor // Use interface instead of concrete type
}

// Mock for testing
type mockKubectlClient struct {
    result CommandResult
    err    error
}

func (m *mockKubectlClient) Execute(args ...string) (CommandResult, error) {
    return m.result, m.err
}
```

---

## Common Patterns

### Screen Navigation Pattern

```go
// ✅ GOOD: Consistent navigation pattern
func (m Model) navigateToScreen(screen Screen) (Model, tea.Cmd) {
    m.previousScreen = m.currentScreen
    m.currentScreen = screen
    
    // Initialize screen-specific state
    switch screen {
    case ResourceSelectionScreen:
        items := []list.Item{
            ui.NewSimpleItem("Pods", "Kubernetes pods"),
            ui.NewSimpleItem("Deployments", "Kubernetes deployments"),
        }
        m.list = ui.NewList(items, "Select Resource Type", m.width, m.height)
    }
    
    return m, nil
}
```

### List Selection Pattern

```go
// ✅ GOOD: Reusable list handling
func (m Model) handleListSelection() (Model, tea.Cmd) {
    selectedItem := m.list.SelectedItem()
    if selectedItem == nil {
        return m, nil
    }
    
    item := selectedItem.(ui.SimpleItem)
    
    switch m.currentScreen {
    case ResourceSelectionScreen:
        return m.handleResourceSelection(item.Title())
    case ActionSelectionScreen:
        return m.handleActionSelection(item.Title())
    }
    
    return m, nil
}
```

### Async Command Execution Pattern

```go
// ✅ GOOD: Command execution with loading state
func (m Model) executeCommand() tea.Cmd {
    return func() tea.Msg {
        result, err := m.kubectlClient.Execute(m.currentCommand)
        return commandResultMsg{result: result, err: err}
    }
}

func (m Model) handleCommandResult(msg commandResultMsg) (Model, tea.Cmd) {
    if msg.err != nil {
        m.err = msg.err
        return m, nil
    }
    
    m.viewport.SetContent(msg.result.Output)
    return m.navigateToScreen(CommandOutputScreen)
}
```

### State Reset Pattern

```go
// ✅ GOOD: Clear state when returning to main menu
func (m Model) resetWizardState() Model {
    m.selectedResource = 0
    m.selectedAction = 0
    m.selectedResourceName = ""
    m.selectedFlags = []string{}
    m.customNamespace = ""
    m.currentCommand = ""
    m.err = nil
    return m
}
```

### Persistence Pattern

```go
// ✅ GOOD: Store pattern for persistence
type Store struct {
    path string
}

func NewStore() (*Store, error) {
    home, err := os.UserHomeDir()
    if err != nil {
        return nil, err
    }
    return &Store{
        path: filepath.Join(home, ".kube-wizard-favourites.json"),
    }, nil
}

func (s *Store) Load() ([]Favourite, error) {
    // Load from file
}

func (s *Store) Save(favourites []Favourite) error {
    // Save to file
}
```

---

## Troubleshooting

### Common Issues

#### Issue: Screen not updating after state change

```go
// ❌ PROBLEM: Forgot to return updated model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    m.currentScreen = NewScreen
    return m, nil // ✅ Must return m, not the old model
}
```

#### Issue: UI components not responding

```go
// ❌ PROBLEM: Not updating UI components
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        return m.handleKeyPress(msg)
    }
    return m, nil // ❌ UI components never get the message
}

// ✅ SOLUTION: Update components in default case
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        return m.handleKeyPress(msg)
    default:
        return m.updateComponents(msg) // ✅ Pass to components
    }
}
```

#### Issue: Command not executing

```go
// ❌ PROBLEM: Returning nil command
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if msg.String() == "enter" {
        m.executeCommand() // ❌ Not returning the command
        return m, nil
    }
    return m, nil
}

// ✅ SOLUTION: Return the command
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if msg.String() == "enter" {
        return m, m.executeCommand() // ✅ Return the command
    }
    return m, nil
}
```

#### Issue: Race conditions with async operations

```go
// ❌ PROBLEM: Modifying model in goroutine
func (m Model) executeCommand() tea.Cmd {
    return func() tea.Msg {
        result, err := m.kubectlClient.Execute(m.currentCommand)
        m.viewport.SetContent(result.Output) // ❌ Don't modify model here
        return commandResultMsg{result: result, err: err}
    }
}

// ✅ SOLUTION: Return message, update in Update()
func (m Model) executeCommand() tea.Cmd {
    return func() tea.Msg {
        result, err := m.kubectlClient.Execute(m.currentCommand)
        return commandResultMsg{result: result, err: err} // ✅ Just return message
    }
}

func (m Model) handleCommandResult(msg commandResultMsg) (Model, tea.Cmd) {
    m.viewport.SetContent(msg.result.Output) // ✅ Update model here
    return m, nil
}
```

### Debugging Tips

1. **Add logging**: Use `log.Printf()` to debug state transitions
2. **Check message flow**: Ensure messages are being handled
3. **Verify command returns**: Make sure commands are returned from Update()
4. **Test state isolation**: Each screen should have independent state
5. **Use the debugger**: Set breakpoints in Update() to trace execution

### Performance Considerations

1. **Avoid expensive operations in View()**: View is called frequently
2. **Cache computed values**: Don't recalculate on every render
3. **Limit list sizes**: Use pagination for large lists
4. **Debounce rapid updates**: Use tea.Tick for rate limiting
5. **Profile with pprof**: Identify bottlenecks in production

---

## Additional Resources

### Official Documentation
- [Bubble Tea Tutorial](https://github.com/charmbracelet/bubbletea/tree/master/tutorials)
- [Bubbles Components](https://github.com/charmbracelet/bubbles)
- [Lipgloss Styling](https://github.com/charmbracelet/lipgloss)
- [Effective Go](https://go.dev/doc/effective_go)

### Example Projects
- [Glow](https://github.com/charmbracelet/glow) - Markdown reader
- [Soft Serve](https://github.com/charmbracelet/soft-serve) - Git server
- [VHS](https://github.com/charmbracelet/vhs) - Terminal recorder

### Community
- [Charm Discord](https://charm.sh/chat)
- [Bubble Tea Discussions](https://github.com/charmbracelet/bubbletea/discussions)

---

## Conclusion

This guide provides the foundation for developing and extending the Kubernetes Wizard. Follow these patterns and principles to maintain code quality, consistency, and maintainability.

**Key Takeaways:**
- Embrace the Elm architecture (Model-View-Update)
- Keep state in the Model, not in global variables
- Use custom messages for async operations
- Split files by concern, not by size
- Test state transitions, not just functions
- Write user-friendly error messages
- Document complex logic

Happy coding! 🚀