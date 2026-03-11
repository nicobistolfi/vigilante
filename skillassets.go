package skillassets

import "embed"

// Skills contains built-in runtime skill files for installed binaries.
//
//go:embed skills/vigilante-issue-implementation skills/vigilante-conflict-resolution
var Skills embed.FS
