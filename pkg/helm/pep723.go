package helm

import (
	_ "embed"
	"fmt"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

// Metadata represents the expected requirements for a Python script, according to PEP 723
type Metadata struct {
	RequiresPython string   `toml:"requires-python"`
	Dependencies   []string `toml:"dependencies"`
}

// Validate checks if the script is PEP 723 compliant
func (schema Metadata) Validate() bool {
	return schema.ensurePY312() && schema.ensureDeps()
}

func (schema Metadata) ensurePY312() bool {
	// TODO: support `==3.12.4`, `>=3.11` too.
	return schema.RequiresPython == ">=3.12"
}

func (schema Metadata) ensureDeps() bool {
	// TODO: check all packages exists
	return true
}

// NewMetadata extracts and parses PEP 723 script blocks from a Python script
func NewMetadata(script string) (Metadata, error) {
	// Regex pattern to match PEP 723 script blocks
	// Matches: # /// script\n...content...\n# ///
	regexPattern := `(?m)^# /// (?P<type>[a-zA-Z0-9-]+)$\s(?P<content>(^#(| .*)$\s)+)^# ///$`

	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return Metadata{}, fmt.Errorf("failed to compile regex: %w", err)
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
		return Metadata{}, fmt.Errorf("multiple script blocks found")
	}

	// If no script block found, return nil
	if len(scriptMatches) == 0 {
		return Metadata{}, nil
	}

	// Extract and clean the content
	content := scriptMatches[0][2] // The 'content' group
	lines := strings.Split(content, "\n")

	var cleanedLines []string
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "# "):
			cleanedLines = append(cleanedLines, line[2:])
		case strings.HasPrefix(line, "#"):
			cleanedLines = append(cleanedLines, line[1:])
		default:
			cleanedLines = append(cleanedLines, line)
		}
	}

	// Join the cleaned lines
	cleanedContent := strings.Join(cleanedLines, "\n")

	// Parse TOML content
	var schema Metadata
	err = toml.Unmarshal([]byte(cleanedContent), &schema)
	if err != nil {
		return Metadata{}, fmt.Errorf("failed to parse TOML: %w", err)
	}

	return schema, nil
}

// func checkPEP723(script string) error {
// 	// check that the script is PEP 723 compliant
// 	// PEP 723: https://peps.python.org/pep-0723/
// 	// PEP 723 is a standard for writing Python scripts that are compliant with the Python language specification.

// 	// Try to read the script block
// 	schema, err := NewMetadata(script)
// 	fmt.Printf("Schema: %#v\n", schema)
// 	if err != nil {
// 		return fmt.Errorf("failed to read metadata: %w", err)
// 	}
// 	if !schema.Validate() {
// 		return fmt.Errorf("schema is not valid")
// 	}

// 	return nil
// }
