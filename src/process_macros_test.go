package main

import (
	"os"
	"testing"
)

func TestProcessMacros(t *testing.T) {
	// Set environment variables for testing
	_ = os.Setenv("TABLE_NAME", "users")
	_ = os.Setenv("COLUMN_NAME", "username")
	defer func() {
		_ = os.Unsetenv("TABLE_NAME")
	}()
	defer func() {
		_ = os.Unsetenv("COLUMN_NAME")
	}()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "single macro replacement",
			input: `
				CREATE TABLE {{TABLE_NAME}} (
					id INTEGER PRIMARY KEY,
					{{COLUMN_NAME}} TEXT NOT NULL
				);
			`,
			expected: `
				CREATE TABLE users (
					id INTEGER PRIMARY KEY,
					username TEXT NOT NULL
				);
			`,
		},
		{
			name: "multiple macros replacement",
			input: `
				INSERT INTO {{TABLE_NAME}} (id, {{COLUMN_NAME}})
				VALUES (1, 'test');
			`,
			expected: `
				INSERT INTO users (id, username)
				VALUES (1, 'test');
			`,
		},
		{
			name: "undefined macro",
			input: `
				CREATE TABLE {{UNDEFINED_MACRO}} (
					id INTEGER PRIMARY KEY
				);
			`,
			expected: `
				CREATE TABLE  (
					id INTEGER PRIMARY KEY
				);
			`, // Macro will be replaced with an empty string
		},
		{
			name: "no macros",
			input: `
				CREATE TABLE simple_table (
					id INTEGER PRIMARY KEY
				);
			`,
			expected: `
				CREATE TABLE simple_table (
					id INTEGER PRIMARY KEY
				);
			`, // Input and expected are the same
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, err := processMacros(test.input)
			if err != nil {
				t.Fatalf("processMacros returned an error: %v", err)
			}

			if output != test.expected {
				t.Errorf("Unexpected result:\nExpected:\n%s\nGot:\n%s", test.expected, output)
			}
		})
	}
}

func TestProcessMacrosWithWarnings(t *testing.T) {
	// Unset a variable to simulate a missing macro
	_ = os.Unsetenv("MISSING_VAR")

	input := `
		CREATE TABLE {{MISSING_VAR}} (
			id INTEGER PRIMARY KEY
		);
	`

	expected := `
		CREATE TABLE  (
			id INTEGER PRIMARY KEY
		);
	` // Macro will be replaced with an empty string

	output, err := processMacros(input)
	if err != nil {
		t.Fatalf("processMacros returned an error: %v", err)
	}

	if output != expected {
		t.Errorf("Unexpected result:\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}
