package kubectl

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Client wraps kubectl command execution
type Client struct{}

// NewClient creates a new kubectl client
func NewClient() *Client {
	return &Client{}
}

// CommandResult holds the output of a kubectl command
type CommandResult struct {
	Command string
	Output  string
	Error   string
}

// GetPods retrieves all pods in the current namespace
func (c *Client) GetPods() (CommandResult, error) {
	return c.execute("get", "pods")
}

// GetDeployments retrieves all deployments in the current namespace
func (c *Client) GetDeployments() (CommandResult, error) {
	return c.execute("get", "deployments")
}

// DescribePod describes a specific pod
func (c *Client) DescribePod(podName string) (CommandResult, error) {
	return c.execute("describe", "pod", podName)
}

// GetPodLogs retrieves logs from a specific pod
func (c *Client) GetPodLogs(podName string) (CommandResult, error) {
	return c.execute("logs", podName)
}

// ListPodNames returns a list of pod names in the current namespace
func (c *Client) ListPodNames() ([]string, error) {
	return c.listResourceNames("pods")
}

// ListDeploymentNames returns a list of deployment names in the current namespace
func (c *Client) ListDeploymentNames() ([]string, error) {
	return c.listResourceNames("deployments")
}

// ListServiceNames returns a list of service names in the current namespace
func (c *Client) ListServiceNames() ([]string, error) {
	return c.listResourceNames("services")
}

// ListNodeNames returns a list of node names in the cluster
func (c *Client) ListNodeNames() ([]string, error) {
	return c.listResourceNames("nodes")
}

// ListConfigMapNames returns a list of configmap names in the current namespace
func (c *Client) ListConfigMapNames() ([]string, error) {
	return c.listResourceNames("configmaps")
}

// ListSecretNames returns a list of secret names in the current namespace
func (c *Client) ListSecretNames() ([]string, error) {
	return c.listResourceNames("secrets")
}

// ListIngressNames returns a list of ingress names in the current namespace
func (c *Client) ListIngressNames() ([]string, error) {
	return c.listResourceNames("ingress")
}

// ListNamespaceNames returns a list of namespaces in the cluster
func (c *Client) ListNamespaceNames() ([]string, error) {
	return c.listResourceNames("namespaces")
}

// ListContexts returns the available kube contexts
func (c *Client) ListContexts() ([]string, error) {
	result, err := c.execute("config", "get-contexts", "-o", "name")
	if err != nil {
		return nil, err
	}
	if result.Error != "" {
		return nil, fmt.Errorf("kubectl error: %s", result.Error)
	}

	names := strings.Fields(result.Output)
	return names, nil
}

// UseContext switches the current kube context
func (c *Client) UseContext(name string) error {
	result, err := c.execute("config", "use-context", name)
	if err != nil {
		return err
	}
	if result.Error != "" {
		return fmt.Errorf("kubectl error: %s", result.Error)
	}
	return nil
}

// GetCurrentContext checks if a Kubernetes cluster context is configured
func (c *Client) GetCurrentContext() (string, error) {
	result, err := c.execute("config", "current-context")
	if err != nil {
		return "", fmt.Errorf("no cluster context configured: %w", err)
	}
	if result.Error != "" {
		return "", fmt.Errorf("no cluster context configured: %s", result.Error)
	}

	context := strings.TrimSpace(result.Output)
	if context == "" {
		return "", fmt.Errorf("no cluster context configured")
	}

	return context, nil
}

// listResourceNames is a helper that lists resource names using a common jsonpath
func (c *Client) listResourceNames(resource string) ([]string, error) {
	result, err := c.execute("get", resource, "-o", "jsonpath={.items[*].metadata.name}")
	if err != nil {
		return nil, err
	}
	if result.Error != "" {
		return nil, fmt.Errorf("kubectl error: %s", result.Error)
	}

	// Split space-separated names
	names := strings.Fields(result.Output)
	return names, nil
}

// ExecuteRaw executes a raw kubectl command string with cluster validation
func (c *Client) ExecuteRaw(commandStr string) (CommandResult, error) {
	// First check if a cluster context is configured
	if _, err := c.GetCurrentContext(); err != nil {
		return CommandResult{
			Command: commandStr,
			Error:   err.Error(),
		}, err
	}

	// Parse the command string to extract kubectl arguments
	// Remove "kubectl " prefix if present
	commandStr = strings.TrimSpace(commandStr)
	if strings.HasPrefix(commandStr, "kubectl ") {
		commandStr = strings.TrimPrefix(commandStr, "kubectl ")
	}

	// Split the command into arguments
	args := strings.Fields(commandStr)
	if len(args) == 0 {
		return CommandResult{
			Command: commandStr,
			Error:   "invalid command",
		}, fmt.Errorf("invalid command")
	}

	return c.execute(args...)
}

// execute runs a kubectl command and captures output
func (c *Client) execute(args ...string) (CommandResult, error) {
	cmd := exec.Command("kubectl", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Build command string for display
	cmdStr := "kubectl " + strings.Join(args, " ")

	err := cmd.Run()

	result := CommandResult{
		Command: cmdStr,
		Output:  stdout.String(),
		Error:   stderr.String(),
	}

	// Return the result even if there's an error
	// The caller can check result.Error for kubectl errors
	return result, err
}
