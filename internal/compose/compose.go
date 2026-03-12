package compose

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type DatabaseService string

const (
	MySQL    DatabaseService = "mysql"
	MariaDB  DatabaseService = "mariadb"
	Postgres DatabaseService = "postgres"
	MongoDB  DatabaseService = "mongodb"
)

type PortCheck func(port int) bool
type LookupCommand func(name string, args ...string) error

type PlanMode string

const (
	ModeRepository PlanMode = "repository"
	ModeGenerated  PlanMode = "generated"
)

type Plan struct {
	Mode            PlanMode
	Command         []string
	WorkingDir      string
	ComposeFile     string
	ProjectName     string
	Services        []RuntimeService
	CleanupCommand  []string
	GeneratedConfig string
}

type RuntimeService struct {
	Kind             DatabaseService
	ServiceName      string
	Host             string
	HostPort         int
	ContainerPort    int
	ConnectionString string
}

type ComposeService struct {
	Name  string
	Image string
	Ports []PortBinding
}

type PortBinding struct {
	HostPort      int
	ContainerPort int
}

type PlanOptions struct {
	WorktreePath  string
	Services      []DatabaseService
	Lookup        LookupCommand
	PortAvailable PortCheck
}

var (
	errNoComposeCommand = errors.New("docker compose is not available")
	errNoServices       = errors.New("at least one database service is required")
)

func SelectComposeCommand(lookup LookupCommand) ([]string, error) {
	if lookup == nil {
		lookup = systemLookup
	}
	if err := lookup("docker", "compose", "version"); err == nil {
		return []string{"docker", "compose"}, nil
	}
	if err := lookup("docker-compose", "version"); err == nil {
		return []string{"docker-compose"}, nil
	}
	return nil, errNoComposeCommand
}

func PlanLaunch(opts PlanOptions) (Plan, error) {
	services, err := normalizeServices(opts.Services)
	if err != nil {
		return Plan{}, err
	}
	command, err := SelectComposeCommand(opts.Lookup)
	if err != nil {
		return Plan{}, err
	}
	worktreePath, err := filepath.Abs(opts.WorktreePath)
	if err != nil {
		return Plan{}, err
	}

	if reusable, ok, err := findRepositoryPlan(worktreePath, command, services); err != nil {
		return Plan{}, err
	} else if ok {
		return reusable, nil
	}

	projectName := projectNameFor(worktreePath)
	runtimeServices := make([]RuntimeService, 0, len(services))
	for _, service := range services {
		hostPort := preferredHostPort(projectName, service)
		if opts.PortAvailable != nil && !opts.PortAvailable(hostPort) {
			return Plan{}, fmt.Errorf("port %d is already in use for %s", hostPort, service)
		}
		runtimeServices = append(runtimeServices, RuntimeService{
			Kind:             service,
			ServiceName:      string(service),
			Host:             "127.0.0.1",
			HostPort:         hostPort,
			ContainerPort:    defaultContainerPort(service),
			ConnectionString: connectionString(service, hostPort),
		})
	}

	composeFile := filepath.Join(worktreePath, ".vigilante", "docker-compose-launch.yml")
	return Plan{
		Mode:            ModeGenerated,
		Command:         command,
		WorkingDir:      filepath.Dir(composeFile),
		ComposeFile:     composeFile,
		ProjectName:     projectName,
		Services:        runtimeServices,
		CleanupCommand:  cleanupCommand(command, composeFile, projectName),
		GeneratedConfig: RenderGeneratedCompose(projectName, runtimeServices),
	}, nil
}

func RenderGeneratedCompose(projectName string, services []RuntimeService) string {
	var b strings.Builder
	b.WriteString("services:\n")
	for _, service := range services {
		image := defaultImage(service.Kind)
		volume := fmt.Sprintf("%s-%s-data", projectName, service.ServiceName)

		fmt.Fprintf(&b, "  %s:\n", service.ServiceName)
		fmt.Fprintf(&b, "    image: %s\n", image)
		fmt.Fprintf(&b, "    container_name: %s-%s\n", projectName, service.ServiceName)
		if env := defaultEnvironment(service.Kind); len(env) > 0 {
			b.WriteString("    environment:\n")
			keys := make([]string, 0, len(env))
			for key := range env {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				fmt.Fprintf(&b, "      %s: %s\n", key, env[key])
			}
		}
		fmt.Fprintf(&b, "    ports:\n      - \"%d:%d\"\n", service.HostPort, service.ContainerPort)
		fmt.Fprintf(&b, "    volumes:\n      - %s:%s\n", volume, defaultDataDir(service.Kind))
	}
	b.WriteString("volumes:\n")
	for _, service := range services {
		fmt.Fprintf(&b, "  %s-%s-data:\n", projectName, service.ServiceName)
	}
	return b.String()
}

