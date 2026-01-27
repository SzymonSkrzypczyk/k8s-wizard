package favourites

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const favouritesFileName = "kube-wizard-favourites.json"

// Store manages persistence of favourites
type Store struct {
	filePath   string
	favourites []Favourite
}

// NewStore creates a new favourites store
// Favourites are stored in the user's home directory
func NewStore() (*Store, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(homeDir, favouritesFileName)
	store := &Store{
		filePath:   filePath,
		favourites: []Favourite{},
	}

	// Load existing favourites if file exists
	if err := store.Load(); err != nil {
		// If file doesn't exist, that's okay - we'll create it on first save
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return store, nil
}

// Load reads favourites from disk
func (s *Store) Load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &s.favourites)
}

// Save writes favourites to disk
func (s *Store) Save() error {
	data, err := json.MarshalIndent(s.favourites, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0644)
}

// Add adds a new favourite and saves to disk
func (s *Store) Add(fav Favourite) error {
	s.favourites = append(s.favourites, fav)
	return s.Save()
}

// Delete removes a favourite by index and saves to disk
func (s *Store) Delete(index int) error {
	if index < 0 || index >= len(s.favourites) {
		return nil
	}

	s.favourites = append(s.favourites[:index], s.favourites[index+1:]...)
	return s.Save()
}

// List returns all favourites
func (s *Store) List() []Favourite {
	return s.favourites
}

// Get returns a favourite by index
func (s *Store) Get(index int) (Favourite, bool) {
	if index < 0 || index >= len(s.favourites) {
		return Favourite{}, false
	}
	return s.favourites[index], true
}

// Rename renames a favourite by index and saves to disk
func (s *Store) Rename(index int, newName string) error {
	if index < 0 || index >= len(s.favourites) {
		return nil
	}

	s.favourites[index].Name = newName
	return s.Save()
}
