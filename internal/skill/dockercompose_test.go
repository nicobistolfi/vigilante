package skill

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nicobistolfi/vigilante/internal/testutil"
)

func TestSelectComposeCommandPrefersDockerPlugin(t *testing.T) {
	cmd, err := SelectComposeCommand(testutil.FakeRunner{
		LookPaths: map[string]string{
			"docker":         "/usr/bin/docker",
			"docker-compose": "/usr/local/bin/docker-compose",
		},
	}.LookPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(cmd.Slice(), " "); got != "docker compose" {
		t.Fatalf("unexpected compose command: %s", got)
	}
}

func TestSelectComposeCommandFallsBackToLegacyBinary(t *testing.T) {
	cmd, err := SelectComposeCommand(testutil.FakeRunner{
		LookPaths: map[string]string{
			"docker-compose": "/usr/local/bin/docker-compose",
		},
	}.LookPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(cmd.Slice(), " "); got != "docker-compose" {
		t.Fatalf("unexpected compose command: %s", got)
	}
}

func TestSelectComposeCommandFailsWithoutDocker(t *testing.T) {
	_, err := SelectComposeCommand(testutil.FakeRunner{}.LookPath)
	if err == nil || !strings.Contains(err.Error(), "neither docker nor docker-compose") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFindRepositoryComposeAssetPrefersKnownComposeNames(t *testing.T) {
	worktree := "/tmp/worktree"
	asset, ok := FindRepositoryComposeAsset(worktree, []string{
		filepath.Join(worktree, "ops", "docker-compose.yml"),
		filepath.Join(worktree, "compose.yaml"),
	})
	if !ok {
		t.Fatal("expected compose asset to be found")
	}
	if asset.FilePath != filepath.Join(worktree, "compose.yaml") {
		t.Fatalf("unexpected compose file: %s", asset.FilePath)
	}
	if asset.WorkingDir != worktree {
		t.Fatalf("unexpected working dir: %s", asset.WorkingDir)
	}
}

func TestBuildGeneratedComposePlanIncludesConnectionsAndCleanup(t *testing.T) {
	plan, err := BuildGeneratedComposePlan("/tmp/issue-66", []DatabaseService{DatabasePostgres, DatabaseMongoDB})
	if err != nil {
		t.Fatal(err)
	}
	if plan.ComposeFile != "/tmp/issue-66/.vigilante/docker-compose.launch.yml" {
		t.Fatalf("unexpected compose file: %s", plan.ComposeFile)
	}
	if len(plan.Services) != 2 {
		t.Fatalf("unexpected service count: %d", len(plan.Services))
	}
	if !strings.Contains(plan.ComposeYAML, "image: postgres:16") {
		t.Fatalf("compose yaml missing postgres image: %s", plan.ComposeYAML)
	}
	if !strings.Contains(plan.ComposeYAML, "image: mongo:7") {
		t.Fatalf("compose yaml missing mongo image: %s", plan.ComposeYAML)
	}
	if !strings.Contains(plan.CleanupExpectation, "down -v") {
		t.Fatalf("cleanup expectation missing down command: %s", plan.CleanupExpectation)
	}
	if plan.Connections[0].ConnectionURI == "" || plan.Connections[1].ConnectionURI == "" {
		t.Fatalf("expected connection URIs: %#v", plan.Connections)
	}
	if !strings.Contains(plan.ProjectName, "issue-66-") {
		t.Fatalf("unexpected project name: %s", plan.ProjectName)
	}
}

func TestBuildGeneratedComposePlanRejectsUnsupportedServices(t *testing.T) {
	_, err := BuildGeneratedComposePlan("/tmp/issue-66", []DatabaseService{"redis"})
	if err == nil {
		t.Fatal("expected unsupported service error")
	}
}

func TestBuildGeneratedComposePlanRequiresAtLeastOneService(t *testing.T) {
	_, err := BuildGeneratedComposePlan("/tmp/issue-66", nil)
	if err == nil || !strings.Contains(err.Error(), "at least one database service") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDockerComposeLaunchScriptReusesRepositoryComposeFile(t *testing.T) {
	worktree := t.TempDir()
	composePath := filepath.Join(worktree, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte("services:\n  postgres:\n    image: postgres:16\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := runLaunchScript(t, worktree, "postgres", fakeBinConfig{dockerPlugin: true})
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "\"compose_file_path\": \""+composePath+"\"") {
		t.Fatalf("expected repository compose file in output: %s", stdout)
	}
	if !strings.Contains(stdout, "\"launched_services\": [\"postgres\"]") {
		t.Fatalf("expected launched service in output: %s", stdout)
	}
}

func TestDockerComposeLaunchScriptGeneratesFallbackComposeFile(t *testing.T) {
	worktree := t.TempDir()

	stdout, stderr, err := runLaunchScript(t, worktree, "postgres,mongodb", fakeBinConfig{dockerPlugin: true})
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
	}

	composePath := filepath.Join(worktree, ".vigilante", "docker-compose.launch.yml")
	data, readErr := os.ReadFile(composePath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !strings.Contains(string(data), "image: postgres:16") || !strings.Contains(string(data), "image: mongo:7") {
		t.Fatalf("unexpected compose file contents: %s", string(data))
	}
	if !strings.Contains(stdout, "\"compose_file_path\": \""+composePath+"\"") {
		t.Fatalf("expected generated compose path in output: %s", stdout)
	}
}

func TestDockerComposeLaunchScriptFailsWithoutDocker(t *testing.T) {
	worktree := t.TempDir()
	_, stderr, err := runLaunchScript(t, worktree, "postgres", fakeBinConfig{})
	if err == nil {
		t.Fatal("expected missing docker failure")
	}
	if !strings.Contains(stderr, "docker compose unavailable") {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

func TestDockerComposeLaunchScriptFailsOnUnsupportedService(t *testing.T) {
	worktree := t.TempDir()
	_, stderr, err := runLaunchScript(t, worktree, "redis", fakeBinConfig{dockerPlugin: true})
	if err == nil {
		t.Fatal("expected unsupported service failure")
	}
	if !strings.Contains(stderr, "unsupported database service") {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

func TestDockerComposeLaunchScriptFailsOnPortConflict(t *testing.T) {
	worktree := t.TempDir()
	_, stderr, err := runLaunchScript(t, worktree, "postgres", fakeBinConfig{
		dockerPlugin:       true,
		conflictAllLsofTCP: true,
	})
	if err == nil {
		t.Fatal("expected port conflict failure")
	}
	if !strings.Contains(stderr, "already in use") {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

type fakeBinConfig struct {
	dockerPlugin       bool
	legacyCompose      bool
	conflictAllLsofTCP bool
}

const testSystemPath = "/usr/bin:/bin:/usr/sbin:/sbin"

func runLaunchScript(t *testing.T, worktree string, services string, cfg fakeBinConfig) (string, string, error) {
	t.Helper()

	binDir := t.TempDir()
	if cfg.dockerPlugin {
		writeExecutable(t, filepath.Join(binDir, "docker"), "#!/bin/bash\nset -euo pipefail\nif [ \"${1-}\" = \"compose\" ] && [ \"${2-}\" = \"version\" ]; then\n  exit 0\nfi\nexit 0\n")
	}
	if cfg.legacyCompose {
		writeExecutable(t, filepath.Join(binDir, "docker-compose"), "#!/bin/bash\nexit 0\n")
	}
	if cfg.conflictAllLsofTCP {
		writeExecutable(t, filepath.Join(binDir, "lsof"), "#!/bin/bash\ncase \"$*\" in\n  *TCP:*) exit 0 ;;\n  *) exit 1 ;;\nesac\n")
	}

	scriptPath := filepath.Join("..", "..", "skills", "docker-compose-launch", "scripts", "launch.sh")
	cmd := exec.Command("/bin/bash", scriptPath, "--worktree", worktree, "--services", services, "--dry-run")
	cmd.Env = append(os.Environ(), "PATH="+binDir+string(os.PathListSeparator)+testSystemPath)
	output, err := cmd.CombinedOutput()
	text := string(output)
	if err != nil {
		return "", text, err
	}
	return text, "", nil
}

func writeExecutable(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}
