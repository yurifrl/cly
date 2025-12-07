package module

import "embed"

// FS contains all blueprint templates embedded into the binary.
//
//go:embed base
var FS embed.FS
