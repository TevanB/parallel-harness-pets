// Package verdict decides whether a shell command was a test run, and if so
// whether it passed.
//
// The command string only ever gates *whether this was a test run*. The verdict
// itself is read from the output alone, because an earlier version scanned the
// command too and any command merely mentioning "3 failed" in a heredoc, a grep
// pattern, or a commit message recorded a failure that never happened.
package verdict

import (
	"regexp"
	"strings"
)

var runners = []string{
	"pytest", "vitest", "jest", "ruff", "eslint", "tsc", "mypy", "pyright",
	"cargo test", "cargo clippy", "cargo check", "go test", "gotestsum",
	"gradle", "rspec", "dotnet test", "swift test", "phpunit", "mix test",
	"golangci-lint", "biome", "clippy", " test",
}

// counted failures must carry a non-zero number. Matching the bare word instead
// reads "test result: ok. 14 passed; 0 failed" as a failure, which is how a clean
// cargo run would dock two hearts.
var countedFailures = regexp.MustCompile(`(?i)\b[1-9]\d*\s+(failed|failures?|errors?|problems?)\b`)

// failures are phrases a runner only prints when something genuinely broke.
var failures = []string{
	"error TS", "test result: FAILED", "--- FAIL", "FAILED (",
	"BUILD FAILED", "Test suite failed", "panic:",
}

// passes are phrases a runner only prints when it finished clean.
var passes = []string{
	" passed", "All checks passed", "test result: ok", "BUILD SUCCESSFUL",
	"0 failures", "no issues found", "Success: no issues",
}

// countedPasses are clean results a runner reports positionally rather than in
// prose. Go prints "ok <package> <time>" per package and "PASS" alone, and
// matching neither leaves a fixed failure showing red until the verdict decays.
var countedPasses = regexp.MustCompile(`(?m)^(ok\s+\S+|PASS)\s*$|^ok\s+\S+\s+[\d.]+m?s`)

// IsTestRun reports whether a command looks like a test or lint invocation.
func IsTestRun(command string) bool {
	lowered := strings.ToLower(command)
	for _, runner := range runners {
		if strings.Contains(lowered, runner) {
			return true
		}
	}
	return false
}

// Read returns "pass", "fail", or false when the output states nothing outright.
//
// Failure wins over success, because a run reporting both has failures in it.
func Read(output string) (string, bool) {
	if countedFailures.MatchString(output) {
		return "fail", true
	}
	for _, phrase := range failures {
		if strings.Contains(output, phrase) {
			return "fail", true
		}
	}
	// Go prints "FAIL" alone on its own line, which is too short to match safely
	// anywhere else in a line of output.
	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) == "FAIL" {
			return "fail", true
		}
	}
	for _, phrase := range passes {
		if strings.Contains(output, phrase) {
			return "pass", true
		}
	}
	if countedPasses.MatchString(output) {
		return "pass", true
	}
	return "", false
}

// Of combines both checks: a verdict is only recorded for a real test run whose
// output says how it went.
func Of(command, output string) (string, bool) {
	if !IsTestRun(command) {
		return "", false
	}
	return Read(output)
}
