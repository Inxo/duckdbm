package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// processMacros replaces macros in the SQL file with environment variable values.
func processMacros(content string) (string, error) {
	// Regex to match macros of the form {{ENV_VAR}}
	re := regexp.MustCompile(`\{\{([A-Z0-9_]+)\}\}`)

	return re.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the environment variable name from the macro
		varName := strings.Trim(match, "{}")
		value := os.Getenv(varName)
		if value == "" {
			fmt.Printf("Warning: Environment variable %s is not set\n", varName)
		}
		return value
	}), nil
}
