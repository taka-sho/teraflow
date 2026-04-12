package skills

import "embed"

// FS contains built-in skill definitions used when skills/ is unavailable.
//
//go:embed *.yml
var FS embed.FS
