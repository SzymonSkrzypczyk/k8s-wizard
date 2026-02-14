package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"github.com/SzymonSkrzypczyk/k8s-wizard/internal/storage"
)

const historyFileName = "kube-wizard-history.json"
const maxHistoryEntries = 50

// Store manages persistence of command history.
type Store struct {
	filePath string
	entries  []Entry
}

// NewStore creates a new history store.
// History is stored in the user's home directory.
func NewStore() (*Store, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(homeDir, historyFileName)
	store := &Store{
		filePath: filePath,
		entries:  []Entry{},
	}

	if err := store.Load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return store, nil
}

// Load reads history from disk.
func (s *Store) Load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &s.entries); err != nil {
		return err
	}

	// Ensure we don't exceed max entries
	if len(s.entries) > maxHistoryEntries {
		s.entries = s.entries[:maxHistoryEntries]
	}

	return nil
}

// Save writes history to disk atomically.
func (s *Store) Save() error {
	// Create backup before saving
	if err := storage.Backup(s.filePath); err != nil {
		// Log error but continue saving
	}

	data, err := json.MarshalIndent(s.entries, "", "  ")
	if err != nil {
		return err
	}

	return storage.WriteAtomic(s.filePath, data)
}

// Add adds a new command to history.
func (s *Store) Add(command string) error {
	entry := NewEntry(command)
	s.entries = append([]Entry{entry}, s.entries...)
	if len(s.entries) > maxHistoryEntries {
		s.entries = s.entries[:maxHistoryEntries]
	}
	return s.Save()
}

// List returns all entries, newest first.
func (s *Store) List() []Entry {
	// Ensure newest first (in case file was manually edited)
	sort.Slice(s.entries, func(i, j int) bool {
		return s.entries[i].Timestamp.After(s.entries[j].Timestamp)
	})
	return s.entries
}

// Get returns an entry by index.
func (s *Store) Get(idx int) (Entry, bool) {
	list := s.List()
	if idx < 0 || idx >= len(list) {
		return Entry{}, false
	}
	return list[idx], true
}
