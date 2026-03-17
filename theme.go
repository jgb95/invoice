package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Color is an RGB triple.
type Color [3]uint8

// Theme defines the visual style of a generated PDF.
type Theme struct {
	// PrimaryText is used for main body text and values.
	PrimaryText Color `json:"primary_text" yaml:"primary_text"`
	// SecondaryText is used for labels, sub-headings, and muted text.
	SecondaryText Color `json:"secondary_text" yaml:"secondary_text"`
	// Accent is used for the title, total row, and divider lines.
	Accent Color `json:"accent" yaml:"accent"`
	// Line is used for separator / rule lines.
	Line Color `json:"line" yaml:"line"`
	// Logo is an optional path to an image that is shown in the header.
	// The --logo CLI flag always takes precedence over this value.
	Logo string `json:"logo" yaml:"logo"`
}

// builtinThemes holds the themes that ship with the binary.
var builtinThemes = map[string]Theme{
	// "default" reproduces the original style of the application exactly.
	"default": {
		PrimaryText:   Color{0, 0, 0},
		SecondaryText: Color{75, 75, 75},
		Accent:        Color{55, 55, 55},
		Line:          Color{225, 225, 225},
	},
	// "bitcoin" keeps the same clean layout but adds Bitcoin-orange accents.
	"bitcoin": {
		PrimaryText:   Color{0, 0, 0},
		SecondaryText: Color{75, 75, 75},
		Accent:        Color{247, 147, 26}, // #F7931A
		Line:          Color{247, 147, 26},
	},
}

// loadTheme resolves a theme by name or file path.
// If nameOrPath is empty, the "default" theme is returned.
func loadTheme(nameOrPath string) (Theme, error) {
	if nameOrPath == "" {
		return builtinThemes["default"], nil
	}

	// Check built-in themes first (case-insensitive).
	if t, ok := builtinThemes[strings.ToLower(nameOrPath)]; ok {
		return t, nil
	}

	// Otherwise treat it as a file path.
	data, err := os.ReadFile(nameOrPath)
	if err != nil {
		return Theme{}, fmt.Errorf("theme: cannot read file %q: %w", nameOrPath, err)
	}

	var t Theme
	if strings.HasSuffix(nameOrPath, ".json") {
		if err := json.Unmarshal(data, &t); err != nil {
			return Theme{}, fmt.Errorf("theme: invalid JSON in %q: %w", nameOrPath, err)
		}
	} else if strings.HasSuffix(nameOrPath, ".yaml") || strings.HasSuffix(nameOrPath, ".yml") {
		if err := yaml.Unmarshal(data, &t); err != nil {
			return Theme{}, fmt.Errorf("theme: invalid YAML in %q: %w", nameOrPath, err)
		}
	} else {
		return Theme{}, fmt.Errorf("theme: unsupported file type for %q (use .json, .yaml, or .yml)", nameOrPath)
	}

	return t, nil
}