func WriteGeneratedCompose(plan Plan) error {
	if plan.Mode != ModeGenerated {
		return nil
	}
	if err := os.MkdirAll(plan.WorkingDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(plan.ComposeFile, []byte(plan.GeneratedConfig), 0o644)
}

func ParseComposeServices(path string) ([]ComposeService, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	var services []ComposeService
	var current *ComposeService
	inServices := false
	inPorts := false

	for _, raw := range lines {
		line := strings.TrimRight(raw, " \t")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		indent := len(line) - len(strings.TrimLeft(line, " "))
		switch {
		case indent == 0 && trimmed == "services:":
			inServices = true
			inPorts = false
			current = nil
		case inServices && indent == 0:
			inServices = false
			inPorts = false
			current = nil
		case inServices && indent == 2 && strings.HasSuffix(trimmed, ":"):
			name := strings.TrimSuffix(trimmed, ":")
			services = append(services, ComposeService{Name: name})
			current = &services[len(services)-1]
			inPorts = false
		case current != nil && indent >= 4 && strings.HasPrefix(trimmed, "image:"):
			current.Image = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "image:")), "\"'")
			inPorts = false
		case current != nil && indent >= 4 && trimmed == "ports:":
			inPorts = true
		case current != nil && inPorts && indent >= 6 && strings.HasPrefix(trimmed, "- "):
			if binding, ok := parsePortBinding(strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))); ok {
				current.Ports = append(current.Ports, binding)
			}
		case current != nil && indent <= 4:
			inPorts = false
		}
	}

	return services, nil
}

func systemLookup(name string, args ...string) error {
	_, err := exec.LookPath(name)
	return err
}

func findRepositoryPlan(worktreePath string, command []string, requested []DatabaseService) (Plan, bool, error) {
	files, err := collectComposeCandidates(worktreePath)
	if err != nil {
		return Plan{}, false, err
	}

	projectName := projectNameFor(worktreePath)
	for _, path := range files {
		services, err := ParseComposeServices(path)
		if err != nil {
			return Plan{}, false, err
		}
		runtimeServices, ok := matchRequestedServices(services, requested)
		if !ok {
			continue
		}
		return Plan{
			Mode:           ModeRepository,
			Command:        command,
			WorkingDir:     filepath.Dir(path),
			ComposeFile:    path,
			ProjectName:    projectName,
			Services:       runtimeServices,
			CleanupCommand: cleanupCommand(command, path, projectName),
		}, true, nil
	}
	return Plan{}, false, nil
}

