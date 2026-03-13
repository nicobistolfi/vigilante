package skillassets

import "embed"

// Skills contains built-in runtime skill files for installed binaries.
//
//go:embed skills/vigilante-issue-implementation skills/vigilante-issue-implementation-on-monorepo skills/vigilante-issue-implementation-on-turborepo skills/vigilante-conflict-resolution skills/vigilante-create-issue skills/vigilante-local-service-dependencies
var Skills embed.FS
