package hotkeys

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const hotkeysFileName = "kube-wizard-hotkeys.json"

// Store manages persistence of hotkey bindings.
type Store struct {
	filePath string
	bindings map[string]Binding
}

// NewStore creates a new hotkeys store.
// Hotkeys are stored in the user's home directory.
func NewStore() (*Store, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(homeDir, hotkeysFileName)
	store := &Store{
		filePath: filePath,
		bindings: map[string]Binding{},
	}

	if err := store.Load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return store, nil
}

// Load reads bindings from disk.
func (s *Store) Load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var bindings []Binding
	if err := json.Unmarshal(data, &bindings); err != nil {
		return err
	}

	s.bindings = map[string]Binding{}
	for _, b := range bindings {
		key := strings.TrimSpace(strings.ToUpper(b.Key))
		if key == "" {
			continue
		}
		b.Key = key
		s.bindings[key] = b
	}

	return nil
}

// Save writes bindings to disk.
func (s *Store) Save() error {
	bindings := make([]Binding, 0, len(s.bindings))
	for _, b := range s.bindings {
		bindings = append(bindings, b)
	}

	data, err := json.MarshalIndent(bindings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0644)
}

// Get returns a binding for a key.
func (s *Store) Get(key string) (Binding, bool) {
	key = strings.TrimSpace(strings.ToUpper(key))
	b, ok := s.bindings[key]
	return b, ok
}

// Set creates or replaces a binding.
func (s *Store) Set(binding Binding) error {
	key := strings.TrimSpace(strings.ToUpper(binding.Key))
	if key == "" {
		return nil
	}
	binding.Key = key
	s.bindings[key] = binding
	return s.Save()
}

// Delete removes a binding.
func (s *Store) Delete(key string) error {
	key = strings.TrimSpace(strings.ToUpper(key))
	delete(s.bindings, key)
	return s.Save()
}

// List returns all bindings.
func (s *Store) List() map[string]Binding {
	out := make(map[string]Binding, len(s.bindings))
	for k, v := range s.bindings {
		out[k] = v
	}
	return out
}
