package app

import (
	"github.com/charmbracelet/lipgloss"
)

// ThemeColors defines the color scheme for the application
type ThemeColors struct {
	// Base colors
	Primary   lipgloss.AdaptiveColor
	Secondary lipgloss.AdaptiveColor
	Success   lipgloss.AdaptiveColor
	Warning   lipgloss.AdaptiveColor
	Error     lipgloss.AdaptiveColor
	
	// Background and text
	Background lipgloss.AdaptiveColor
	Text       lipgloss.AdaptiveColor
	Subtle     lipgloss.AdaptiveColor
	
	// Borders and highlights
	Border lipgloss.AdaptiveColor
	
	// UI elements
	Highlight lipgloss.AdaptiveColor
	Focus     lipgloss.AdaptiveColor
}

// GetThemeColors returns the color scheme for the current theme
func GetThemeColors(theme Theme) ThemeColors {
	switch theme {
	case ThemeLight:
		return ThemeColors{
			Primary:   lipgloss.AdaptiveColor{Light: "#0066CC", Dark: "#4DA6FF"},
			Secondary: lipgloss.AdaptiveColor{Light: "#333333", Dark: "#FFFFFF"},
			Success:   lipgloss.AdaptiveColor{Light: "#008000", Dark: "#00D700"},
			Warning:   lipgloss.AdaptiveColor{Light: "#FF8C00", Dark: "#FFCC00"},
			Error:     lipgloss.AdaptiveColor{Light: "#CC0000", Dark: "#FF3333"},
			Background: lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#000000"},
			Text:       lipgloss.AdaptiveColor{Light: "#333333", Dark: "#FFFFFF"},
			Subtle:     lipgloss.AdaptiveColor{Light: "#999999", Dark: "#999999"},
			Border:     lipgloss.AdaptiveColor{Light: "#CCCCCC", Dark: "#333333"},
			Highlight:  lipgloss.AdaptiveColor{Light: "#E6F3FF", Dark: "#003366"},
			Focus:      lipgloss.AdaptiveColor{Light: "#0066CC", Dark: "#4DA6FF"},
		}
	default: // ThemeDark
		return ThemeColors{
			Primary:   lipgloss.AdaptiveColor{Light: "#0066CC", Dark: "#4DA6FF"},
			Secondary: lipgloss.AdaptiveColor{Light: "#333333", Dark: "#FFFFFF"},
			Success:   lipgloss.AdaptiveColor{Light: "#008000", Dark: "#00D700"},
			Warning:   lipgloss.AdaptiveColor{Light: "#FF8C00", Dark: "#FFCC00"},
			Error:     lipgloss.AdaptiveColor{Light: "#CC0000", Dark: "#FF3333"},
			Background: lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#000000"},
			Text:       lipgloss.AdaptiveColor{Light: "#333333", Dark: "#FFFFFF"},
			Subtle:     lipgloss.AdaptiveColor{Light: "#999999", Dark: "#999999"},
			Border:     lipgloss.AdaptiveColor{Light: "#CCCCCC", Dark: "#333333"},
			Highlight:  lipgloss.AdaptiveColor{Light: "#E6F3FF", Dark: "#003366"},
			Focus:      lipgloss.AdaptiveColor{Light: "#0066CC", Dark: "#4DA6FF"},
		}
	}
}

// GetStyle returns a styled string based on the current theme
func (m Model) GetStyle() lipgloss.Style {
	colors := GetThemeColors(m.theme)
	return lipgloss.NewStyle().
		Foreground(colors.Text).
		Background(colors.Background)
}

// GetHeaderStyle returns a styled header
func (m Model) GetHeaderStyle() lipgloss.Style {
	colors := GetThemeColors(m.theme)
	return lipgloss.NewStyle().
		Foreground(colors.Primary).
		Bold(true).
		Underline(true)
}

// GetBorderStyle returns a styled border
func (m Model) GetBorderStyle() lipgloss.Style {
	colors := GetThemeColors(m.theme)
	return lipgloss.NewStyle().
		Foreground(colors.Border)
}

// GetHighlightStyle returns a styled highlight
func (m Model) GetHighlightStyle() lipgloss.Style {
	colors := GetThemeColors(m.theme)
	return lipgloss.NewStyle().
		Background(colors.Highlight).
		Foreground(colors.Text)
}

// GetSuccessStyle returns a styled success message
func (m Model) GetSuccessStyle() lipgloss.Style {
	colors := GetThemeColors(m.theme)
	return lipgloss.NewStyle().
		Foreground(colors.Success).
		Bold(true)
}

// GetErrorStyle returns a styled error message
func (m Model) GetErrorStyle() lipgloss.Style {
	colors := GetThemeColors(m.theme)
	return lipgloss.NewStyle().
		Foreground(colors.Error).
		Bold(true)
}

// GetWarningStyle returns a styled warning message
func (m Model) GetWarningStyle() lipgloss.Style {
	colors := GetThemeColors(m.theme)
	return lipgloss.NewStyle().
		Foreground(colors.Warning).
		Bold(true)
}

// GetHelpStyle returns a styled help text
func (m Model) GetHelpStyle() lipgloss.Style {
	colors := GetThemeColors(m.theme)
	return lipgloss.NewStyle().
		Foreground(colors.Subtle)
}