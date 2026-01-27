# Kubernetes Wizard

A terminal-based wizard for running common kubectl commands with an intuitive, guided interface built with Bubble Tea.

## Features

- **Wizard-style interface**: Step-by-step guidance through kubectl operations
- **Supported commands**:
  - `kubectl get pods` (with flags: -o wide, -o yaml, -o json, --show-labels, -A)
  - `kubectl get deployments` (with flags: -o wide, -o yaml, -o json, --show-labels, -A)
  - `kubectl describe pod <name>` (with flags: --show-events=true)
  - `kubectl logs <pod-name>` (with flags: -f, --tail=N, --since=Xm/h, --previous)
- **Common flags/options**: Select from commonly used kubectl flags for each command
- **Favourites**: Save frequently used commands for quick access, rename them as needed
- **Scrollable output**: View command results in a scrollable viewport
- **Clean architecture**: Modular design for easy extension

## Prerequisites

- Go 1.21 or higher
- `kubectl` installed and configured
- Active Kubernetes cluster connection

## Installation

```bash
# Clone the repository
git clone <repository-url>
cd k8s-wizard

# Download dependencies
go mod tidy

# Build the application
go build -o kube-wizard ./cmd/kube-wizard

# Run the application
./kube-wizard
```

On Windows:
```bash
go build -o kube-wizard.exe ./cmd/kube-wizard
kube-wizard.exe
```

## Usage

### Main Menu
When you start the application, you'll see three options:
1. **Run Command** - Execute kubectl commands through the wizard
2. **Favourites** - View and run saved commands
3. **Exit** - Quit the application

### Running Commands
1. Select "Run Command" from the main menu
2. Choose a resource type (Pods or Deployments)
3. Select an action:
   - **Get**: List all resources
   - **Describe**: Get detailed information about a specific resource
   - **Logs**: View logs from a specific pod
4. If needed, select a specific resource name from the list
5. Select flags/options (multiple selection supported):
   - Use **Space** to toggle flags on/off (checkboxes: [ ] or [x])
   - Select **multiple flags** to combine them in one command
   - Choose **Done (Continue)** when finished selecting
   - Available flags per command:
     - For `get`: -o wide, -o yaml, -o json, --show-labels, -A (all namespaces)
     - For `describe`: --show-events=true
     - For `logs`: -f (follow), --tail=N, --since=Xm/h, --previous
6. Preview the complete command with all selected flags and choose to:
   - **Execute**: Run the command immediately
   - **Save as Favourite**: Save for later use
   - **Back**: Return to previous screen

### Managing Favourites
- From the main menu, select "Favourites"
- Press Enter on a favourite to execute it
- Press 'd' to delete a favourite
- Press 'r' to rename a favourite
- Favourites are stored in `~/kube-wizard-favourites.json`

### Keyboard Shortcuts
- **Arrow keys / j/k**: Navigate lists
- **Enter**: Select item / Confirm selection
- **Space**: Toggle flag selection (in flags screen)
- **Esc**: Go back to previous screen
- **q**: Quit (from main menu) or return to main menu (from other screens)
- **d**: Delete favourite (in favourites list)
- **r**: Rename favourite (in favourites list)

## Project Structure

```
k8s-wizard/
├── cmd/
│   └── kube-wizard/
│       └── main.go              # Application entry point
├── internal/
│   ├── app/
│   │   ├── model.go             # Bubble Tea model and update logic
│   │   ├── messages.go          # Bubble Tea custom messages
│   │   └── navigation.go        # Screen navigation and state
│   ├── kubectl/
│   │   └── client.go            # kubectl command execution
│   ├── favourites/
│   │   ├── model.go             # Favourite data structure
│   │   └── store.go             # JSON persistence layer
│   └── ui/
│       ├── lists.go             # Reusable list components
│       └── viewport.go          # Output display helpers
├── go.mod
├── go.sum
└── README.md
```

## Extending the Application

### Adding New Commands

1. **Add command execution in `internal/kubectl/client.go`**:
```go
func (c *Client) NewCommand(args ...string) (CommandResult, error) {
    return c.execute("new", "command", args...)
}
```

2. **Add action type in `internal/app/navigation.go`**:
```go
const (
    ActionGet Action = iota
    ActionDescribe
    ActionLogs
    ActionNewAction  // Add new action
)
```

3. **Update action selection in `internal/app/model.go`**:
   - Add to `navigateToActionSelection()` to show in menu
   - Add to `handleActionSelection()` to handle selection
   - Add to `executeCommand()` to execute the command

### Adding Command Flags

Modify the `execute()` function in `internal/kubectl/client.go` to accept additional parameters:

```go
func (c *Client) GetPodsWithNamespace(namespace string) (CommandResult, error) {
    return c.execute("get", "pods", "-n", namespace)
}
```

### Adding New Resource Types

1. Add resource type constant in `internal/app/navigation.go`
2. Update `navigateToResourceSelection()` in `internal/app/model.go`
3. Update `navigateToActionSelection()` to show appropriate actions
4. Add command execution logic in `executeCommand()`

## Architecture Notes

- **Separation of concerns**: kubectl operations, UI components, and business logic are separated
- **No global state**: All state is contained in the Bubble Tea model
- **Command execution**: Uses `os/exec` directly (no shell interpretation)
- **Error handling**: Errors are displayed in the UI, not as crashes
- **Extensibility**: Modular design allows easy addition of new commands and features

## License

See LICENSE file for details.
