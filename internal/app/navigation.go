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
	// ClusterInfoScreen displays cluster information and metrics
	ClusterInfoScreen
	// DeleteConfirmationScreen asks for confirmation before deleting a resource
	DeleteConfirmationScreen
	// PortInputScreen allows entering ports for port-forwarding
	PortInputScreen
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
	ActionEdit
	ActionDelete
	ActionExec
	ActionPortForward
	ActionTop
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
	case ActionEdit:
		return "Edit"
	case ActionDelete:
		return "Delete"
	case ActionExec:
		return "Exec"
	case ActionPortForward:
		return "Port Forward"
	case ActionTop:
		return "Top (Metrics)"
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
	case ActionEdit:
		cmd += "edit " + getResourceShortName(resource) + " " + resourceName
	case ActionDelete:
		cmd += "delete " + getResourceShortName(resource) + " " + resourceName
	case ActionExec:
		if resource == ResourcePods {
			cmd += "exec -it " + resourceName + " -- /bin/sh"
		} else if resource == ResourceDeployments {
			cmd += "exec -it deployment/" + resourceName + " -- /bin/sh"
		}
	case ActionPortForward:
		if resource == ResourcePods {
			cmd += "port-forward pod/" + resourceName
		} else if resource == ResourceServices {
			cmd += "port-forward svc/" + resourceName
		} else if resource == ResourceDeployments {
			cmd += "port-forward deployment/" + resourceName
		}
	case ActionTop:
		if resource == ResourcePods {
			cmd += "top pod"
		} else if resource == ResourceNodes {
			cmd += "top node"
		} else {
			cmd += "top " + getResourceShortName(resource)
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

func getResourceShortName(r ResourceType) string {
	switch r {
	case ResourcePods:
		return "pod"
	case ResourceDeployments:
		return "deployment"
	case ResourceServices:
		return "service"
	case ResourceNodes:
		return "node"
	case ResourceConfigMaps:
		return "configmap"
	case ResourceSecrets:
		return "secret"
	case ResourceIngress:
		return "ingress"
	default:
		return ""
	}
}
