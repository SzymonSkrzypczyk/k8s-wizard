package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Command execution and kubectl helpers.

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

func (m Model) fetchPodNames() tea.Cmd {
	return func() tea.Msg {
		names, err := m.kubectlClient.ListPodNames()
		return resourceNamesLoadedMsg{names: names, err: err}
	}
}

func (m Model) fetchResourceNames() tea.Cmd {
	return func() tea.Msg {
		var (
			names []string
			err   error
		)

		switch m.selectedResource {
		case ResourcePods:
			names, err = m.kubectlClient.ListPodNames()
		case ResourceDeployments:
			names, err = m.kubectlClient.ListDeploymentNames()
		case ResourceServices:
			names, err = m.kubectlClient.ListServiceNames()
		case ResourceNodes:
			names, err = m.kubectlClient.ListNodeNames()
		case ResourceConfigMaps:
			names, err = m.kubectlClient.ListConfigMapNames()
		case ResourceSecrets:
			names, err = m.kubectlClient.ListSecretNames()
		case ResourceIngress:
			names, err = m.kubectlClient.ListIngressNames()
		default:
			err = fmt.Errorf("unsupported resource type: %s", m.selectedResource.String())
		}

		return resourceNamesLoadedMsg{names: names, err: err}
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
