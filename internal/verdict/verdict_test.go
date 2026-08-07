package verdict

import "testing"

// The regression this package exists to prevent: a verdict must never be read
// out of the command string, only out of what the runner actually printed.
func TestVerdictNeverComesFromTheCommand(t *testing.T) {
	poisoned := []string{
		`git commit -m "fix the thing where 3 failed"`,
		`grep -r " failed" ./src`,
		`pytest -k "not failed"`,
		`echo "1 failed" > notes.txt`,
	}
	for _, command := range poisoned {
		if result, ok := Of(command, "collected 12 items\n"); ok {
			t.Errorf("Of(%q, cleanOutput) recorded %q from the command text", command, result)
		}
	}
}

func TestOfReadsRealRunnerOutput(t *testing.T) {
	cases := []struct {
		name    string
		command string
		output  string
		want    string
		record  bool
	}{
		{"pytest failure", "pytest tests/", "== 3 failed, 9 passed in 2.1s ==", "fail", true},
		{"pytest success", "pytest tests/", "== 12 passed in 1.4s ==", "pass", true},
		{"cargo failure", "cargo test", "test result: FAILED. 2 passed; 1 failed", "fail", true},
		{"cargo success", "cargo test", "test result: ok. 14 passed; 0 failed", "pass", true},
		{"go failure", "go test ./...", "--- FAIL: TestThing (0.00s)\nFAIL", "fail", true},
		{"go success", "go test ./...", "ok  \tgithub.com/x/y\t0.101s\n", "pass", true},
		{"tsc failure", "tsc --noEmit", "src/a.ts(3,1): error TS2304: Cannot find name", "fail", true},
		{"ruff success", "ruff check .", "All checks passed!", "pass", true},
		{"not a test run", "ls -la", "3 failed", "", false},
		{"silent runner", "pytest -q", "collected 0 items", "", false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got, ok := Of(testCase.command, testCase.output)
			if ok != testCase.record || got != testCase.want {
				t.Errorf("Of(%q, ...) = %q/%v, want %q/%v",
					testCase.command, got, ok, testCase.want, testCase.record)
			}
		})
	}
}

// Failure has to win, or a run printing "2 failed, 9 passed" reads as a pass.
func TestFailureBeatsPassInMixedOutput(t *testing.T) {
	if got, _ := Read("== 2 failed, 9 passed =="); got != "fail" {
		t.Errorf("mixed output read as %q, want fail", got)
	}
}

// A clean run that reports its zero counts must not read as a failure. The shell
// version matched the bare word " failed" and would have called every one of
// these red.
func TestZeroCountsAreNotFailures(t *testing.T) {
	clean := []string{
		"test result: ok. 14 passed; 0 failed; 0 ignored",
		"Tests: 0 failed, 12 passed, 12 total",
		"Found 0 errors in 3 files",
		"✔ 0 problems (0 errors, 0 warnings)",
	}
	for _, output := range clean {
		if got, _ := Read(output); got == "fail" {
			t.Errorf("Read(%q) = fail, want pass or nothing", output)
		}
	}
}

// The shell version only knew Python and JavaScript runners, so a Rust or Go
// developer's pet could never see a red test.
func TestRecognisesNonPythonJavaScriptRunners(t *testing.T) {
	for _, command := range []string{
		"cargo test --all", "go test ./...", "gradle test", "bundle exec rspec",
		"dotnet test", "swift test", "mix test", "golangci-lint run",
	} {
		if !IsTestRun(command) {
			t.Errorf("IsTestRun(%q) = false, want true", command)
		}
	}
}

// A green run has to clear a red one. Go reports success positionally rather
// than in prose, so matching only phrases left a fixed failure showing red for
// the full two-hour decay window: the fix-verify loop never cleared the flag.
func TestGreenRunClearsAPreviousFailure(t *testing.T) {
	green := []string{
		"ok  \tgithub.com/x/y\t0.101s\n",
		"ok  \tgithub.com/x/y\t(cached)\nok  \tgithub.com/x/z\t1.2s\n",
		"PASS\n",
	}
	for _, output := range green {
		got, ok := Of("go test ./...", output)
		if !ok || got != "pass" {
			t.Errorf("Of(go test, %q) = %q/%v, want pass/true", output, got, ok)
		}
	}
}

// The positional match must not fire on prose that merely starts with "ok".
func TestPositionalPassDoesNotOverreach(t *testing.T) {
	for _, output := range []string{"okay then", "ok", "looks ok to me", "not ok"} {
		if got, ok := Of("go test ./...", output); ok {
			t.Errorf("Of(go test, %q) recorded %q, want nothing", output, got)
		}
	}
}

// A run with both failures and passing packages is still a failure.
func TestMixedGoOutputIsAFailure(t *testing.T) {
	mixed := "ok  \tgithub.com/x/y\t0.1s\n--- FAIL: TestThing\nFAIL\tgithub.com/x/z\t0.2s\n"
	if got, _ := Of("go test ./...", mixed); got != "fail" {
		t.Errorf("mixed go output read as %q, want fail", got)
	}
}
