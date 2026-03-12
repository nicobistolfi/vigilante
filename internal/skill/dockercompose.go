package skill

import (
	"fmt"
	"hash/crc32"
	"path/filepath"
	"strings"
)

type DatabaseService string

const (
	DatabaseMySQL    DatabaseService = "mysql"
	DatabaseMariaDB  DatabaseService = "mariadb"
	DatabasePostgres DatabaseService = "postgres"
	DatabaseMongoDB  DatabaseService = "mongodb"
)

type ComposeCommand struct {
	Command string
	Args    []string
}

func (c ComposeCommand) Slice() []string {
	return append([]string{c.Command}, c.Args...)
}

type RepositoryComposeAsset struct {
	WorkingDir string
	FilePath   string
}

type ConnectionInfo struct {
	Service       DatabaseService
	Host          string
	Port          int
	Database      string
	Username      string
	Password      string
	ConnectionURI string
}

type GeneratedComposePlan struct {
	WorkingDir         string
	ComposeFile        string
	ProjectName        string
	Services           []string
	Connections        []ConnectionInfo
	CleanupExpectation string
	ComposeYAML        string
}

type lookPathFunc func(string) (string, error)

func SelectComposeCommand(lookPath lookPathFunc) (ComposeCommand, error) {
	if _, err := lookPath("docker"); err == nil {
		return ComposeCommand{Command: "docker", Args: []string{"compose"}}, nil
	}
	if _, err := lookPath("docker-compose"); err == nil {
		return ComposeCommand{Command: "docker-compose"}, nil
	}
	return ComposeCommand{}, fmt.Errorf("docker compose unavailable: neither docker nor docker-compose was found in PATH")
}

func FindRepositoryComposeAsset(worktree string, files []string) (RepositoryComposeAsset, bool) {
	candidates := []string{
		"compose.yaml",
		"compose.yml",
		"docker-compose.yaml",
		"docker-compose.yml",
	}
	for _, candidate := range candidates {
		for _, file := range files {
			if filepath.Base(file) != candidate {
				continue
			}
			return RepositoryComposeAsset{
				WorkingDir: filepath.Dir(file),
				FilePath:   file,
			}, true
		}
	}
	return RepositoryComposeAsset{}, false
}

func BuildGeneratedComposePlan(worktree string, services []DatabaseService) (GeneratedComposePlan, error) {
	if len(services) == 0 {
		return GeneratedComposePlan{}, fmt.Errorf("at least one database service is required")
	}

	projectName := projectNameForWorktree(worktree)
	composeFile := filepath.Join(worktree, ".vigilante", "docker-compose.launch.yml")
	plan := GeneratedComposePlan{
		WorkingDir:         worktree,
		ComposeFile:        composeFile,
		ProjectName:        projectName,
		CleanupExpectation: "Run `docker compose down -v` or `docker-compose down -v` from the reported working directory and compose file when the worktree session is finished.",
	}

	sections := []string{"services:"}
	for _, service := range services {
		def, err := composeDefinition(service, worktree)
		if err != nil {
			return GeneratedComposePlan{}, err
		}
		plan.Services = append(plan.Services, def.Name)
		plan.Connections = append(plan.Connections, def.Connection)
		sections = append(sections, def.YAML...)
	}
	plan.ComposeYAML = strings.Join(sections, "\n") + "\n"
	return plan, nil
}

type composeServiceDefinition struct {
	Name       string
	Connection ConnectionInfo
	YAML       []string
}

func composeDefinition(service DatabaseService, worktree string) (composeServiceDefinition, error) {
	offset := int(crc32.ChecksumIEEE([]byte(worktree+"-"+string(service))) % 200)

	switch service {
	case DatabaseMySQL:
		port := 23306 + offset
		return composeServiceDefinition{
			Name: "mysql",
			Connection: ConnectionInfo{
				Service:       service,
				Host:          "127.0.0.1",
				Port:          port,
				Database:      "app",
				Username:      "app",
				Password:      "app",
				ConnectionURI: fmt.Sprintf("mysql://app:app@127.0.0.1:%d/app", port),
			},
			YAML: []string{
				"  mysql:",
				"    image: mysql:8.4",
				"    environment:",
				"      MYSQL_DATABASE: app",
				"      MYSQL_USER: app",
				"      MYSQL_PASSWORD: app",
				"      MYSQL_ROOT_PASSWORD: root",
				"    ports:",
				fmt.Sprintf("      - \"%d:3306\"", port),
			},
		}, nil
	case DatabaseMariaDB:
		port := 23307 + offset
		return composeServiceDefinition{
			Name: "mariadb",
			Connection: ConnectionInfo{
				Service:       service,
				Host:          "127.0.0.1",
				Port:          port,
				Database:      "app",
				Username:      "app",
				Password:      "app",
				ConnectionURI: fmt.Sprintf("mysql://app:app@127.0.0.1:%d/app", port),
			},
			YAML: []string{
				"  mariadb:",
				"    image: mariadb:11",
				"    environment:",
				"      MARIADB_DATABASE: app",
				"      MARIADB_USER: app",
				"      MARIADB_PASSWORD: app",
				"      MARIADB_ROOT_PASSWORD: root",
				"    ports:",
				fmt.Sprintf("      - \"%d:3306\"", port),
			},
		}, nil
	case DatabasePostgres:
		port := 25432 + offset
		return composeServiceDefinition{
			Name: "postgres",
			Connection: ConnectionInfo{
				Service:       service,
				Host:          "127.0.0.1",
				Port:          port,
				Database:      "app",
				Username:      "app",
				Password:      "app",
				ConnectionURI: fmt.Sprintf("postgres://app:app@127.0.0.1:%d/app?sslmode=disable", port),
			},
			YAML: []string{
				"  postgres:",
				"    image: postgres:16",
				"    environment:",
				"      POSTGRES_DB: app",
				"      POSTGRES_USER: app",
				"      POSTGRES_PASSWORD: app",
				"    ports:",
				fmt.Sprintf("      - \"%d:5432\"", port),
			},
		}, nil
	case DatabaseMongoDB:
		port := 27018 + offset
		return composeServiceDefinition{
			Name: "mongodb",
			Connection: ConnectionInfo{
				Service:       service,
				Host:          "127.0.0.1",
				Port:          port,
				Database:      "app",
				ConnectionURI: fmt.Sprintf("mongodb://127.0.0.1:%d/app", port),
			},
			YAML: []string{
				"  mongodb:",
				"    image: mongo:7",
				"    ports:",
				fmt.Sprintf("      - \"%d:27017\"", port),
			},
		}, nil
	default:
		return composeServiceDefinition{}, fmt.Errorf("unsupported database service %q", service)
	}
}

func projectNameForWorktree(worktree string) string {
	base := strings.ToLower(filepath.Base(worktree))
	base = strings.ReplaceAll(base, "_", "-")
	base = strings.ReplaceAll(base, " ", "-")
	base = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '-':
			return r
		default:
			return '-'
		}
	}, base)
	base = strings.Trim(base, "-")
	if base == "" {
		base = "worktree"
	}
	suffix := fmt.Sprintf("%08x", crc32.ChecksumIEEE([]byte(worktree)))
	return base + "-" + suffix[:8]
}
