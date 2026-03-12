package skillassets

import "embed"

// Skills contains built-in runtime skill files for installed binaries.
//
//go:embed skills/vigilante-issue-implementation skills/vigilante-conflict-resolution skills/vigilante-create-issue skills/docker-compose-launch
var Skills embed.FS
