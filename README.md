# Kubernetes Wizard

A terminal-based wizard for running common kubectl commands with an intuitive, guided interface built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Features

### Core Functionality
- **Wizard-style interface**: Step-by-step guidance through kubectl operations
- **Multiple resource types**: Pods, Deployments, Services, Nodes, ConfigMaps, Secrets, Ingress
- **Supported actions**:
  - `Get`: List resources with various output formats
  - `Describe`: Get detailed information about specific resources
  - `Logs`: View pod logs with follow, tail, and time filters
  - `Extract Field`: Extract and decode secret fields
- **Common flags/options**: Select from commonly used kubectl flags for each command
- **Custom namespace**: Specify a custom namespace with user-provided value

### Advanced Features
- **Favourites**: Save frequently used commands for quick access, rename them as needed
- **Hotkeys**: Bind keyboard shortcuts to favourite commands for instant execution
- **Command History**: View and re-run previously executed commands with timestamps
- **Saved Outputs**: Save command outputs with versioning support for later reference
- **Context & Namespace Management**: Switch between Kubernetes contexts and set default namespaces
- **Cluster Connectivity Check**: Verify connection to your Kubernetes cluster
- **Scrollable output**: View command results in a scrollable viewport with mouse support
- **Clean architecture**: Modular design following best practices for easy extension

## Prerequisites

- Go 1.21 or higher
- `kubectl` installed and configured
- Active Kubernetes cluster connection

## Installation

### Using `go install` (Recommended)

You can install the latest version of `kube-wizard` directly using Go:

```bash
go install github.com/SzymonSkrzypczyk/k8s-wizard/cmd/kube-wizard@latest
```

Ensure your `GOBIN` directory (usually `$HOME/go/bin` or `%USERPROFILE%\go\bin`) is in your system's `PATH`.

### From Source

```bash
# Clone the repository
git clone https://github.com/SzymonSkrzypczyk/k8s-wizard.git
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
.\kube-wizard.exe
```

## Usage

### Main Menu
When you start the application, you'll see the following options:
1. **Run Command** - Execute kubectl commands through the wizard
2. **Favourites** - View and run saved commands
3. **Command History** - View and re-run previous commands
4. **Saved Outputs** - View previously saved command outputs
5. **Hotkeys** - Manage keyboard shortcuts for favourite commands
6. **Check Cluster Connectivity** - Verify connection to Kubernetes cluster
7. **Exit** - Quit the application

### Running Commands
1. Select "Run Command" from the main menu
2. Choose a resource type:
   - Pods
   - Deployments
   - Services
   - Nodes
   - ConfigMaps
   - Secrets
   - Ingress
3. Select an action:
   - **Get**: List all resources
   - **Describe**: Get detailed information about a specific resource
   - **Logs**: View logs from a specific pod (Pods/Deployments only)
   - **Extract Field**: Decode and view secret fields (Secrets only)
4. If needed, select a specific resource name from the list
5. Select flags/options (multiple selection supported):
   - Use **Space** to toggle flags on/off (checkboxes: [ ] or [x])
   - Select **multiple flags** to combine them in one command
   - Select **-n <namespace>** to specify a custom namespace (will prompt for input)
   - Choose **Done (Continue)** when finished selecting
   - Available flags per command:
     - For `get`: -o wide, -o yaml, -o json, --show-labels, -A (all namespaces), -n <namespace>
     - For `describe`: --show-events=true, -n <namespace>
     - For `logs`: -f (follow), --tail=N, --since=Xm/h, --previous, -n <namespace>
6. If namespace flag was selected, enter the namespace name
7. Preview the complete command with all selected flags and choose to:
   - **Execute**: Run the command immediately
   - **Save as Favourite**: Save for later use
   - **Back**: Return to previous screen
8. After execution, you can:
   - **Save Output**: Save the output for later reference
   - **Bind Hotkey**: Assign a keyboard shortcut to this command
   - **Back to Main Menu**: Return to the main menu

### Managing Favourites
- From the main menu, select "Favourites"
- Press **Enter** on a favourite to execute it
- Press **'d'** to delete a favourite
- Press **'r'** to rename a favourite
- Press **'h'** to bind a hotkey to a favourite
- Favourites are stored in `~/.kube-wizard-favourites.json`

### Using Hotkeys
- Bind hotkeys to your favourite commands for instant execution
- Press the assigned key from the main menu to run the command immediately
- Manage hotkeys from the "Hotkeys" menu option
- Hotkeys are stored in `~/.kube-wizard-hotkeys.json`

### Command History
- View all previously executed commands with timestamps
- Re-run any command from history
- History is stored in `~/.kube-wizard-history.json`

### Saved Outputs
- Save command outputs with custom names
- View saved outputs with versioning support
- Rename or delete saved outputs
- Outputs are stored in `~/.kube-wizard-outputs/`

### Context & Namespace Management
- Switch between Kubernetes contexts
- Set a default namespace for commands
- View current context and namespace

