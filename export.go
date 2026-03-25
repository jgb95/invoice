package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// exportDocument serialises a Document to YAML or JSON and writes it to disk.
func exportDocument(doc *Document, format, outputPath string) error {
	format = strings.ToLower(format)

	var (
		data []byte
		err  error
	)

	switch format {
	case "yaml", "yml":
		data, err = yaml.Marshal(doc)
	case "json":
		data, err = json.MarshalIndent(doc, "", "  ")
	default:
		return fmt.Errorf("unsupported export format %q — use 'yaml' or 'json'", format)
	}
	if err != nil {
		return fmt.Errorf("failed to serialise document: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", outputPath, err)
	}

	fmt.Printf("Exported %s\n", outputPath)
	return nil
}