func collectComposeCandidates(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		name := strings.ToLower(filepath.Base(path))
		if strings.Contains(name, "compose") && (strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml")) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func matchRequestedServices(composeServices []ComposeService, requested []DatabaseService) ([]RuntimeService, bool) {
	runtimeServices := make([]RuntimeService, 0, len(requested))
	for _, requestedService := range requested {
		match, ok := findMatchingService(composeServices, requestedService)
		if !ok {
			return nil, false
		}
		runtimeServices = append(runtimeServices, match)
	}
	return runtimeServices, true
}

func findMatchingService(composeServices []ComposeService, requested DatabaseService) (RuntimeService, bool) {
	for _, candidate := range composeServices {
		if serviceKindFor(candidate) != requested {
			continue
		}
		for _, binding := range candidate.Ports {
			if binding.HostPort == 0 || binding.ContainerPort != defaultContainerPort(requested) {
				continue
			}
			return RuntimeService{
				Kind:             requested,
				ServiceName:      candidate.Name,
				Host:             "127.0.0.1",
				HostPort:         binding.HostPort,
				ContainerPort:    binding.ContainerPort,
				ConnectionString: connectionString(requested, binding.HostPort),
			}, true
		}
	}
	return RuntimeService{}, false
}

func normalizeServices(services []DatabaseService) ([]DatabaseService, error) {
	if len(services) == 0 {
		return nil, errNoServices
	}

	seen := make(map[DatabaseService]struct{}, len(services))
	normalized := make([]DatabaseService, 0, len(services))
	for _, service := range services {
		switch service {
		case MySQL, MariaDB, Postgres, MongoDB:
		default:
			return nil, fmt.Errorf("unsupported database service %q", service)
		}
		if _, ok := seen[service]; ok {
			continue
		}
		seen[service] = struct{}{}
		normalized = append(normalized, service)
	}
	sort.Slice(normalized, func(i, j int) bool { return normalized[i] < normalized[j] })
	return normalized, nil
}

func projectNameFor(worktreePath string) string {
	sum := sha1.Sum([]byte(worktreePath))
	return "vig-" + hex.EncodeToString(sum[:])[:8]
}

func preferredHostPort(projectName string, service DatabaseService) int {
	base := map[DatabaseService]int{
		MySQL:    33306,
		MariaDB:  34306,
		Postgres: 35432,
		MongoDB:  37017,
	}[service]
	sum := sha1.Sum([]byte(projectName + ":" + string(service)))
	offset := int(sum[0])%50 + int(sum[1])%10
	return base + offset
}

func defaultContainerPort(service DatabaseService) int {
	switch service {
	case MySQL, MariaDB:
		return 3306
	case Postgres:
		return 5432
	case MongoDB:
		return 27017
	default:
		return 0
	}
}

func defaultImage(service DatabaseService) string {
	switch service {
	case MySQL:
		return "mysql:8.4"
	case MariaDB:
		return "mariadb:11"
	case Postgres:
		return "postgres:16"
	case MongoDB:
		return "mongo:7"
	default:
		return ""
	}
}

func defaultEnvironment(service DatabaseService) map[string]string {
	switch service {
	case MySQL:
		return map[string]string{
			"MYSQL_DATABASE":      "app",
			"MYSQL_PASSWORD":      "app",
			"MYSQL_ROOT_PASSWORD": "root",
			"MYSQL_USER":          "app",
		}
	case MariaDB:
		return map[string]string{
			"MARIADB_DATABASE":      "app",
			"MARIADB_PASSWORD":      "app",
			"MARIADB_ROOT_PASSWORD": "root",
			"MARIADB_USER":          "app",
		}
	case Postgres:
		return map[string]string{
			"POSTGRES_DB":       "app",
			"POSTGRES_PASSWORD": "app",
			"POSTGRES_USER":     "app",
		}
	case MongoDB:
		return map[string]string{
			"MONGO_INITDB_ROOT_PASSWORD": "app",
			"MONGO_INITDB_ROOT_USERNAME": "app",
		}
	default:
		return nil
	}
}

func defaultDataDir(service DatabaseService) string {
	switch service {
	case MongoDB:
		return "/data/db"
	case Postgres:
		return "/var/lib/postgresql/data"
	default:
		return "/var/lib/" + string(service)
	}
}

func cleanupCommand(command []string, composeFile, projectName string) []string {
	result := append([]string{}, command...)
	result = append(result, "-f", composeFile, "-p", projectName, "down", "--volumes", "--remove-orphans")
	return result
}

func serviceKindFor(service ComposeService) DatabaseService {
	candidate := strings.ToLower(service.Name + " " + service.Image)
	switch {
	case strings.Contains(candidate, "mariadb"):
		return MariaDB
	case strings.Contains(candidate, "mysql"):
		return MySQL
	case strings.Contains(candidate, "postgres"):
		return Postgres
	case strings.Contains(candidate, "mongo"):
		return MongoDB
	default:
		return ""
	}
}

func connectionString(service DatabaseService, port int) string {
	switch service {
	case MySQL, MariaDB:
		return fmt.Sprintf("mysql://app:app@tcp(127.0.0.1:%d)/app", port)
	case Postgres:
		return fmt.Sprintf("postgres://app:app@127.0.0.1:%d/app?sslmode=disable", port)
	case MongoDB:
		return fmt.Sprintf("mongodb://app:app@127.0.0.1:%d/admin", port)
	default:
		return ""
	}
}

func parsePortBinding(raw string) (PortBinding, bool) {
	raw = strings.Trim(raw, "\"'")
	parts := strings.Split(raw, ":")
	if len(parts) < 2 {
		return PortBinding{}, false
	}
	hostPart := strings.TrimSpace(parts[len(parts)-2])
	containerPart := strings.TrimSpace(parts[len(parts)-1])
	hostPort, err := strconv.Atoi(hostPart)
	if err != nil {
		return PortBinding{}, false
	}
	containerPort, err := strconv.Atoi(containerPart)
	if err != nil {
		return PortBinding{}, false
	}
	return PortBinding{HostPort: hostPort, ContainerPort: containerPort}, true
}
