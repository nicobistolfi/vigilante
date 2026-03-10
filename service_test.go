package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderLaunchdPlist(t *testing.T) {
	state := &StateStore{root: filepath.Join(t.TempDir(), ".vigilante")}
	cfg := ServiceConfig{
		Executable: "/Users/test/.local/bin/vigilante",
		PathEnv:    "/opt/homebrew/bin:/usr/bin:/bin",
		HomeDir:    "/Users/test",
	}
	plist := renderLaunchdPlist(state, cfg)
	if !strings.Contains(plist, "<string>daemon</string>") || !strings.Contains(plist, state.LogsDir()) {
		t.Fatalf("unexpected plist: %s", plist)
	}
	if !strings.Contains(plist, cfg.PathEnv) || !strings.Contains(plist, cfg.HomeDir) {
		t.Fatalf("plist missing environment variables: %s", plist)
	}
}

func TestRenderSystemdUnit(t *testing.T) {
	state := &StateStore{root: filepath.Join(t.TempDir(), ".vigilante")}
	cfg := ServiceConfig{
		Executable: "/home/test/.local/bin/vigilante",
		PathEnv:    "/usr/local/bin:/usr/bin:/bin",
		HomeDir:    "/home/test",
	}
	unit := renderSystemdUnit(state, cfg)
	if !strings.Contains(unit, "ExecStart=") || !strings.Contains(unit, state.LogsDir()) {
		t.Fatalf("unexpected unit: %s", unit)
	}
	if !strings.Contains(unit, "Environment=PATH="+cfg.PathEnv) || !strings.Contains(unit, "Environment=HOME="+cfg.HomeDir) {
		t.Fatalf("unit missing environment variables: %s", unit)
	}
}

func TestBuildServiceConfigUsesShellPath(t *testing.T) {
	t.Setenv("HOME", "/Users/test")
	t.Setenv("SHELL", "/bin/zsh")
	t.Setenv("PATH", "/usr/bin:/bin")

	env := &Environment{
		OS: "darwin",
		Runner: fakeRunner{
			outputs: map[string]string{
				`/bin/zsh -lic printf "%s" "$PATH"`: "/opt/homebrew/bin:/Users/test/.local/bin:/usr/bin:/bin",
				`/bin/sh -lc PATH="/opt/homebrew/bin:/Users/test/.local/bin:/usr/bin:/bin" command -v 'git'`:   "/opt/homebrew/bin/git\n",
				`/bin/sh -lc PATH="/opt/homebrew/bin:/Users/test/.local/bin:/usr/bin:/bin" command -v 'gh'`:    "/opt/homebrew/bin/gh\n",
				`/bin/sh -lc PATH="/opt/homebrew/bin:/Users/test/.local/bin:/usr/bin:/bin" command -v 'codex'`: "/Users/test/.local/bin/codex\n",
			},
		},
	}

	cfg, err := buildServiceConfig(context.Background(), env)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PathEnv != "/opt/homebrew/bin:/Users/test/.local/bin:/usr/bin:/bin" {
		t.Fatalf("unexpected PATH: %#v", cfg)
	}
	if cfg.HomeDir != "/Users/test" {
		t.Fatalf("unexpected HOME: %#v", cfg)
	}
}

func TestBuildServiceConfigFailsWhenDaemonPathCannotResolveTools(t *testing.T) {
	t.Setenv("HOME", "/Users/test")
	t.Setenv("SHELL", "/bin/zsh")

	env := &Environment{
		OS: "darwin",
		Runner: fakeRunner{
			outputs: map[string]string{
				`/bin/zsh -lic printf "%s" "$PATH"`:                   "/usr/bin:/bin",
				`/bin/sh -lc PATH="/usr/bin:/bin" command -v 'git'`:   "/usr/bin/git\n",
				`/bin/sh -lc PATH="/usr/bin:/bin" command -v 'gh'`:    "",
				`/bin/sh -lc PATH="/usr/bin:/bin" command -v 'codex'`: "",
			},
			errors: map[string]error{
				`/bin/sh -lc PATH="/usr/bin:/bin" command -v 'gh'`:    errString("missing"),
				`/bin/sh -lc PATH="/usr/bin:/bin" command -v 'codex'`: errString("missing"),
			},
		},
	}

	_, err := buildServiceConfig(context.Background(), env)
	if err == nil || !strings.Contains(err.Error(), "codex, gh") {
		t.Fatalf("unexpected error: %v", err)
	}
}
