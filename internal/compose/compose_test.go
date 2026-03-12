package compose

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSelectComposeCommandPrefersDockerCompose(t *testing.T) {
	command, err := SelectComposeCommand(func(name string, args ...string) error {
		if name == "docker" && reflect.DeepEqual(args, []string{"compose", "version"}) {
			return nil
		}
		return errors.New("missing")
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(command, []string{"docker", "compose"}) {
		t.Fatalf("unexpected command: %#v", command)
	}
}

func TestPlanLaunchReusesRepositoryComposeFile(t *testing.T) {
	worktree := t.TempDir()
	composePath := filepath.Join(worktree, "docker-compose.yml")
	body := "services:\n  postgres:\n    image: postgres:16\n    ports:\n      - \"15432:5432\"\n"
	if err := os.WriteFile(composePath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	plan, err := PlanLaunch(PlanOptions{
		WorktreePath: worktree,
		Services:     []DatabaseService{Postgres},
		Lookup: func(name string, args ...string) error {
			if name == "docker" {
				return nil
			}
			return errors.New("missing")
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Mode != ModeRepository {
		t.Fatalf("expected repository mode, got %s", plan.Mode)
	}
	if plan.ComposeFile != composePath {
		t.Fatalf("unexpected compose file: %s", plan.ComposeFile)
	}
	if got := plan.Services[0].ConnectionString; got != "postgres://app:app@127.0.0.1:15432/app?sslmode=disable" {
		t.Fatalf("unexpected connection string: %s", got)
	}
}

func TestPlanLaunchGeneratesFallbackConfig(t *testing.T) {
	worktree := t.TempDir()
	plan, err := PlanLaunch(PlanOptions{
		WorktreePath: worktree,
		Services:     []DatabaseService{MySQL, MongoDB},
		Lookup: func(name string, args ...string) error {
			if name == "docker-compose" {
				return nil
			}
			return errors.New("missing")
		},
		PortAvailable: func(port int) bool { return true },
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Mode != ModeGenerated {
		t.Fatalf("expected generated mode, got %s", plan.Mode)
	}
	if !strings.HasSuffix(plan.ComposeFile, filepath.Join(".vigilante", "docker-compose-launch.yml")) {
		t.Fatalf("unexpected compose path: %s", plan.ComposeFile)
	}
	if !strings.Contains(plan.GeneratedConfig, "image: mysql:8.4") || !strings.Contains(plan.GeneratedConfig, "image: mongo:7") {
		t.Fatalf("generated compose missing expected images:\n%s", plan.GeneratedConfig)
	}
	if err := WriteGeneratedCompose(plan); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(plan.ComposeFile); err != nil {
		t.Fatalf("expected generated compose file: %v", err)
	}
}

func TestPlanLaunchFailsForUnsupportedService(t *testing.T) {
	_, err := PlanLaunch(PlanOptions{
		WorktreePath: t.TempDir(),
		Services:     []DatabaseService{"redis"},
		Lookup:       func(name string, args ...string) error { return nil },
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported database service") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPlanLaunchFailsWithoutComposeCommand(t *testing.T) {
	_, err := PlanLaunch(PlanOptions{
		WorktreePath: t.TempDir(),
		Services:     []DatabaseService{Postgres},
		Lookup: func(name string, args ...string) error {
			return errors.New("missing")
		},
	})
	if !errors.Is(err, errNoComposeCommand) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPlanLaunchFailsWhenFallbackPortIsBusy(t *testing.T) {
	_, err := PlanLaunch(PlanOptions{
		WorktreePath: t.TempDir(),
		Services:     []DatabaseService{MariaDB},
		Lookup: func(name string, args ...string) error {
			if name == "docker" {
				return nil
			}
			return errors.New("missing")
		},
		PortAvailable: func(port int) bool { return false },
	})
	if err == nil || !strings.Contains(err.Error(), "already in use") {
		t.Fatalf("unexpected error: %v", err)
	}
}
