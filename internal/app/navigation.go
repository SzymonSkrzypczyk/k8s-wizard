package app

// Screen represents different screens in the wizard
type Screen int

const (
	// MainMenuScreen is the initial screen with main options
	MainMenuScreen Screen = iota
	// ResourceSelectionScreen allows selecting resource type (pods/deployments)
	ResourceSelectionScreen
	// ActionSelectionScreen allows selecting action (get/describe/logs)
	ActionSelectionScreen
	// ResourceNameSelectionScreen allows selecting specific resource name
	ResourceNameSelectionScreen
	// FlagsSelectionScreen allows selecting command flags/options
	FlagsSelectionScreen
	// CommandPreviewScreen shows the command before execution
	CommandPreviewScreen
	// CommandOutputScreen shows the command output
	CommandOutputScreen
	// FavouritesListScreen shows saved favourites
	FavouritesListScreen
	// SaveFavouriteScreen allows naming a favourite
	SaveFavouriteScreen
	// RenameFavouriteScreen allows renaming an existing favourite
	RenameFavouriteScreen
)

// ResourceType represents the type of Kubernetes resource
type ResourceType int

const (
	ResourcePods ResourceType = iota
	ResourceDeployments
)

// Action represents an action to perform on a resource
type Action int

const (
	ActionGet Action = iota
	ActionDescribe
	ActionLogs
)

// String returns the string representation of a ResourceType
func (r ResourceType) String() string {
	switch r {
	case ResourcePods:
		return "Pods"
	case ResourceDeployments:
		return "Deployments"
	default:
		return "Unknown"
	}
}

// String returns the string representation of an Action
func (a Action) String() string {
	switch a {
	case ActionGet:
		return "Get"
	case ActionDescribe:
		return "Describe"
	case ActionLogs:
		return "Logs"
	default:
		return "Unknown"
	}
}

// buildCommand constructs the kubectl command string based on selections
func buildCommand(resource ResourceType, action Action, resourceName string, flags []string) string {
	cmd := "kubectl "

	switch action {
	case ActionGet:
		if resource == ResourcePods {
			cmd += "get pods"
		} else {
			cmd += "get deployments"
		}
	case ActionDescribe:
		cmd += "describe pod " + resourceName
	case ActionLogs:
		cmd += "logs " + resourceName
	}

	// Append flags if any
	for _, flag := range flags {
		if flag != "" {
			cmd += " " + flag
		}
	}

	return cmd
}
