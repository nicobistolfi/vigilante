package skillassets

import "embed"

// Skills contains built-in runtime skill files for installed binaries.
//
//go:embed skills/vigilante-issue-implementation skills/vigilante-conflict-resolution skills/vigilante-create-issue skills/vigilante-turborepo-issue-implementation skills/vigilante-nx-issue-implementation skills/vigilante-rush-issue-implementation skills/vigilante-bazel-issue-implementation skills/vigilante-gradle-issue-implementation
var Skills embed.FS
