package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func noEnv(string) string { return "" }

func TestRunGreets(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run(context.Background(), []string{"-name", "Subbu", "-times", "2"}, noEnv, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code %d, stderr %s", code, errOut.String())
	}
	if out.String() != "Hello, Subbu! (1)\nHello, Subbu! (2)\n" {
		t.Fatalf("stdout = %q", out.String())
	}
	if !strings.Contains(errOut.String(), "msg=greeted") {
		t.Fatalf("log = %q", errOut.String())
	}
}

func TestEnvironmentVariableConfiguresGreeting(t *testing.T) {
	var out bytes.Buffer
	env := func(k string) string { return map[string]string{"APP_GREETING": "Namaste"}[k] }
	run(context.Background(), []string{"-name", "Ravi"}, env, &out, &bytes.Buffer{})
	if out.String() != "Namaste, Ravi! (1)\n" {
		t.Fatalf("got %q", out.String())
	}
}

func TestJSONLogs(t *testing.T) {
	var errOut bytes.Buffer
	run(context.Background(), []string{"-log-format", "json", "-verbose"}, noEnv, &bytes.Buffer{}, &errOut)
	lines := strings.Split(strings.TrimSpace(errOut.String()), "\n")
	var first map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("not JSON: %q", lines[0])
	}
	if first["level"] != "DEBUG" || first["app"] != "greeter" || first["msg"] != "config loaded" {
		t.Fatalf("got %v", first)
	}
}

func TestInvalidInputExitsWithUsageError(t *testing.T) {
	for _, args := range [][]string{{"-times", "99"}, {"-log-format", "xml"}, {"-unknown"}} {
		var errOut bytes.Buffer
		if code := run(context.Background(), args, noEnv, &bytes.Buffer{}, &errOut); code != 2 {
			t.Errorf("%v: exit code %d, want 2", args, code)
		}
	}
}

func TestWatchStopsWhenContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	var errOut bytes.Buffer
	if code := run(ctx, []string{"-watch"}, noEnv, &bytes.Buffer{}, &errOut); code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(errOut.String(), "shutting down cleanly") {
		t.Fatalf("log = %q", errOut.String())
	}
}
