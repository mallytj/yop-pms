// rtmcheck verifies that every requirement ID in the reservation RTM
// (docs/requirements/reservations.md) has a corresponding test in
// internal/booking/. Deferred IDs are loaded from
// docs/requirements/reservations.deferred.txt and skipped.
//
// **DEPRECATED** — RTM format superseded by Agentic Engineering Workflow
// (docs/agentic-engineering-workflow.md). Existing IDs maintained for
// historical traceability of implemented code. New work uses Linear Job
// Specs; do not add new IDs to the RTM.
//
// Usage:
//
//	go run ./cmd/tools/rtmcheck            # warn-only, exit 0
//	go run ./cmd/tools/rtmcheck -strict    # exit non-zero on gaps
//	go run ./cmd/tools/rtmcheck -family CRUD
//	go run ./cmd/tools/rtmcheck -pattern 'R-[A-Z]+-[A-Z]+-[0-9]+'
//
// A test "covers" an ID if the literal ID string appears in any
// *_test.go file under -tests and the enclosing test function is not
// a t.Skip-only stub.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	rtmPath      = flag.String("rtm", "docs/requirements/reservations.md", "path to RTM markdown")
	deferredPath = flag.String("deferred", "docs/requirements/reservations.deferred.txt", "path to deferred ID allow-list")
	testsDir     = flag.String("tests", "internal/booking,internal/platform/middleware", "comma-separated directory trees to grep for IDs")
	family       = flag.String("family", "", "restrict check to one family (e.g. CRUD, EDGE, VALID)")
	strict       = flag.Bool("strict", false, "exit non-zero when gaps exist")
	pattern      = flag.String("pattern", `R-RES-[A-Z]+-[0-9]+[a-z]?`, "regex pattern for requirement IDs")
)

func main() {
	flag.Parse()

	idRegex := regexp.MustCompile(*pattern)

	rtmIDs, err := parseRTM(*rtmPath, idRegex)
	if err != nil {
		fmt.Fprintf(os.Stderr, "rtmcheck: read RTM: %v\n", err)
		os.Exit(2)
	}

	deferred, err := parseDeferred(*deferredPath)
	if err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "rtmcheck: read deferred list: %v\n", err)
		os.Exit(2)
	}

	covered, err := scanTests(*testsDir, idRegex)
	if err != nil {
		fmt.Fprintf(os.Stderr, "rtmcheck: scan tests: %v\n", err)
		os.Exit(2)
	}

	var missing []string
	for _, id := range rtmIDs {
		if *family != "" && !inFamily(id, *family) {
			continue
		}
		if isDeferred(id, deferred) {
			continue
		}
		if _, ok := covered[id]; !ok {
			missing = append(missing, id)
		}
	}
	sort.Strings(missing)

	total := 0
	for _, id := range rtmIDs {
		if *family != "" && !inFamily(id, *family) {
			continue
		}
		if isDeferred(id, deferred) {
			continue
		}
		total++
	}

	covCount := total - len(missing)
	fmt.Printf("RTM check: %d / %d covered", covCount, total)
	if *family != "" {
		fmt.Printf(" (family=%s)", *family)
	}
	fmt.Println()

	if len(missing) == 0 {
		fmt.Println("✓ all in-scope RTM IDs have tests")
		return
	}

	fmt.Printf("✗ %d missing test%s:\n", len(missing), pluralS(len(missing)))
	for _, id := range missing {
		fmt.Printf("  - %s\n", id)
	}

	if *strict {
		os.Exit(1)
	}
}

func parseRTM(path string, idRegex *regexp.Regexp) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	seen := make(map[string]struct{})
	var ids []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		for _, m := range idRegex.FindAllString(scanner.Text(), -1) {
			if _, ok := seen[m]; ok {
				continue
			}
			seen[m] = struct{}{}
			ids = append(ids, m)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	sort.Strings(ids)
	return ids, nil
}

func parseDeferred(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var patterns []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if i := strings.Index(line, "#"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if line == "" {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns, scanner.Err()
}

func isDeferred(id string, patterns []string) bool {
	for _, p := range patterns {
		if prefix, ok := strings.CutSuffix(p, "*"); ok {
			if strings.HasPrefix(id, prefix) {
				return true
			}
			continue
		}
		if id == p {
			return true
		}
	}
	return false
}

func scanTests(dirs string, idRegex *regexp.Regexp) (map[string][]string, error) {
	covered := make(map[string][]string)
	for dir := range strings.SplitSeq(dirs, ",") {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		if err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, "_test.go") {
				return nil
			}
			fileIDs, err := parseTestFile(path, idRegex)
			if err != nil {
				return err
			}
			for id, fns := range fileIDs {
				covered[id] = append(covered[id], fns...)
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}
	return covered, nil
}

var (
	funcDecl = regexp.MustCompile(`^func\s+(Test\w+)\s*\(`)
	skipCall = regexp.MustCompile(`\bt\.Skip(?:f|Now)?\s*\(`)
)

// parseTestFile returns a map of RTM ID -> test function names that
// reference the ID and are NOT skip-only stubs.
func parseTestFile(path string, idRegex *regexp.Regexp) (map[string][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	type block struct {
		name       string
		ids        map[string]struct{}
		hasSkip    bool
		hasNonSkip bool
		braceDepth int
		started    bool
	}

	out := make(map[string][]string)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	// Track the contiguous comment block immediately preceding a func
	// declaration — RTM ID annotations live in doc comments.
	var preceding []string

	var cur *block
	flush := func() {
		if cur == nil {
			return
		}
		// Skip-only stubs don't count as coverage.
		if cur.hasSkip && !cur.hasNonSkip {
			cur = nil
			return
		}
		for id := range cur.ids {
			out[id] = append(out[id], cur.name)
		}
		cur = nil
	}

	for scanner.Scan() {
		line := scanner.Text()
		if cur == nil {
			if m := funcDecl.FindStringSubmatch(line); m != nil {
				cur = &block{name: m[1], ids: map[string]struct{}{}}
				cur.braceDepth = strings.Count(line, "{") - strings.Count(line, "}")
				cur.started = strings.Contains(line, "{")
				for _, c := range preceding {
					for _, id := range idRegex.FindAllString(c, -1) {
						cur.ids[id] = struct{}{}
					}
				}
				preceding = preceding[:0]
				continue
			}
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				preceding = append(preceding, line)
			} else if trimmed != "" {
				preceding = preceding[:0]
			}
			continue
		}

		for _, id := range idRegex.FindAllString(line, -1) {
			cur.ids[id] = struct{}{}
		}
		if skipCall.MatchString(line) {
			cur.hasSkip = true
		} else if trimmed := strings.TrimSpace(line); trimmed != "" &&
			!strings.HasPrefix(trimmed, "//") &&
			!strings.HasPrefix(trimmed, "func ") &&
			trimmed != "}" {
			cur.hasNonSkip = true
		}

		cur.braceDepth += strings.Count(line, "{") - strings.Count(line, "}")
		if cur.started && cur.braceDepth <= 0 {
			flush()
		} else if !cur.started && strings.Contains(line, "{") {
			cur.started = true
		}
	}
	flush()
	return out, scanner.Err()
}

func inFamily(id, fam string) bool {
	return strings.HasPrefix(id, "R-RES-"+strings.ToUpper(fam)+"-")
}

func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
