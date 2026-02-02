package hotkeys

// Binding represents a hotkey binding to a command.
type Binding struct {
	Key     string `json:"key"`
	Name    string `json:"name"`
	Command string `json:"command"`
}
