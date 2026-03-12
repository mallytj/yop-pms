package docs

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// TODO FIX

// This file is used to generate the TTM (Technical Test Matrix) documentation.
// It contains metadata about the tests implemented in the codebase.
// It also links each tests to the relevant requirements they cover.
// TTMGenerator is a struct that holds metadata for generating the TTM documentation.

type TTMGenerator struct {
	// TestName is the name of the test case.
	TestName string `json:"Test"`
	// Action is the result of the test case (e.g., output, pass, fail).
	Action string `json:"Action"`
	// Timestamp is the time when the test was executed.
	Timestamp string `json:"Time"`
	// Package is the package where the test case is located.
	Package string `json:"Package"`
	// TestCaseID is the unique identifier for the test case.
}

// TTMEntry represents a single entry in the Technical Test Matrix.
type TTMEntry struct {
	// TestCaseID is the unique identifier for the test case.
	TestCaseID string
	// RequirementIDs is a list of requirement IDs that this test case covers.
	RequirementIDs []string
	// Description provides a brief overview of what the test case covers.
	Description string
}

// Tests2Json converts the test metadata to JSON format for documentation purposes.
func Tests2Json() {
	cmd := exec.Command("go", "test", "./internal/...", "-json")

	outfile, err := os.Create("docs/raw_test_results.json")
	if err != nil {
		fmt.Println("Error creating output file:", err)
		panic(err)
	}
	defer outfile.Close()

	cmd.Stdout = outfile
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			fmt.Println("Tests failed as expected, continuing...")
		} else {
			fmt.Println("Error executing command:", err)
			panic(err)
		}
	}

	fmt.Println("Test results exported to docs/raw_test_results.json")
}

func readJsonToTTMGenerator() {
	// 1. Read raw test results from JSON file.
	readFile, err := os.Open("docs/raw_test_results.json")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer readFile.Close()

	// 2. Parse JSON and extract test metadata.
	parsedJson := json.NewDecoder(readFile)

	var testGenerators []TTMGenerator
	if err := parsedJson.Decode(&testGenerators); err != nil {
		fmt.Println("Error decoding JSON:", err)
		return
	}

	// 3. Map test metadata to TTMEntry structs.
	var ttmEntries []TTMEntry
	for _, tg := range testGenerators {
		entry := extractEntryFromGenerator(tg)
		if entry.TestCaseID != "" {
			ttmEntries = append(ttmEntries, entry)
		}
	}

	// 4. Write TTM entries to a new JSON file for documentation.
	outputFile, err := os.Create("docs/ttm_entries.json")
	if err != nil {
		fmt.Println("Error creating output file:", err)
		return
	}
	defer outputFile.Close()

	encoder := json.NewEncoder(outputFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(ttmEntries); err != nil {
		fmt.Println("Error encoding JSON:", err)
		return
	}

	fmt.Println("TTM entries exported to docs/ttm_entries.json")
}

func extractEntryFromGenerator(tg TTMGenerator) TTMEntry {
	var entry TTMEntry
	// 1. Extract test case ID and requirement IDs from test metadata
	tcRe := regexp.MustCompile(`(TC-[A-Z]+-\d+)_-?_(.*?)\/`)
	matches := tcRe.FindStringSubmatch(tg.TestName)

	if len(matches) > 2 {
		entry.TestCaseID = matches[1]

		entry.Description = strings.ReplaceAll(matches[2], "_", " ")
	}

	return entry
}

func main() {
	Tests2Json()
	readJsonToTTMGenerator()
}
