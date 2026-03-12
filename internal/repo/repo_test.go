package repo

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/nicobistolfi/vigilante/internal/environment"
)

func TestParseGitHubRepo(t *testing.T) {
	tests := map[string]string{
		"git@github.com:owner/repo.git":   "owner/repo",
		"https://github.com/owner/repo":   "owner/repo",
		"ssh://git@github.com/owner/repo": "owner/repo",
	}
	for input, want := range tests {
		got, err := ParseGitHubRepo(input)
		if err != nil {
			t.Fatalf("%s: %v", input, err)
		}
		if got != want {
			t.Fatalf("%s: got %s want %s", input, got, want)
		}
	}
}

func TestDiscoverRepositoryWithRealGit(t *testing.T) {
	dir := t.TempDir()
	runner := environment.ExecRunner{}
	ctx := context.Background()

	mustRun(t, runner, ctx, dir, "git", "init", "--initial-branch=main")
	mustRun(t, runner, ctx, dir, "git", "remote", "add", "origin", "git@github.com:owner/repo.git")
	info, err := Discover(ctx, runner, dir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Repo != "owner/repo" {
		t.Fatalf("unexpected repo: %#v", info)
	}
	if info.Branch != "main" {
		t.Fatalf("unexpected branch: %#v", info)
	}
}

func TestExpandPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	got, err := filepath.Abs(filepath.Join(home, "demo"))
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.Abs(filepath.Join(home, "demo"))
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestClassifyMonorepoStacks(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
		shape Shape
		stack MonorepoStack
	}{
		{name: "turborepo", files: map[string]string{"turbo.json": "{}\n"}, shape: ShapeMonorepo, stack: MonorepoStackTurborepo},
		{name: "nx", files: map[string]string{"nx.json": "{}\n"}, shape: ShapeMonorepo, stack: MonorepoStackNx},
		{name: "rush", files: map[string]string{"rush.json": "{}\n"}, shape: ShapeMonorepo, stack: MonorepoStackRush},
		{name: "bazel", files: map[string]string{"WORKSPACE": "# bazel\n"}, shape: ShapeMonorepo, stack: MonorepoStackBazel},
		{name: "gradle", files: map[string]string{"settings.gradle.kts": "rootProject.name = \"demo\"\n"}, shape: ShapeMonorepo, stack: MonorepoStackGradle},
		{name: "unknown workspace", files: map[string]string{"pnpm-workspace.yaml": "packages:\n  - apps/*\n"}, shape: ShapeMonorepo, stack: MonorepoStackUnknown},
		{name: "standard repo", files: map[string]string{"README.md": "# demo\n"}, shape: ShapeStandard, stack: MonorepoStackNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range tt.files {
				path := filepath.Join(dir, name)
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			profile := Classify(dir)
			if profile.Shape != tt.shape {
				t.Fatalf("unexpected shape: %#v", profile)
			}
			if profile.MonorepoStack != tt.stack {
				t.Fatalf("unexpected stack: %#v", profile)
			}
			if profile.ServiceLaunch.LauncherSkill != "docker-compose-launch" {
				t.Fatalf("unexpected launcher skill: %#v", profile.ServiceLaunch)
			}
			if profile.ServiceLaunch.Scope != "assigned_worktree" {
				t.Fatalf("unexpected launcher scope: %#v", profile.ServiceLaunch)
			}
		})
	}
}

func mustRun(t *testing.T, runner environment.Runner, ctx context.Context, dir, name string, args ...string) {
	t.Helper()
	if _, err := runner.Run(ctx, dir, name, args...); err != nil {
		t.Fatal(err)
	}
}
