package app

import (
	"encoding/json"
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

func (m Model) fetchSecretKeys() tea.Cmd {
	return func() tea.Msg {
		// Get the secret as JSON to extract keys
		cmd := fmt.Sprintf("kubectl get secret %s -o json", m.selectedResourceName)
		if m.customNamespace != "" {
			cmd += " -n " + m.customNamespace
		} else if m.defaultNamespace != "" && !m.hasExplicitNamespaceFlag() {
			cmd += " -n " + m.defaultNamespace
		}

		result, err := m.kubectlClient.ExecuteRaw(cmd)
		if err != nil {
			return secretKeysLoadedMsg{err: err}
		}
		if result.Error != "" {
			return secretKeysLoadedMsg{err: fmt.Errorf(result.Error)}
		}

		var secretData struct {
			Metadata struct {
				Name        string            `json:"name"`
				Namespace   string            `json:"namespace"`
				Labels      map[string]string `json:"labels"`
				Annotations map[string]string `json:"annotations"`
			} `json:"metadata"`
			Data       map[string]interface{} `json:"data"`
			StringData map[string]interface{} `json:"stringData"`
			Type       string                 `json:"type"`
		}
		if err := json.Unmarshal([]byte(result.Output), &secretData); err != nil {
			return secretKeysLoadedMsg{err: fmt.Errorf("failed to parse secret JSON: %v", err)}
		}

		// Collect all available fields
		keys := make([]string, 0)

		// Add metadata fields
		keys = append(keys, "metadata.name", "metadata.namespace", "metadata.type")

		// Add label keys if any
		for k := range secretData.Metadata.Labels {
			keys = append(keys, fmt.Sprintf("metadata.labels.%s", k))
		}

		// Add annotation keys if any
		for k := range secretData.Metadata.Annotations {
			keys = append(keys, fmt.Sprintf("metadata.annotations.%s", k))
		}

		// Add data fields (base64 encoded)
		for k := range secretData.Data {
			keys = append(keys, fmt.Sprintf("data.%s", k))
		}

		// Add stringData fields if any (plain text)
		for k := range secretData.StringData {
			keys = append(keys, fmt.Sprintf("stringData.%s", k))
		}

		return secretKeysLoadedMsg{keys: keys}
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
