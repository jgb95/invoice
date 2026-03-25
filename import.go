package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

// importData loads a Document from a JSON or YAML file, then applies any flags
// that were explicitly set by the user on top of the imported values. Only
// flags that the user actually provided on the command line override the file.
func importData(path string, doc *Document, flags *pflag.FlagSet) error {
	fileText, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("unable to read file: %w", err)
	}

	if strings.HasSuffix(path, ".json") {
		if err := importJSON(fileText, doc); err != nil {
			return err
		}
	} else if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		if err := importYAML(fileText, doc); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("unsupported file type: %q (use .json, .yaml, or .yml)", path)
	}

	// Apply only the flags the user explicitly set, overriding the imported values.
	flags.Visit(func(f *pflag.Flag) {
		switch f.Name {
		case "id":
			doc.Id = f.Value.String()
		case "from":
			doc.From = f.Value.String()
		case "to":
			doc.To = f.Value.String()
		case "logo":
			doc.Logo = f.Value.String()
		case "date":
			doc.Date = f.Value.String()
		case "due":
			doc.Due = f.Value.String()
		case "note":
			doc.Note = f.Value.String()
		case "currency":
			doc.Currency = f.Value.String()
		case "tax":
			if v, err := parseFloat64Flag(f); err == nil {
				doc.Tax = v
			}
		case "discount":
			if v, err := parseFloat64Flag(f); err == nil {
				doc.Discount = v
			}
		case "item":
			doc.Items = splitSliceFlag(f.Value.String())
		case "quantity":
			doc.Quantities = parseIntSliceFlag(f.Value.String())
		case "rate":
			doc.Rates = parseFloat64SliceFlag(f.Value.String())
		case "bitcoin":
			doc.BitcoinAddress = f.Value.String()
		case "lightning":
			doc.LightningAddress = f.Value.String()
		}
	})

	return nil
}

func importJSON(text []byte, doc *Document) error {
	if !json.Valid(text) {
		return fmt.Errorf("JSON file is not valid")
	}
	if err := json.Unmarshal(text, doc); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}
	return nil
}

func importYAML(text []byte, doc *Document) error {
	if err := yaml.Unmarshal(text, doc); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}
	return nil
}

// --- flag value helpers ------------------------------------------------------

func parseFloat64Flag(f *pflag.Flag) (float64, error) {
	var v float64
	_, err := fmt.Sscanf(f.Value.String(), "%f", &v)
	return v, err
}

// splitSliceFlag parses the string representation of a cobra StringSlice flag
// (e.g. "[a,b,c]") into a []string.
func splitSliceFlag(raw string) []string {
	raw = strings.TrimPrefix(raw, "[")
	raw = strings.TrimSuffix(raw, "]")
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return parts
}

// parseIntSliceFlag parses the string representation of a cobra IntSlice flag.
func parseIntSliceFlag(raw string) []int {
	parts := splitSliceFlag(raw)
	result := make([]int, 0, len(parts))
	for _, p := range parts {
		var v int
		if _, err := fmt.Sscanf(p, "%d", &v); err == nil {
			result = append(result, v)
		}
	}
	return result
}

// parseFloat64SliceFlag parses the string representation of a cobra Float64Slice flag.
func parseFloat64SliceFlag(raw string) []float64 {
	parts := splitSliceFlag(raw)
	result := make([]float64, 0, len(parts))
	for _, p := range parts {
		var v float64
		if _, err := fmt.Sscanf(p, "%f", &v); err == nil {
			result = append(result, v)
		}
	}
	return result
}
