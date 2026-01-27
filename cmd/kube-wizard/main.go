package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/k8s-wizard/internal/app"
)

func main() {
	// Initialize the Bubble Tea program with our app model
	p := tea.NewProgram(
		app.NewModel(),
		tea.WithAltScreen(),       // Use alternate screen buffer
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	// Run the program
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running application: %v\n", err)
		os.Exit(1)
	}
}
