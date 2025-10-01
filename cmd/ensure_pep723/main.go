// Package main checks that a given python script is PEP 723 compliant.
package main

import (
	_ "embed"
	"fmt"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

//go:embed echo.py
var pythonScript string

type Schema struct {
	RequiresPython string   `toml:"requires-python"`
	Dependencies   []string `toml:"dependencies"`
}

func (schema Schema) Validate() bool {
	return schema.EnsurePy312() && schema.EnsureDependencies()
}

func (schema Schema) EnsurePy312() bool {
	// TODO: support `==3.12.4`, `>=3.11` too.
	return schema.RequiresPython == ">=3.12"
}

func (schema Schema) EnsureDependencies() bool {
	// TODO: check all packages exists
	return true
}

// read extracts and parses PEP 723 script blocks from a Python script
func read(script string) (Schema, error) {
	// Regex pattern to match PEP 723 script blocks
	// Matches: # /// script\n...content...\n# ///
	regexPattern := `(?m)^# /// (?P<type>[a-zA-Z0-9-]+)$\s(?P<content>(^#(| .*)$\s)+)^# ///$`

	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return Schema{}, fmt.Errorf("failed to compile regex: %w", err)
	}

	// Find all matches
	matches := re.FindAllStringSubmatch(script, -1)

	// Filter for 'script' type blocks
	var scriptMatches [][]string
	for _, match := range matches {
		if len(match) >= 2 && match[1] == "script" {
			scriptMatches = append(scriptMatches, match)
		}
	}

	// Check for multiple script blocks
	if len(scriptMatches) > 1 {
		return Schema{}, fmt.Errorf("multiple script blocks found")
	}

	// If no script block found, return nil
	if len(scriptMatches) == 0 {
		return Schema{}, nil
	}

	// Extract and clean the content
	content := scriptMatches[0][2] // The 'content' group
	lines := strings.Split(content, "\n")

	var cleanedLines []string
	for _, line := range lines {
		if strings.HasPrefix(line, "# ") {
			// Remove "# " prefix
			cleanedLines = append(cleanedLines, line[2:])
		} else if strings.HasPrefix(line, "#") {
			// Remove "#" prefix
			cleanedLines = append(cleanedLines, line[1:])
		} else {
			cleanedLines = append(cleanedLines, line)
		}
	}

	// Join the cleaned lines
	cleanedContent := strings.Join(cleanedLines, "\n")

	// Parse TOML content
	var schema Schema
	err = toml.Unmarshal([]byte(cleanedContent), &schema)
	if err != nil {
		return Schema{}, fmt.Errorf("failed to parse TOML: %w", err)
	}

	return schema, nil
}

func checkPEP723(script string) (bool, error) {
	// check that the script is PEP 723 compliant
	// PEP 723: https://peps.python.org/pep-0723/
	// PEP 723 is a standard for writing Python scripts that are compliant with the Python language specification.

	// Try to read the script block
	schema, err := read(script)
	fmt.Printf("Schema: %#v\n", schema)
	if err != nil {
		return false, err
	}
	if !schema.Validate() {
		return false, fmt.Errorf("schema is not valid")
	}

	return true, nil
}

func main() {
	ok, err := checkPEP723(pythonScript)
	if err != nil {
		panic(err)
	}
	fmt.Printf("PEP 723 compliant: %t\n", ok)
}
