package main

import (
	"encoding/json"
	"fmt"
	"os"

	"charm.land/lipgloss/v2"
)

// ── Color palette variables ──────────────────────────────────────────────────
var (
	colPrimary   = lipgloss.Color("#F4810A")
	colSecondary = lipgloss.Color("#FFBE3B")
	colMuted     = lipgloss.Color("#9E7D5A")
	colBgDark    = lipgloss.Color("#1A0F00")
	colSubtle    = lipgloss.Color("#3D3D3D")
	colResponse  = lipgloss.Color("#E8D5B0")
)

// ── Config paths ─────────────────────────────────────────────────────────────
const (
	schemesFile = "colorschemes.json"
	configFile  = "color_config.json"
)

// ── Structs ──────────────────────────────────────────────────────────────────
type Theme struct {
	Name      string `json:"name"`
	Primary   string `json:"primary"`
	Secondary string `json:"secondary"`
	Muted     string `json:"muted"`
	BgDark    string `json:"bg_dark"`
	Subtle    string `json:"subtle"`
	Response  string `json:"response"`
}

type Config struct {
	Theme  string `json:"theme"`
	Custom Theme  `json:"custom"`
}

// ── Default Themes ───────────────────────────────────────────────────────────
var defaultThemes = []Theme{
	{
		Name:      "default",
		Primary:   "#F4810A",
		Secondary: "#FFBE3B",
		Muted:     "#9E7D5A",
		BgDark:    "#1A0F00",
		Subtle:    "#3D3D3D",
		Response:  "#E8D5B0",
	},
	{
		Name:      "dracula",
		Primary:   "#FF79C6",
		Secondary: "#50FA7B",
		Muted:     "#6272A4",
		BgDark:    "#282A36",
		Subtle:    "#44475A",
		Response:  "#F8F8F2",
	},
	{
		Name:      "nord",
		Primary:   "#88C0D0",
		Secondary: "#A3BE8C",
		Muted:     "#616E88",
		BgDark:    "#2E3440",
		Subtle:    "#3B4252",
		Response:  "#D8DEE9",
	},
	{
		Name:      "sunset",
		Primary:   "#FF2A7A",
		Secondary: "#FF7A00",
		Muted:     "#A04A9E",
		BgDark:    "#1E0B25",
		Subtle:    "#451B54",
		Response:  "#FAD4DF",
	},
	{
		Name:      "monochrome",
		Primary:   "#FFFFFF",
		Secondary: "#B0B0B0",
		Muted:     "#707070",
		BgDark:    "#121212",
		Subtle:    "#333333",
		Response:  "#E0E0E0",
	},
}

// LoadColorPalette initializes/loads colors from the config and schemes files.
func LoadColorPalette() error {
	// 1. Ensure colorschemes.json exists, otherwise write defaults
	var themes []Theme
	if _, err := os.Stat(schemesFile); os.IsNotExist(err) {
		themes = defaultThemes
		data, err := json.MarshalIndent(themes, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal default themes: %w", err)
		}
		if err := os.WriteFile(schemesFile, data, 0644); err != nil {
			return fmt.Errorf("write default themes: %w", err)
		}
	} else {
		data, err := os.ReadFile(schemesFile)
		if err != nil {
			return fmt.Errorf("read themes file: %w", err)
		}
		if err := json.Unmarshal(data, &themes); err != nil {
			return fmt.Errorf("unmarshal themes: %w", err)
		}
	}

	// 2. Ensure color_config.json exists, otherwise write default config
	var config Config
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		config = Config{
			Theme:  "default",
			Custom: defaultThemes[0], // default copy
		}
		config.Custom.Name = "custom"
		data, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal default config: %w", err)
		}
		if err := os.WriteFile(configFile, data, 0644); err != nil {
			return fmt.Errorf("write default config: %w", err)
		}
	} else {
		data, err := os.ReadFile(configFile)
		if err != nil {
			return fmt.Errorf("read config file: %w", err)
		}
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("unmarshal config: %w", err)
		}
	}

	// 3. Find and apply the configured theme
	var targetTheme *Theme
	if config.Theme == "custom" {
		targetTheme = &config.Custom
	} else {
		for i := range themes {
			if themes[i].Name == config.Theme {
				targetTheme = &themes[i]
				break
			}
		}
	}

	// Fallback to default if not found
	if targetTheme == nil {
		targetTheme = &defaultThemes[0]
	}

	applyTheme(*targetTheme)
	return nil
}

// SaveActiveTheme updates the theme choice in color_config.json.
func SaveActiveTheme(themeName string) error {
	var config Config
	if data, err := os.ReadFile(configFile); err == nil {
		_ = json.Unmarshal(data, &config)
	}
	config.Theme = themeName
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFile, data, 0644)
}

// LoadThemesList reads and returns all available theme names.
func LoadThemesList() ([]string, error) {
	var themes []Theme
	data, err := os.ReadFile(schemesFile)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &themes); err != nil {
		return nil, err
	}
	names := make([]string, len(themes))
	for i, t := range themes {
		names[i] = t.Name
	}
	return names, nil
}

// LoadThemeByName loads a theme from schemes or custom, applies it, and updates styles.
func LoadThemeByName(name string) (Theme, error) {
	if name == "custom" {
		var config Config
		data, err := os.ReadFile(configFile)
		if err != nil {
			return Theme{}, err
		}
		if err := json.Unmarshal(data, &config); err != nil {
			return Theme{}, err
		}
		applyTheme(config.Custom)
		return config.Custom, nil
	}

	var themes []Theme
	data, err := os.ReadFile(schemesFile)
	if err != nil {
		return Theme{}, err
	}
	if err := json.Unmarshal(data, &themes); err != nil {
		return Theme{}, err
	}

	for _, t := range themes {
		if t.Name == name {
			applyTheme(t)
			return t, nil
		}
	}

	return Theme{}, fmt.Errorf("theme not found: %s", name)
}

func applyTheme(t Theme) {
	colPrimary = lipgloss.Color(t.Primary)
	colSecondary = lipgloss.Color(t.Secondary)
	colMuted = lipgloss.Color(t.Muted)
	colBgDark = lipgloss.Color(t.BgDark)
	colSubtle = lipgloss.Color(t.Subtle)
	colResponse = lipgloss.Color(t.Response)
	UpdateStyles()
}

// UpdateStyles re-allocates all style variables when colors change.
func UpdateStyles() {
	userLabelStyle = lipgloss.NewStyle().Foreground(colSecondary).Bold(true)
	userTextStyle = lipgloss.NewStyle().Foreground(colSecondary)
	botLabelStyle = lipgloss.NewStyle().Foreground(colPrimary).Bold(true)
	placeholderStyle = lipgloss.NewStyle().Foreground(colMuted).Italic(true)
	promptStyle = lipgloss.NewStyle().Foreground(colPrimary).Bold(true)
	cursorStyle = lipgloss.NewStyle().Foreground(colSecondary)
	inputTextStyle = lipgloss.NewStyle().Foreground(colSecondary)
	placeholderInputStyle = lipgloss.NewStyle().Foreground(colMuted).Italic(true)
	hintStyle = lipgloss.NewStyle().Foreground(colMuted)
	divStyle = lipgloss.NewStyle().Foreground(colPrimary)
}
