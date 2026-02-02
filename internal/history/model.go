package history

import "time"

// Entry represents a command in history.
type Entry struct {
	Command   string    `json:"command"`
	Timestamp time.Time `json:"timestamp"`
}

// NewEntry creates a new history entry.
func NewEntry(command string) Entry {
	return Entry{
		Command:   command,
		Timestamp: time.Now(),
	}
}
