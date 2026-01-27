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
	result, err := c.execute("get", "pods", "-o", "jsonpath={.items[*].metadata.name}")
	if err != nil {
		return nil, err
	}
	if result.Error != "" {
		return nil, fmt.Errorf("kubectl error: %s", result.Error)
	}

	// Split space-separated pod names
	names := strings.Fields(result.Output)
	return names, nil
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
