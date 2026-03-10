package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestLoggingRunnerLogsCommands(t *testing.T) {
	var entries []string
	runner := LoggingRunner{
		Base: fakeRunner{
			outputs: map[string]string{
				"gh issue list": "[]",
			},
		},
		Logf: func(format string, args ...any) {
			entries = append(entries, sprintf(format, args...))
		},
	}

	if _, err := runner.Run(context.Background(), "/tmp/repo", "gh", "issue", "list"); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("unexpected log entries: %#v", entries)
	}
	if !strings.Contains(entries[0], `command start dir="/tmp/repo" cmd=gh issue list`) {
		t.Fatalf("unexpected start log: %s", entries[0])
	}
	if !strings.Contains(entries[1], "command ok cmd=gh issue list output=[]") {
		t.Fatalf("unexpected success log: %s", entries[1])
	}
}

func TestLoggingRunnerLogsFailures(t *testing.T) {
	var entries []string
	runner := LoggingRunner{
		Base: fakeRunner{
			errors: map[string]error{
				"git status": errors.New("boom"),
			},
		},
		Logf: func(format string, args ...any) {
			entries = append(entries, sprintf(format, args...))
		},
	}

	if _, err := runner.Run(context.Background(), "", "git", "status"); err == nil {
		t.Fatal("expected error")
	}
	if len(entries) != 2 {
		t.Fatalf("unexpected log entries: %#v", entries)
	}
	if !strings.Contains(entries[1], "command failed cmd=git status err=boom") {
		t.Fatalf("unexpected failure log: %s", entries[1])
	}
}

func sprintf(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}
