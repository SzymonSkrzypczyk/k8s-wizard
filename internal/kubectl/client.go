package kubectl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Client wraps kubectl command execution
type Client struct{}

// NewClient creates a new kubectl client
func NewClient() *Client {
	return &Client{}
}

// CheckKubectlInstalled verifies if kubectl is available in the PATH
func (c *Client) CheckKubectlInstalled() error {
	_, err := exec.LookPath("kubectl")
	if err != nil {
		return fmt.Errorf("kubectl not found in PATH: %w. Please ensure kubectl is installed and configured", err)
	}
	return nil
}

// GetKubectlVersion returns the client version of kubectl
func (c *Client) GetKubectlVersion() (int, int, error) {
	cmd := exec.Command("kubectl", "version", "--client", "-o", "json")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Run()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get kubectl version: %w", err)
	}

	var versionInfo struct {
		ClientVersion struct {
			Major string `json:"major"`
			Minor string `json:"minor"`
		} `json:"clientVersion"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &versionInfo); err != nil {
		// Fallback for older kubectl that might not output JSON correctly or in different format
		return 0, 0, fmt.Errorf("failed to parse kubectl version: %w", err)
	}

	major, err := strconv.Atoi(versionInfo.ClientVersion.Major)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid major version: %s", versionInfo.ClientVersion.Major)
	}

	// Minor version can sometimes contain symbols like +, so we need to clean it
	minorStr := versionInfo.ClientVersion.Minor
	for i, char := range minorStr {
		if char < '0' || char > '9' {
			minorStr = minorStr[:i]
			break
		}
	}

	minor, err := strconv.Atoi(minorStr)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid minor version: %s", versionInfo.ClientVersion.Minor)
	}

	return major, minor, nil
}

// CommandResult holds the output of a kubectl command
type CommandResult struct {
	Command string
	Output  string
	Error   string
}

// NodeInfo represents information about a single node
type NodeInfo struct {
	Name              string
	Status            string
	Roles             string
	Age               string
	Version           string
	InternalIP        string
	CPUCapacity       string
	MemoryCapacity    string
	CPUAllocatable    string
	MemoryAllocatable string
	CPUUsage          string
	MemoryUsage       string
	PodCount          string
	PodCapacity       string
}

// ClusterInfo represents comprehensive cluster information
type ClusterInfo struct {
	Context           string
	Nodes             []NodeInfo
	TotalNodes        int
	ReadyNodes        int
	TotalCPU          string
	TotalMemory       string
	AllocatableCPU    string
	AllocatableMemory string
	TotalPods         int
	NamespaceCount    int
	Version           string
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

// GetClusterInfo retrieves comprehensive cluster information
func (c *Client) GetClusterInfo() (*ClusterInfo, error) {
	// Get current context
	context, err := c.GetCurrentContext()
	if err != nil {
		return nil, fmt.Errorf("failed to get current context: %w", err)
	}

	info := &ClusterInfo{
		Context: context,
	}

	// Get nodes information
	nodes, err := c.getNodesInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes info: %w", err)
	}
	info.Nodes = nodes
	info.TotalNodes = len(nodes)

	// Count ready nodes
	readyCount := 0
	for _, node := range nodes {
		if node.Status == "Ready" {
			readyCount++
		}
	}
	info.ReadyNodes = readyCount

	// Calculate total resources
	totalCPU := 0.0
	totalMemory := 0.0
	allocatableCPU := 0.0
	allocatableMemory := 0.0

	for _, node := range nodes {
		if cpu, err := parseResourceValue(node.CPUCapacity); err == nil {
			totalCPU += cpu
		}
		if mem, err := parseResourceValue(node.MemoryCapacity); err == nil {
			totalMemory += mem
		}
		if cpu, err := parseResourceValue(node.CPUAllocatable); err == nil {
			allocatableCPU += cpu
		}
		if mem, err := parseResourceValue(node.MemoryAllocatable); err == nil {
			allocatableMemory += mem
		}
	}

	info.TotalCPU = formatCPU(totalCPU)
	info.TotalMemory = formatMemory(totalMemory)
	info.AllocatableCPU = formatCPU(allocatableCPU)
	info.AllocatableMemory = formatMemory(allocatableMemory)

	// Get total pod count
	podCount, err := c.getTotalPodCount()
	if err == nil {
		info.TotalPods = podCount
	}

	// Get namespace count
	namespaces, err := c.ListNamespaceNames()
	if err == nil {
		info.NamespaceCount = len(namespaces)
	}

	// Get cluster version
	version, err := c.getClusterVersion()
	if err == nil {
		info.Version = version
	}

	return info, nil
}

// getNodesInfo retrieves detailed information about all nodes
func (c *Client) getNodesInfo() ([]NodeInfo, error) {
	// Get nodes with custom columns for basic info
	result, err := c.execute("get", "nodes", "-o", "json")
	if err != nil {
		return nil, err
	}
	if result.Error != "" {
		return nil, fmt.Errorf("kubectl error: %s", result.Error)
	}

	// Parse JSON response
	var nodesData struct {
		Items []struct {
			Metadata struct {
				Name              string            `json:"name"`
				CreationTimestamp string            `json:"creationTimestamp"`
				Labels            map[string]string `json:"labels"`
			} `json:"metadata"`
			Status struct {
				Conditions []struct {
					Type   string `json:"type"`
					Status string `json:"status"`
				} `json:"conditions"`
				Addresses []struct {
					Type    string `json:"type"`
					Address string `json:"address"`
				} `json:"addresses"`
				Capacity struct {
					CPU    string `json:"cpu"`
					Memory string `json:"memory"`
					Pods   string `json:"pods"`
				} `json:"capacity"`
				Allocatable struct {
					CPU    string `json:"cpu"`
					Memory string `json:"memory"`
					Pods   string `json:"pods"`
				} `json:"allocatable"`
				NodeInfo struct {
					KubeletVersion string `json:"kubeletVersion"`
				} `json:"nodeInfo"`
			} `json:"status"`
		} `json:"items"`
	}

	if err := json.Unmarshal([]byte(result.Output), &nodesData); err != nil {
		return nil, fmt.Errorf("failed to parse nodes JSON: %w", err)
	}

	nodes := make([]NodeInfo, 0, len(nodesData.Items))
	for _, item := range nodesData.Items {
		node := NodeInfo{
			Name:              item.Metadata.Name,
			Version:           item.Status.NodeInfo.KubeletVersion,
			CPUCapacity:       item.Status.Capacity.CPU,
			MemoryCapacity:    item.Status.Capacity.Memory,
			CPUAllocatable:    item.Status.Allocatable.CPU,
			MemoryAllocatable: item.Status.Allocatable.Memory,
			PodCapacity:       item.Status.Capacity.Pods,
		}

		// Get node status
		for _, condition := range item.Status.Conditions {
			if condition.Type == "Ready" {
				if condition.Status == "True" {
					node.Status = "Ready"
				} else {
					node.Status = "NotReady"
				}
				break
			}
		}

		// Get internal IP
		for _, addr := range item.Status.Addresses {
			if addr.Type == "InternalIP" {
				node.InternalIP = addr.Address
				break
			}
		}

		// Get roles from labels
		roles := []string{}
		for key := range item.Metadata.Labels {
			if strings.HasPrefix(key, "node-role.kubernetes.io/") {
				role := strings.TrimPrefix(key, "node-role.kubernetes.io/")
				if role != "" {
					roles = append(roles, role)
				}
			}
		}
		if len(roles) == 0 {
			node.Roles = "<none>"
		} else {
			node.Roles = strings.Join(roles, ",")
		}

		// Get pod count for this node
		podCount, _ := c.getNodePodCount(node.Name)
		node.PodCount = strconv.Itoa(podCount)

		// Try to get metrics (may not be available if metrics-server is not installed)
		metrics, _ := c.getNodeMetrics(node.Name)
		if metrics != nil {
			node.CPUUsage = metrics["cpu"]
			node.MemoryUsage = metrics["memory"]
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}

// getNodeMetrics retrieves CPU and memory usage for a node
func (c *Client) getNodeMetrics(nodeName string) (map[string]string, error) {
	result, err := c.execute("top", "node", nodeName, "--no-headers")
	if err != nil {
		return nil, err
	}
	if result.Error != "" {
		return nil, fmt.Errorf("kubectl error: %s", result.Error)
	}

	// Parse output: NAME CPU(cores) CPU% MEMORY(bytes) MEMORY%
	fields := strings.Fields(result.Output)
	if len(fields) < 5 {
		return nil, fmt.Errorf("unexpected output format")
	}

	return map[string]string{
		"cpu":    fields[1] + " (" + fields[2] + ")",
		"memory": fields[3] + " (" + fields[4] + ")",
	}, nil
}

// getNodePodCount returns the number of pods running on a specific node
func (c *Client) getNodePodCount(nodeName string) (int, error) {
	result, err := c.execute("get", "pods", "--all-namespaces", "--field-selector", "spec.nodeName="+nodeName, "-o", "json")
	if err != nil {
		return 0, err
	}
	if result.Error != "" {
		return 0, fmt.Errorf("kubectl error: %s", result.Error)
	}

	var podsData struct {
		Items []interface{} `json:"items"`
	}

	if err := json.Unmarshal([]byte(result.Output), &podsData); err != nil {
		return 0, err
	}

	return len(podsData.Items), nil
}

// getTotalPodCount returns the total number of pods across all namespaces
func (c *Client) getTotalPodCount() (int, error) {
	result, err := c.execute("get", "pods", "--all-namespaces", "-o", "json")
	if err != nil {
		return 0, err
	}
	if result.Error != "" {
		return 0, fmt.Errorf("kubectl error: %s", result.Error)
	}

	var podsData struct {
		Items []interface{} `json:"items"`
	}

	if err := json.Unmarshal([]byte(result.Output), &podsData); err != nil {
		return 0, err
	}

	return len(podsData.Items), nil
}

// getClusterVersion retrieves the Kubernetes cluster version
func (c *Client) getClusterVersion() (string, error) {
	result, err := c.execute("version", "--short")
	if err != nil {
		return "", err
	}
	if result.Error != "" {
		return "", fmt.Errorf("kubectl error: %s", result.Error)
	}

	// Parse version output
	lines := strings.Split(result.Output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Server Version") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				return strings.TrimSpace(parts[1]), nil
			}
		}
	}

	return "Unknown", nil
}

// parseResourceValue converts Kubernetes resource strings to numeric values
func parseResourceValue(value string) (float64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("empty value")
	}

	// Handle CPU values (e.g., "4", "4000m")
	if strings.HasSuffix(value, "m") {
		val := strings.TrimSuffix(value, "m")
		num, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0, err
		}
		return num / 1000.0, nil
	}

	// Handle memory values (e.g., "16Gi", "16384Mi", "16777216Ki")
	multipliers := map[string]float64{
		"Ki": 1024,
		"Mi": 1024 * 1024,
		"Gi": 1024 * 1024 * 1024,
		"Ti": 1024 * 1024 * 1024 * 1024,
		"K":  1000,
		"M":  1000 * 1000,
		"G":  1000 * 1000 * 1000,
		"T":  1000 * 1000 * 1000 * 1000,
	}

	for suffix, multiplier := range multipliers {
		if strings.HasSuffix(value, suffix) {
			val := strings.TrimSuffix(value, suffix)
			num, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return 0, err
			}
			return num * multiplier, nil
		}
	}

	// Plain number
	return strconv.ParseFloat(value, 64)
}

// formatCPU formats CPU value in cores
func formatCPU(cores float64) string {
	if cores >= 1 {
		return fmt.Sprintf("%.1f cores", cores)
	}
	return fmt.Sprintf("%.0f millicores", cores*1000)
}

// formatMemory formats memory value in human-readable format
func formatMemory(bytes float64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TiB", bytes/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.2f GiB", bytes/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MiB", bytes/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KiB", bytes/KB)
	default:
		return fmt.Sprintf("%.0f B", bytes)
	}
}
