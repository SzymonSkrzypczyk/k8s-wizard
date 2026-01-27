package favourites

// Favourite represents a saved kubectl command
type Favourite struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// NewFavourite creates a new favourite
func NewFavourite(name, command string) Favourite {
	return Favourite{
		Name:    name,
		Command: command,
	}
}
