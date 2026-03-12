package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/nicobistolfi/vigilante/internal/environment"
)

type Shape string

const (
	ShapeStandard Shape = "standard"
	ShapeMonorepo Shape = "monorepo"
)

type MonorepoStack string

const (
	MonorepoStackNone      MonorepoStack = "none"
	MonorepoStackTurborepo MonorepoStack = "turborepo"
	MonorepoStackNx        MonorepoStack = "nx"
	MonorepoStackRush      MonorepoStack = "rush"
	MonorepoStackBazel     MonorepoStack = "bazel"
	MonorepoStackGradle    MonorepoStack = "gradle"
	MonorepoStackUnknown   MonorepoStack = "unknown"
)

type ServiceType string

const (
	ServiceTypeMySQL    ServiceType = "mysql"
	ServiceTypeMariaDB  ServiceType = "mariadb"
	ServiceTypePostgres ServiceType = "postgres"
	ServiceTypeMongoDB  ServiceType = "mongodb"
)

type ServiceLaunchContract struct {
	Required        bool          `json:"required,omitempty"`
	Services        []ServiceType `json:"services,omitempty"`
	Scope           string        `json:"scope,omitempty"`
	LauncherSkill   string        `json:"launcher_skill,omitempty"`
	LauncherPurpose string        `json:"launcher_purpose,omitempty"`
}

type Profile struct {
	Shape          Shape                 `json:"shape,omitempty"`
	MonorepoStack  MonorepoStack         `json:"monorepo_stack,omitempty"`
	WorkspaceHints []string              `json:"workspace_hints,omitempty"`
	ProcessHints   []string              `json:"process_hints,omitempty"`
	ServiceLaunch  ServiceLaunchContract `json:"service_launch,omitempty"`
}

type Info struct {
	Path    string
	Repo    string
	Branch  string
	Profile Profile
}

func Discover(ctx context.Context, runner environment.Runner, path string) (Info, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return Info{}, err
	}
	if _, err := runner.Run(ctx, absPath, "git", "rev-parse", "--is-inside-work-tree"); err != nil {
		return Info{}, fmt.Errorf("%s is not a git repository: %w", absPath, err)
	}

	remoteURL, err := runner.Run(ctx, absPath, "git", "remote", "get-url", "origin")
	if err != nil {
		return Info{}, fmt.Errorf("origin remote not found: %w", err)
	}
	repo, err := ParseGitHubRepo(strings.TrimSpace(remoteURL))
	if err != nil {
		return Info{}, err
	}

	branch := "main"
	if remoteHead, err := runner.Run(ctx, absPath, "git", "symbolic-ref", "--short", "refs/remotes/origin/HEAD"); err == nil {
		branch = strings.TrimPrefix(strings.TrimSpace(remoteHead), "origin/")
	} else if current, err := runner.Run(ctx, absPath, "git", "branch", "--show-current"); err == nil && strings.TrimSpace(current) != "" {
		branch = strings.TrimSpace(current)
	}

	return Info{
		Path:    absPath,
		Repo:    repo,
		Branch:  branch,
		Profile: Classify(absPath),
	}, nil
}

func Classify(path string) Profile {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	profile := Profile{
		Shape:         ShapeStandard,
		MonorepoStack: MonorepoStackNone,
		ServiceLaunch: ServiceLaunchContract{
			Required:        false,
			Scope:           "assigned_worktree",
			LauncherSkill:   "docker-compose-launch",
			LauncherPurpose: "local implementation/test dependencies only",
		},
	}

	switch {
	case fileExists(absPath, "turbo.json"):
		profile.Shape = ShapeMonorepo
		profile.MonorepoStack = MonorepoStackTurborepo
		profile.WorkspaceHints = []string{"Use Turbo workspace filters and task pipelines from the repo root."}
		profile.ProcessHints = []string{"Prefer `turbo run <task> --filter <workspace>` for targeted validation."}
	case fileExists(absPath, "nx.json"):
		profile.Shape = ShapeMonorepo
		profile.MonorepoStack = MonorepoStackNx
		profile.WorkspaceHints = []string{"Use Nx project names and targets from the repo root."}
		profile.ProcessHints = []string{"Prefer `nx <target> <project>` or the workspace package-manager wrapper."}
	case fileExists(absPath, "rush.json"):
		profile.Shape = ShapeMonorepo
		profile.MonorepoStack = MonorepoStackRush
		profile.WorkspaceHints = []string{"Use Rush-managed package names and commands from the repo root."}
		profile.ProcessHints = []string{"Prefer `rush build`, `rush test`, or the narrowest Rush command that matches the issue scope."}
	case fileExists(absPath, "WORKSPACE"), fileExists(absPath, "WORKSPACE.bazel"), fileExists(absPath, "MODULE.bazel"):
		profile.Shape = ShapeMonorepo
		profile.MonorepoStack = MonorepoStackBazel
		profile.WorkspaceHints = []string{"Use Bazel targets scoped to the affected packages."}
		profile.ProcessHints = []string{"Prefer `bazel test` or `bazel run` on the smallest affected target set."}
	case fileExists(absPath, "settings.gradle"), fileExists(absPath, "settings.gradle.kts"):
		profile.Shape = ShapeMonorepo
		profile.MonorepoStack = MonorepoStackGradle
		profile.WorkspaceHints = []string{"Use Gradle project paths and root tasks from the repo root."}
		profile.ProcessHints = []string{"Prefer `./gradlew <task>` with project selectors when possible."}
	case hasUnknownMonorepoMarkers(absPath):
		profile.Shape = ShapeMonorepo
		profile.MonorepoStack = MonorepoStackUnknown
		profile.WorkspaceHints = []string{"Inspect the workspace manifest to identify the affected package(s) before changing commands."}
		profile.ProcessHints = []string{"Prefer the narrowest repo-native workspace command once the affected package(s) are known."}
	}

	return profile
}

func ParseGitHubRepo(remote string) (string, error) {
	remote = strings.TrimSpace(remote)
	if remote == "" {
		return "", errors.New("empty remote URL")
	}

	if strings.HasPrefix(remote, "git@github.com:") {
		path := strings.TrimPrefix(remote, "git@github.com:")
		return normalizeGitHubPath(path)
	}

	if strings.HasPrefix(remote, "ssh://") || strings.HasPrefix(remote, "https://") || strings.HasPrefix(remote, "http://") {
		parsed, err := url.Parse(remote)
		if err != nil {
			return "", err
		}
		if !strings.EqualFold(parsed.Host, "github.com") {
			return "", fmt.Errorf("unsupported remote host %q", parsed.Host)
		}
		return normalizeGitHubPath(strings.TrimPrefix(parsed.Path, "/"))
	}

	return "", fmt.Errorf("unsupported remote format %q", remote)
}

func normalizeGitHubPath(path string) (string, error) {
	path = strings.TrimSuffix(path, ".git")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("invalid GitHub repo path %q", path)
	}
	return parts[0] + "/" + parts[1], nil
}

func fileExists(root string, name string) bool {
	_, err := os.Stat(filepath.Join(root, name))
	return err == nil
}

func hasUnknownMonorepoMarkers(root string) bool {
	if fileExists(root, "pnpm-workspace.yaml") || fileExists(root, "lerna.json") {
		return true
	}

	data, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		return false
	}

	var pkg map[string]any
	if err := json.Unmarshal(data, &pkg); err != nil {
		return false
	}
	_, hasWorkspaces := pkg["workspaces"]
	return hasWorkspaces
}