### Keyboard Shortcuts
- **Arrow keys / j/k**: Navigate lists
- **Enter**: Select item / Confirm selection
- **Space**: Toggle flag selection (in flags screen)
- **Esc**: Go back to previous screen
- **q**: Quit (from main menu) or return to main menu (from other screens)
- **d**: Delete item (in favourites/saved outputs list)
- **r**: Rename item (in favourites/saved outputs list)
- **h**: Bind hotkey (in favourites list)
- **Custom hotkeys**: Execute bound commands from main menu

## Project Structure

```
k8s-wizard/
├── cmd/
│   └── kube-wizard/
│       └── main.go                          # Application entry point with CLI flags
├── internal/
│   ├── app/
│   │   ├── model.go                         # Core Bubble Tea model and state
│   │   ├── model_commands.go                # Command execution logic
│   │   ├── model_contexts_namespaces.go     # Context/namespace management
│   │   ├── model_favourites_hotkeys.go      # Favourites and hotkeys handling
│   │   ├── model_navigation.go              # Screen navigation logic
│   │   ├── model_saved_outputs.go           # Saved outputs management
│   │   ├── model_selection.go               # Resource/action selection
│   │   ├── model_update.go                  # Bubble Tea Update method
│   │   ├── model_view.go                    # Bubble Tea View method
│   │   ├── messages.go                      # Custom Bubble Tea messages
│   │   └── navigation.go                    # Screen types and navigation helpers
│   ├── kubectl/
│   │   └── client.go                        # kubectl command execution wrapper
│   ├── favourites/
│   │   ├── model.go                         # Favourite data structure
│   │   └── store.go                         # JSON persistence for favourites
│   ├── hotkeys/
│   │   ├── model.go                         # Hotkey binding data structure
│   │   └── store.go                         # JSON persistence for hotkeys
│   ├── history/
│   │   ├── model.go                         # Command history entry structure
│   │   └── store.go                         # JSON persistence for history
│   └── ui/
│       ├── lists.go                         # Reusable list components
│       └── viewport.go                      # Output display helpers
├── go.mod
├── go.sum
├── README.md
└── agents.md                                # Development guidelines (see below)
```

## Extending the Application

### Adding New Resource Types

1. **Add resource type constant in `internal/app/navigation.go`**:
```go
const (
    ResourcePods ResourceType = iota
    ResourceDeployments
    ResourceServices
    ResourceNodes
    ResourceConfigMaps
    ResourceSecrets
    ResourceIngress
    ResourceNewType  // Add new resource type
)
```

2. **Update the String() method** to return the display name
3. **Update `navigateToResourceSelection()` in `internal/app/model_selection.go`** to show in menu
4. **Update `buildCommand()` in `internal/app/navigation.go`** to handle the new resource type

### Adding New Actions

1. **Add action type in `internal/app/navigation.go`**:
```go
const (
    ActionGet Action = iota
    ActionDescribe
    ActionLogs
    ActionExtractField
    ActionNewAction  // Add new action
)
```

2. **Update the String() method** to return the display name
3. **Update `navigateToActionSelection()` in `internal/app/model_selection.go`** to show appropriate actions for resources
4. **Update `buildCommand()` and command execution logic** in `internal/app/model_commands.go`

### Adding Command Flags

1. **Update flag selection in `internal/app/model_selection.go`**:
   - Modify `navigateToFlagsSelection()` to include new flags for specific commands
   - Add flag handling logic in the appropriate sections

2. **Update command building in `internal/app/navigation.go`**:
   - Ensure `buildCommand()` properly appends the new flags

### Adding New Screens

1. **Add screen constant in `internal/app/navigation.go`**:
```go
const (
    MainMenuScreen Screen = iota
    // ... existing screens
    NewCustomScreen  // Add new screen
)
```

2. **Implement navigation logic in `internal/app/model_navigation.go`**
3. **Add view rendering in `internal/app/model_view.go`**
4. **Handle updates in `internal/app/model_update.go`**

## Architecture Notes

### Design Principles
- **Separation of concerns**: kubectl operations, UI components, and business logic are separated into distinct packages
- **No global state**: All state is contained in the Bubble Tea model, passed through the Elm architecture
- **Command execution**: Uses `os/exec` directly (no shell interpretation) for security
- **Error handling**: Errors are displayed in the UI, not as crashes
- **Extensibility**: Modular design allows easy addition of new commands and features

### Key Patterns
- **Model splitting**: The main model is split across multiple files by concern (commands, navigation, selection, etc.)
- **Message passing**: Custom messages (in `messages.go`) enable async operations and state updates
- **Store pattern**: Separate store packages handle persistence (favourites, hotkeys, history)
- **UI components**: Reusable UI components in the `ui` package for consistency

### State Management
- All application state lives in the `Model` struct
- State transitions happen through the `Update()` method
- Screen navigation is managed through the `Screen` enum and navigation functions
- User selections are accumulated in the model until command execution

### Testing
- Unit tests are provided in `internal/app/model_test.go`
- Test the model's Update and Init methods
- Mock kubectl client for testing command execution

For detailed development guidelines and best practices, see [agents.md](agents.md).

## License

See LICENSE file for details.
