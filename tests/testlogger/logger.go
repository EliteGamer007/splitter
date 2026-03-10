// Package testlogger provides a concurrency-safe logging mechanism for test results.
// Each test appends a formatted entry to tests/results/<category>_results.txt.
package testlogger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

var mu sync.Mutex

// LogTestResult appends a single test result entry to
// tests/results/<category>_results.txt.
// Call it via defer at the top of each test function:
//
//	start := time.Now()
//	var err error
//	defer func() { testlogger.LogTestResult(t, "unit", start, err) }()
func LogTestResult(t *testing.T, category string, start time.Time, err error) {
	t.Helper()

	mu.Lock()
	defer mu.Unlock()

	// Walk up from the current working directory (or call site) to find go.mod
	root := findModuleRoot()

	resultsDir := filepath.Join(root, "tests", "results")
	if mkErr := os.MkdirAll(resultsDir, 0o755); mkErr != nil {
		t.Logf("[testlogger] failed to create results dir: %v", mkErr)
		return
	}

	resultFile := filepath.Join(resultsDir, category+"_results.txt")
	f, fileErr := os.OpenFile(resultFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if fileErr != nil {
		t.Logf("[testlogger] failed to open results file: %v", fileErr)
		return
	}
	defer f.Close()

	status := "PASS"
	if t.Failed() {
		status = "FAIL"
	}

	elapsed := time.Since(start)

	var b strings.Builder
	b.WriteString(fmt.Sprintf("TestName: %s\n", t.Name()))
	b.WriteString(fmt.Sprintf("Status: %s\n", status))
	b.WriteString(fmt.Sprintf("Execution Time: %s\n", elapsed.String()))
	b.WriteString(fmt.Sprintf("Timestamp: %s\n", time.Now().UTC().Format("2006-01-02 15:04:05")))
	if err != nil {
		b.WriteString(fmt.Sprintf("Error: %v\n", err))
	} else if t.Failed() {
		b.WriteString("Error: test failed (via t.Fail/t.Error)\n")
	}
	b.WriteString("\n")

	if _, writeErr := f.WriteString(b.String()); writeErr != nil {
		t.Logf("[testlogger] failed to write result: %v", writeErr)
	}
}

// findModuleRoot searches upward from the current working directory for go.mod.
func findModuleRoot() string {
	// Try cwd first
	dir, err := os.Getwd()
	if err == nil {
		if root, ok := walkUp(dir); ok {
			return root
		}
	}
	// Fallback: caller source file
	_, file, _, ok := runtime.Caller(2)
	if ok {
		d := filepath.Dir(file)
		if root, ok2 := walkUp(d); ok2 {
			return root
		}
	}
	return dir
}

func walkUp(dir string) (string, bool) {
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}
