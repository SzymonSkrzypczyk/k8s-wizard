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
	// NamespaceInputScreen allows entering a custom namespace
	NamespaceInputScreen
	// CommandPreviewScreen shows the command before execution
	CommandPreviewScreen
	// CommandOutputScreen shows the command output
	CommandOutputScreen
	CommandHelpScreen
	HotkeysListScreen
	HotkeyBindScreen
	ClusterConnectivityScreen
	CommandHistoryScreen
	// FavouritesListScreen shows saved favourites
	FavouritesListScreen
	// SaveFavouriteScreen allows naming a favourite
	SaveFavouriteScreen
	// RenameFavouriteScreen allows renaming an existing favourite
	RenameFavouriteScreen
	// SaveOutputNameScreen allows naming output before saving
	SaveOutputNameScreen
	// SavedOutputsListScreen shows list of saved outputs
	SavedOutputsListScreen
	SavedOutputVersionsScreen
	// SavedOutputViewScreen shows a saved output
	SavedOutputViewScreen
	RenameSavedOutputScreen
	// ContextsNamespacesMenuScreen shows context/namespace operations
	ContextsNamespacesMenuScreen
	// ContextsListScreen shows available kube contexts
	ContextsListScreen
	// NamespacesListScreen shows available namespaces for default selection
	NamespacesListScreen
	// CustomCommandScreen lets users build an arbitrary kubectl command
	CustomCommandScreen
	// SecretFieldSelectionScreen allows selecting a field from a secret
	SecretFieldSelectionScreen
)

// ResourceType represents the type of Kubernetes resource
type ResourceType int

const (
	ResourcePods ResourceType = iota
	ResourceDeployments
	ResourceServices
	ResourceNodes
	ResourceConfigMaps
	ResourceSecrets
	ResourceIngress
)

// Action represents an action to perform on a resource
type Action int

const (
	ActionGet Action = iota
	ActionDescribe
	ActionLogs
	ActionExtractField
)

// String returns the string representation of a ResourceType
func (r ResourceType) String() string {
	switch r {
	case ResourcePods:
		return "Pods"
	case ResourceDeployments:
		return "Deployments"
	case ResourceServices:
		return "Services"
	case ResourceNodes:
		return "Nodes"
	case ResourceConfigMaps:
		return "ConfigMaps"
	case ResourceSecrets:
		return "Secrets"
	case ResourceIngress:
		return "Ingress"
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
	case ActionExtractField:
		return "Extract Field"
	default:
		return "Unknown"
	}
}

// buildCommand constructs the kubectl command string based on selections
func buildCommand(resource ResourceType, action Action, resourceName string, flags []string) string {
	cmd := "kubectl "

	switch action {
	case ActionGet:
		switch resource {
		case ResourcePods:
			cmd += "get pods"
		case ResourceDeployments:
			cmd += "get deployments"
		case ResourceServices:
			cmd += "get services"
		case ResourceNodes:
			cmd += "get nodes"
		case ResourceConfigMaps:
			cmd += "get configmaps"
		case ResourceSecrets:
			cmd += "get secrets"
		case ResourceIngress:
			cmd += "get ingress"
		default:
			cmd += "get"
		}
	case ActionDescribe:
		switch resource {
		case ResourcePods:
			cmd += "describe pod " + resourceName
		case ResourceDeployments:
			cmd += "describe deployment " + resourceName
		case ResourceServices:
			cmd += "describe service " + resourceName
		case ResourceNodes:
			cmd += "describe node " + resourceName
		case ResourceConfigMaps:
			cmd += "describe configmap " + resourceName
		case ResourceSecrets:
			cmd += "describe secret " + resourceName
		case ResourceIngress:
			cmd += "describe ingress " + resourceName
		default:
			cmd += "describe " + resource.String() + " " + resourceName
		}
	case ActionLogs:
		switch resource {
		case ResourcePods:
			cmd += "logs " + resourceName
		case ResourceDeployments:
			cmd += "logs deployment/" + resourceName
		default:
			cmd += "logs " + resourceName
		}
	case ActionExtractField:
		// This is partially handled in handleSecretFieldSelection, but for consistency:
		if resource == ResourceSecrets {
			cmd += "get secret " + resourceName + " -o go-template='{{range $k, $v := .data}}{{$k}}: {{$v | base64decode}}{{\"\\n\"}}{{end}}'"
		}
	}

	// Append flags if any
	for _, flag := range flags {
		if flag != "" {
			cmd += " " + flag
		}
	}

	return cmd
}
